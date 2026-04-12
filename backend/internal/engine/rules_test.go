package engine

import (
	"testing"
)

const (
	attrCountry = "attributes.country"
	attrVal     = "attributes.val"
)

func baseCtx() EvaluationContext {
	return EvaluationContext{
		TenantID:    "acme",
		SubjectID:   "user_123",
		Environment: "production",
		Attributes: map[string]any{
			"country":     "BR",
			"plan":        "premium",
			"email":       "alice@acme.com",
			"path":        "/v2/api/test",
			"hostname":    "node1.internal",
			"age":         float64(25),
			"cart_total":  float64(80),
			"user_agent":  "Mozilla/5.0",
			"role":        "admin",
			"app_version": "2.3.1",
		},
	}
}

func TestEvaluateCondition_Equals(t *testing.T) {
	ctx := baseCtx()
	cond := Condition{Attribute: attrCountry, Operator: OpEquals, Value: "BR"}
	if !EvaluateCondition(cond, ctx) {
		t.Error("expected equals to match")
	}

	cond.Value = "US"
	if EvaluateCondition(cond, ctx) {
		t.Error("expected equals to not match")
	}
}

func TestEvaluateCondition_NotEquals(t *testing.T) {
	ctx := baseCtx()
	cond := Condition{Attribute: attrCountry, Operator: OpNotEquals, Value: "US"}
	if !EvaluateCondition(cond, ctx) {
		t.Error("expected not_equals to match")
	}

	cond.Value = "BR"
	if EvaluateCondition(cond, ctx) {
		t.Error("expected not_equals to not match")
	}
}

func TestEvaluateCondition_In(t *testing.T) {
	ctx := baseCtx()
	cond := Condition{Attribute: attrCountry, Operator: OpIn, Value: []any{"BR", "US", "DE"}}
	if !EvaluateCondition(cond, ctx) {
		t.Error("expected in to match")
	}

	cond.Value = []any{"US", "DE"}
	if EvaluateCondition(cond, ctx) {
		t.Error("expected in to not match")
	}
}

func TestEvaluateCondition_InStringSlice(t *testing.T) {
	ctx := baseCtx()
	cond := Condition{Attribute: attrCountry, Operator: OpIn, Value: []string{"BR", "US"}}
	if !EvaluateCondition(cond, ctx) {
		t.Error("expected in with string slice to match")
	}
}

func TestEvaluateCondition_NotIn(t *testing.T) {
	ctx := baseCtx()
	cond := Condition{Attribute: "attributes.role", Operator: OpNotIn, Value: []any{"bot", "internal"}}
	if !EvaluateCondition(cond, ctx) {
		t.Error("expected not_in to match")
	}

	cond.Value = []any{"admin", "bot"}
	if EvaluateCondition(cond, ctx) {
		t.Error("expected not_in to not match")
	}
}

func TestEvaluateCondition_Contains(t *testing.T) {
	ctx := baseCtx()
	cond := Condition{Attribute: "attributes.email", Operator: OpContains, Value: "@acme.com"}
	if !EvaluateCondition(cond, ctx) {
		t.Error("expected contains to match")
	}

	cond.Value = "@other.com"
	if EvaluateCondition(cond, ctx) {
		t.Error("expected contains to not match")
	}
}

func TestEvaluateCondition_StartsWith(t *testing.T) {
	ctx := baseCtx()
	cond := Condition{Attribute: "attributes.path", Operator: OpStartsWith, Value: "/v2/"}
	if !EvaluateCondition(cond, ctx) {
		t.Error("expected starts_with to match")
	}

	cond.Value = "/v3/"
	if EvaluateCondition(cond, ctx) {
		t.Error("expected starts_with to not match")
	}
}

func TestEvaluateCondition_EndsWith(t *testing.T) {
	ctx := baseCtx()
	cond := Condition{Attribute: "attributes.hostname", Operator: OpEndsWith, Value: ".internal"}
	if !EvaluateCondition(cond, ctx) {
		t.Error("expected ends_with to match")
	}

	cond.Value = ".external"
	if EvaluateCondition(cond, ctx) {
		t.Error("expected ends_with to not match")
	}
}

