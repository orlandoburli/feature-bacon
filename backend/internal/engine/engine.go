package engine

import (
	"math/rand/v2"
	"time"
)

// FlagStore abstracts retrieval of flag definitions.
// Implemented by config file loader, gRPC persistence client, etc.
type FlagStore interface {
	GetFlag(tenantID, flagKey string) (*FlagDefinition, error)
	ListFlagKeys(tenantID string) ([]string, error)
}

// Assignment represents a persisted flag assignment for a subject.
type Assignment struct {
	SubjectID  string
	FlagKey    string
	Enabled    bool
	Variant    string
	AssignedAt time.Time
	ExpiresAt  time.Time // zero value means no expiry
}

// AssignmentStore abstracts persistent assignment read/write operations.
// Nil when running in config-file-only mode.
type AssignmentStore interface {
	GetAssignment(tenantID, subjectID, flagKey string) (*Assignment, bool, error)
	SaveAssignment(tenantID string, a *Assignment) error
}

// Engine evaluates feature flags against an evaluation context.
type Engine struct {
	store       FlagStore
	assignments AssignmentStore
}

// New creates an Engine backed by the given FlagStore and optional AssignmentStore.
func New(store FlagStore, assignments AssignmentStore) *Engine {
	return &Engine{store: store, assignments: assignments}
}

// Store returns the underlying FlagStore.
func (e *Engine) Store() FlagStore {
	return e.store
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

	if flag.Semantics == SemanticsPersistent {
		return e.evaluatePersistent(flag, ctx)
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

// evaluatePersistent checks for a saved assignment first. If none exists or it
// has expired, evaluates rules normally and persists the result. If no
// AssignmentStore is configured, falls back to deterministic evaluation.
func (e *Engine) evaluatePersistent(flag *FlagDefinition, ctx EvaluationContext) EvaluationResult {
	if e.assignments == nil {
		result := e.evaluateRules(flag, ctx)
		result.Reason = ReasonNoPersistence
		return result
	}

	existing, found, err := e.assignments.GetAssignment(ctx.TenantID, ctx.SubjectID, flag.Key)
	if err == nil && found {
		return EvaluationResult{
			TenantID: ctx.TenantID,
			FlagKey:  flag.Key,
			Enabled:  existing.Enabled,
			Variant:  existing.Variant,
			Reason:   ReasonPersisted,
		}
	}

	result := e.evaluateRules(flag, ctx)

	_ = e.assignments.SaveAssignment(ctx.TenantID, &Assignment{
		SubjectID:  ctx.SubjectID,
		FlagKey:    flag.Key,
		Enabled:    result.Enabled,
		Variant:    result.Variant,
		AssignedAt: time.Now(),
	})

	return result
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
