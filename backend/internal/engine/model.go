package engine

// FlagType distinguishes boolean flags from multi-variant flags.
type FlagType string

const (
	FlagTypeBoolean FlagType = "boolean"
	FlagTypeVariant FlagType = "variant"
)

// FlagSemantics controls how evaluation results are determined and cached.
type FlagSemantics string

const (
	SemanticsDeterministic FlagSemantics = "deterministic"
	SemanticsRandom        FlagSemantics = "random"
	SemanticsPersistent    FlagSemantics = "persistent"
)

// Operator enumerates the supported condition operators.
type Operator string

const (
	OpEquals      Operator = "equals"
	OpNotEquals   Operator = "not_equals"
	OpIn          Operator = "in"
	OpNotIn       Operator = "not_in"
	OpContains    Operator = "contains"
	OpStartsWith  Operator = "starts_with"
	OpEndsWith    Operator = "ends_with"
	OpGreaterThan Operator = "greater_than"
	OpLessThan    Operator = "less_than"
	OpRegex       Operator = "regex"
	OpSemverMatch Operator = "semver_match"
)

// Reason describes why a particular evaluation result was returned.
type Reason string

const (
	ReasonRuleMatch     Reason = "rule_match"
	ReasonDefault       Reason = "default"
	ReasonPersisted     Reason = "persisted"
	ReasonNoPersistence Reason = "no_persistence"
	ReasonDisabled      Reason = "disabled"
	ReasonNotFound      Reason = "not_found"
	ReasonError         Reason = "error"
)

// EvaluationContext carries all inputs needed to evaluate a flag.
type EvaluationContext struct {
	TenantID    string
	SubjectID   string
	Environment string
	Attributes  map[string]any
}

// EvaluationResult is the output of evaluating a single flag.
type EvaluationResult struct {
	TenantID string `json:"tenantId,omitempty"`
	FlagKey  string `json:"flagKey"`
	Enabled  bool   `json:"enabled"`
	Variant  string `json:"variant"`
	Reason   Reason `json:"reason"`
}

// Condition tests a single attribute from the evaluation context.
type Condition struct {
	Attribute string   `json:"attribute"`
	Operator  Operator `json:"operator"`
	Value     any      `json:"value"`
}

// Rule is an ordered entry in a flag's rule list.
type Rule struct {
	Conditions        []Condition `json:"conditions"`
	RolloutPercentage int         `json:"rolloutPercentage"`
	Variant           string      `json:"variant"`
}

// EvalResult is the outcome embedded in a flag definition (default result).
type EvalResult struct {
	Enabled bool   `json:"enabled"`
	Variant string `json:"variant"`
}

// FlagDefinition is the full specification of a feature flag.
type FlagDefinition struct {
	Key           string        `json:"key"`
	Type          FlagType      `json:"type"`
	Semantics     FlagSemantics `json:"semantics"`
	Enabled       bool          `json:"enabled"`
	Description   string        `json:"description"`
	Rules         []Rule        `json:"rules"`
	DefaultResult EvalResult    `json:"defaultResult"`
}