func TestEvaluateCondition_GreaterThan(t *testing.T) {
	ctx := baseCtx()
	cond := Condition{Attribute: "attributes.age", Operator: OpGreaterThan, Value: float64(18)}
	if !EvaluateCondition(cond, ctx) {
		t.Error("expected greater_than to match")
	}

	cond.Value = float64(30)
	if EvaluateCondition(cond, ctx) {
		t.Error("expected greater_than to not match")
	}
}

func TestEvaluateCondition_LessThan(t *testing.T) {
	ctx := baseCtx()
	cond := Condition{Attribute: "attributes.cart_total", Operator: OpLessThan, Value: float64(100)}
	if !EvaluateCondition(cond, ctx) {
		t.Error("expected less_than to match")
	}

	cond.Value = float64(50)
	if EvaluateCondition(cond, ctx) {
		t.Error("expected less_than to not match")
	}
}

func TestEvaluateCondition_GreaterThan_IntTypes(t *testing.T) {
	tests := []struct {
		name string
		val  any
	}{
		{"int", 25},
		{"int32", int32(25)},
		{"int64", int64(25)},
		{"uint", uint(25)},
		{"uint32", uint32(25)},
		{"uint64", uint64(25)},
		{"float32", float32(25)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := EvaluationContext{
				Attributes: map[string]any{"val": tt.val},
			}
			cond := Condition{Attribute: attrVal, Operator: OpGreaterThan, Value: float64(10)}
			if !EvaluateCondition(cond, ctx) {
				t.Errorf("expected greater_than to match for %T", tt.val)
			}
		})
	}
}

func TestEvaluateCondition_Regex(t *testing.T) {
	ctx := baseCtx()
	cond := Condition{Attribute: "attributes.user_agent", Operator: OpRegex, Value: "^Mozilla.*"}
	if !EvaluateCondition(cond, ctx) {
		t.Error("expected regex to match")
	}

	cond.Value = "^Chrome.*"
	if EvaluateCondition(cond, ctx) {
		t.Error("expected regex to not match")
	}
}

func TestEvaluateCondition_Regex_Invalid(t *testing.T) {
	ctx := baseCtx()
	cond := Condition{Attribute: "attributes.user_agent", Operator: OpRegex, Value: "[invalid"}
	if EvaluateCondition(cond, ctx) {
		t.Error("expected invalid regex to not match")
	}
}

func TestEvaluateCondition_SemverMatch(t *testing.T) {
	ctx := baseCtx()

	tests := []struct {
		constraint string
		want       bool
	}{
		{">=2.0.0", true},
		{">=3.0.0", false},
		{">2.3.0", true},
		{">2.3.1", false},
		{"<3.0.0", true},
		{"<2.0.0", false},
		{"<=2.3.1", true},
		{"<=2.3.0", false},
		{"=2.3.1", true},
		{"=2.3.2", false},
		{"2.3.1", true},  // implicit =
		{"v2.3.1", true}, // v prefix
	}

	for _, tt := range tests {
		t.Run(tt.constraint, func(t *testing.T) {
			cond := Condition{Attribute: "attributes.app_version", Operator: OpSemverMatch, Value: tt.constraint}
			got := EvaluateCondition(cond, ctx)
			if got != tt.want {
				t.Errorf("semver_match(%q) = %v, want %v", tt.constraint, got, tt.want)
			}
		})
	}
}

func TestEvaluateCondition_SemverMatch_WithVPrefix(t *testing.T) {
	ctx := EvaluationContext{
		Attributes: map[string]any{"app_version": "v2.3.1"},
	}
	cond := Condition{Attribute: "attributes.app_version", Operator: OpSemverMatch, Value: ">=2.0.0"}
	if !EvaluateCondition(cond, ctx) {
		t.Error("expected semver with v prefix to match")
	}
}

