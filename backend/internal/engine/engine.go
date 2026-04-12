package engine

import (
	"math/rand/v2"
)

// FlagStore abstracts retrieval of flag definitions.
// Implemented by config file loader, gRPC persistence client, etc.
type FlagStore interface {
	GetFlag(tenantID, flagKey string) (*FlagDefinition, error)
	ListFlagKeys(tenantID string) ([]string, error)
}

// Engine evaluates feature flags against an evaluation context.
type Engine struct {
	store FlagStore
}

// New creates an Engine backed by the given FlagStore.
func New(store FlagStore) *Engine {
	return &Engine{store: store}
}

// Evaluate processes a single flag and returns the result.
func (e *Engine) Evaluate(flagKey string, ctx EvaluationContext) EvaluationResult {
	flag, err := e.store.GetFlag(ctx.TenantID, flagKey)
	if err != nil {
		return EvaluationResult{
			TenantID: ctx.TenantID,
			FlagKey:  flagKey,
			Enabled:  false,
			Reason:   ReasonError,
		}
	}
	if flag == nil {
		return EvaluationResult{
			TenantID: ctx.TenantID,
			FlagKey:  flagKey,
			Enabled:  false,
			Reason:   ReasonNotFound,
		}
	}

	if !flag.Enabled {
		return EvaluationResult{
			TenantID: ctx.TenantID,
			FlagKey:  flagKey,
			Enabled:  false,
			Reason:   ReasonDisabled,
		}
	}

	return e.evaluateRules(flag, ctx)
}

// EvaluateBatch processes multiple flags and returns all results.
// Each flag is evaluated independently; a failure on one does not affect others.
func (e *Engine) EvaluateBatch(flagKeys []string, ctx EvaluationContext) []EvaluationResult {
	results := make([]EvaluationResult, len(flagKeys))
	for i, key := range flagKeys {
		results[i] = e.Evaluate(key, ctx)
	}
	return results
}

// evaluateRules walks rules top-to-bottom (first match wins).
func (e *Engine) evaluateRules(flag *FlagDefinition, ctx EvaluationContext) EvaluationResult {
	for _, rule := range flag.Rules {
		if !AllConditionsMatch(rule.Conditions, ctx) {
			continue
		}

		if e.subjectInRollout(flag, ctx, rule.RolloutPercentage) {
			return EvaluationResult{
				TenantID: ctx.TenantID,
				FlagKey:  flag.Key,
				Enabled:  true,
				Variant:  rule.Variant,
				Reason:   ReasonRuleMatch,
			}
		}
	}

	return EvaluationResult{
		TenantID: ctx.TenantID,
		FlagKey:  flag.Key,
		Enabled:  flag.DefaultResult.Enabled,
		Variant:  flag.DefaultResult.Variant,
		Reason:   ReasonDefault,
	}
}

// subjectInRollout checks whether the subject falls within the rollout percentage
// using deterministic bucketing for deterministic/persistent flags or random for random flags.
func (e *Engine) subjectInRollout(flag *FlagDefinition, ctx EvaluationContext, rolloutPercentage int) bool {
	switch flag.Semantics {
	case SemanticsRandom:
		if rolloutPercentage <= 0 {
			return false
		}
		if rolloutPercentage >= 100 {
			return true
		}
		return rand.IntN(100) < rolloutPercentage
	default:
		return InRollout(ctx.TenantID, flag.Key, ctx.SubjectID, rolloutPercentage)
	}
}
