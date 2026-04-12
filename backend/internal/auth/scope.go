package auth

type Scope string

const (
	ScopeEvaluation Scope = "evaluation"
	ScopeManagement Scope = "management"
)

func (s Scope) Valid() bool {
	return s == ScopeEvaluation || s == ScopeManagement
}