func TestEvaluateCondition_AbsentAttribute(t *testing.T) {
	ctx := EvaluationContext{TenantID: "acme"}
	cond := Condition{Attribute: "attributes.nonexistent", Operator: OpEquals, Value: "anything"}
	if EvaluateCondition(cond, ctx) {
		t.Error("expected absent attribute to not match")
	}
}

func TestEvaluateCondition_NilAttributes(t *testing.T) {
	ctx := EvaluationContext{TenantID: "acme"}
	cond := Condition{Attribute: "attributes.foo", Operator: OpEquals, Value: "bar"}
	if EvaluateCondition(cond, ctx) {
		t.Error("expected nil attributes map to not match")
	}
}

func TestEvaluateCondition_TopLevelAttributes(t *testing.T) {
	ctx := baseCtx()

	tests := []struct {
		attr string
		val  string
	}{
		{"subjectId", "user_123"},
		{"environment", "production"},
		{"tenantId", "acme"},
	}
	for _, tt := range tests {
		t.Run(tt.attr, func(t *testing.T) {
			cond := Condition{Attribute: tt.attr, Operator: OpEquals, Value: tt.val}
			if !EvaluateCondition(cond, ctx) {
				t.Errorf("expected top-level %s to match", tt.attr)
			}
		})
	}
}

func TestEvaluateCondition_EmptySubjectId(t *testing.T) {
	ctx := EvaluationContext{TenantID: "acme"}
	cond := Condition{Attribute: "subjectId", Operator: OpEquals, Value: "user_123"}
	if EvaluateCondition(cond, ctx) {
		t.Error("expected empty subjectId to not resolve")
	}
}

func TestEvaluateCondition_UnknownOperator(t *testing.T) {
	ctx := baseCtx()
	cond := Condition{Attribute: attrCountry, Operator: "unknown_op", Value: "BR"}
	if EvaluateCondition(cond, ctx) {
		t.Error("expected unknown operator to not match")
	}
}

func TestEvaluateCondition_UnknownPath(t *testing.T) {
	ctx := baseCtx()
	cond := Condition{Attribute: "something.else", Operator: OpEquals, Value: "x"}
	if EvaluateCondition(cond, ctx) {
		t.Error("expected unknown path to not match")
	}
}

func TestEvaluateCondition_InInvalidType(t *testing.T) {
	ctx := baseCtx()
	cond := Condition{Attribute: attrCountry, Operator: OpIn, Value: "not_a_slice"}
	if EvaluateCondition(cond, ctx) {
		t.Error("expected non-slice value for in to not match")
	}
}

func TestEvaluateCondition_NumericNonNumericValue(t *testing.T) {
	ctx := EvaluationContext{
		Attributes: map[string]any{"val": "not_a_number"},
	}
	cond := Condition{Attribute: attrVal, Operator: OpGreaterThan, Value: float64(10)}
	if EvaluateCondition(cond, ctx) {
		t.Error("expected non-numeric attribute to not match greater_than")
	}
}

func TestEvaluateCondition_NumericNonNumericExpected(t *testing.T) {
	ctx := EvaluationContext{
		Attributes: map[string]any{"val": float64(25)},
	}
	cond := Condition{Attribute: attrVal, Operator: OpGreaterThan, Value: "not_a_number"}
	if EvaluateCondition(cond, ctx) {
		t.Error("expected non-numeric expected value to not match")
	}
}

func TestAllConditionsMatch(t *testing.T) {
	ctx := baseCtx()
	conditions := []Condition{
		{Attribute: attrCountry, Operator: OpEquals, Value: "BR"},
		{Attribute: "attributes.plan", Operator: OpEquals, Value: "premium"},
	}
	if !AllConditionsMatch(conditions, ctx) {
		t.Error("expected all conditions to match")
	}

	conditions = append(conditions, Condition{
		Attribute: attrCountry, Operator: OpEquals, Value: "US",
	})
	if AllConditionsMatch(conditions, ctx) {
		t.Error("expected not all conditions to match")
	}
}

func TestAllConditionsMatch_Empty(t *testing.T) {
	ctx := baseCtx()
	if !AllConditionsMatch(nil, ctx) {
		t.Error("expected empty conditions to match (vacuous truth)")
	}
}
