package engine

import (
	"errors"
	"testing"
	"time"
)

// stubStore is a simple in-memory FlagStore for testing.
type stubStore struct {
	flags map[string]map[string]*FlagDefinition // tenant -> flagKey -> definition
	err   error
}

func (s *stubStore) GetFlag(tenantID, flagKey string) (*FlagDefinition, error) {
	if s.err != nil {
		return nil, s.err
	}
	tenant, ok := s.flags[tenantID]
	if !ok {
		return nil, nil
	}
	return tenant[flagKey], nil
}

func (s *stubStore) ListFlagKeys(tenantID string) ([]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	tenant, ok := s.flags[tenantID]
	if !ok {
		return nil, nil
	}
	keys := make([]string, 0, len(tenant))
	for k := range tenant {
		keys = append(keys, k)
	}
	return keys, nil
}

func newStore(flags ...*FlagDefinition) *stubStore {
	store := &stubStore{
		flags: map[string]map[string]*FlagDefinition{
			"acme": {},
		},
	}
	for _, f := range flags {
		store.flags["acme"][f.Key] = f
	}
	return store
}

func TestEngine_Store(t *testing.T) {
	store := newStore()
	eng := New(store, nil)
	if eng.Store() != store {
		t.Error("expected Store() to return the underlying FlagStore")
	}
}

func TestEvaluate_NotFound(t *testing.T) {
	eng := New(newStore(), nil)
	ctx := EvaluationContext{TenantID: "acme", SubjectID: "user_1"}

	result := eng.Evaluate("nonexistent", ctx)

	if result.Enabled {
		t.Error("expected not found flag to be disabled")
	}
	if result.Reason != ReasonNotFound {
		t.Errorf("expected reason not_found, got %s", result.Reason)
	}
}

func TestEvaluate_DisabledFlag(t *testing.T) {
	flag := &FlagDefinition{
		Key:       "old_feature",
		Type:      FlagTypeBoolean,
		Semantics: SemanticsDeterministic,
		Enabled:   false,
	}
	eng := New(newStore(flag), nil)
	ctx := EvaluationContext{TenantID: "acme", SubjectID: "user_1"}

	result := eng.Evaluate("old_feature", ctx)

	if result.Enabled {
		t.Error("expected disabled flag to be disabled")
	}
	if result.Reason != ReasonDisabled {
		t.Errorf("expected reason disabled, got %s", result.Reason)
	}
}

func TestEvaluate_StoreError(t *testing.T) {
	store := &stubStore{err: errors.New("connection lost")}
	eng := New(store, nil)
	ctx := EvaluationContext{TenantID: "acme", SubjectID: "user_1"}

	result := eng.Evaluate("any_flag", ctx)

	if result.Enabled {
		t.Error("expected error to disable flag")
	}
	if result.Reason != ReasonError {
		t.Errorf("expected reason error, got %s", result.Reason)
	}
}

func TestEvaluate_DeterministicRuleMatch(t *testing.T) {
	flag := &FlagDefinition{
		Key:       "checkout_redesign",
		Type:      FlagTypeVariant,
		Semantics: SemanticsDeterministic,
		Enabled:   true,
		Rules: []Rule{
			{
				Conditions:        []Condition{{Attribute: attrCountry, Operator: OpIn, Value: []any{"BR", "US"}}},
				RolloutPercentage: 100,
				Variant:           "redesign",
			},
		},
		DefaultResult: EvalResult{Enabled: true, Variant: "control"},
	}
	eng := New(newStore(flag), nil)
	ctx := EvaluationContext{
		TenantID:  "acme",
		SubjectID: "user_123",
		Attributes: map[string]any{
			"country": "BR",
		},
	}

	result := eng.Evaluate("checkout_redesign", ctx)

	if !result.Enabled {
		t.Error("expected flag to be enabled")
	}
	if result.Variant != "redesign" {
		t.Errorf("expected variant redesign, got %s", result.Variant)
	}
	if result.Reason != ReasonRuleMatch {
		t.Errorf("expected reason rule_match, got %s", result.Reason)
	}
}

func TestEvaluate_DefaultResult(t *testing.T) {
	flag := &FlagDefinition{
		Key:       "checkout_redesign",
		Type:      FlagTypeVariant,
		Semantics: SemanticsDeterministic,
		Enabled:   true,
		Rules: []Rule{
			{
				Conditions:        []Condition{{Attribute: attrCountry, Operator: OpEquals, Value: "US"}},
				RolloutPercentage: 100,
				Variant:           "redesign",
			},
		},
		DefaultResult: EvalResult{Enabled: true, Variant: "control"},
	}
	eng := New(newStore(flag), nil)
	ctx := EvaluationContext{
		TenantID:  "acme",
		SubjectID: "user_123",
		Attributes: map[string]any{
			"country": "BR",
		},
	}

	result := eng.Evaluate("checkout_redesign", ctx)

	if result.Variant != "control" {
		t.Errorf("expected default variant control, got %s", result.Variant)
	}
	if result.Reason != ReasonDefault {
		t.Errorf("expected reason default, got %s", result.Reason)
	}
}

func TestEvaluate_RolloutPercentageFilters(t *testing.T) {
	flag := &FlagDefinition{
		Key:       "gradual_rollout",
		Type:      FlagTypeBoolean,
		Semantics: SemanticsDeterministic,
		Enabled:   true,
		Rules: []Rule{
			{
				Conditions:        nil, // matches all
				RolloutPercentage: 0,   // nobody
				Variant:           "",
			},
		},
		DefaultResult: EvalResult{Enabled: false},
	}
	eng := New(newStore(flag), nil)
	ctx := EvaluationContext{TenantID: "acme", SubjectID: "user_123"}

	result := eng.Evaluate("gradual_rollout", ctx)

	if result.Reason != ReasonDefault {
		t.Errorf("expected 0%% rollout to fall through to default, got reason %s", result.Reason)
	}
}

func TestEvaluate_FirstMatchWins(t *testing.T) {
	flag := &FlagDefinition{
		Key:       "multi_rule",
		Type:      FlagTypeVariant,
		Semantics: SemanticsDeterministic,
		Enabled:   true,
		Rules: []Rule{
			{
				Conditions:        []Condition{{Attribute: "attributes.plan", Operator: OpEquals, Value: "premium"}},
				RolloutPercentage: 100,
				Variant:           "premium_variant",
			},
			{
				Conditions:        nil, // matches all
				RolloutPercentage: 100,
				Variant:           "general_variant",
			},
		},
		DefaultResult: EvalResult{Enabled: false, Variant: "default"},
	}
	eng := New(newStore(flag), nil)
	ctx := EvaluationContext{
		TenantID:  "acme",
		SubjectID: "user_123",
		Attributes: map[string]any{
			"plan": "premium",
		},
	}

	result := eng.Evaluate("multi_rule", ctx)

	if result.Variant != "premium_variant" {
		t.Errorf("expected first matching rule (premium_variant), got %s", result.Variant)
	}
}

func TestEvaluate_BooleanFlag(t *testing.T) {
	flag := &FlagDefinition{
		Key:       "maintenance_mode",
		Type:      FlagTypeBoolean,
		Semantics: SemanticsDeterministic,
		Enabled:   true,
		Rules: []Rule{
			{
				Conditions:        nil,
				RolloutPercentage: 100,
			},
		},
		DefaultResult: EvalResult{Enabled: false},
	}
	eng := New(newStore(flag), nil)
	ctx := EvaluationContext{TenantID: "acme", SubjectID: "user_1"}

	result := eng.Evaluate("maintenance_mode", ctx)

	if !result.Enabled {
		t.Error("expected boolean flag to be enabled")
	}
	if result.Variant != "" {
		t.Errorf("expected empty variant for boolean flag, got %q", result.Variant)
	}
}

func TestEvaluate_TenantIsolation(t *testing.T) {
	store := &stubStore{
		flags: map[string]map[string]*FlagDefinition{
			"acme": {
				"dark_mode": {
					Key: "dark_mode", Type: FlagTypeBoolean, Semantics: SemanticsDeterministic,
					Enabled: true,
					Rules:   []Rule{{Conditions: nil, RolloutPercentage: 100}},
				},
			},
			"globex": {
				"dark_mode": {
					Key: "dark_mode", Type: FlagTypeBoolean, Semantics: SemanticsDeterministic,
					Enabled:       false,
					DefaultResult: EvalResult{Enabled: false},
				},
			},
		},
	}
	eng := New(store, nil)

	acmeResult := eng.Evaluate("dark_mode", EvaluationContext{TenantID: "acme", SubjectID: "u1"})
	if !acmeResult.Enabled {
		t.Error("expected acme dark_mode to be enabled")
	}

	globexResult := eng.Evaluate("dark_mode", EvaluationContext{TenantID: "globex", SubjectID: "u1"})
	if globexResult.Enabled {
		t.Error("expected globex dark_mode to be disabled")
	}
}

func TestEvaluate_WrongTenant(t *testing.T) {
	flag := &FlagDefinition{
		Key: "some_flag", Type: FlagTypeBoolean, Semantics: SemanticsDeterministic,
		Enabled: true,
		Rules:   []Rule{{Conditions: nil, RolloutPercentage: 100}},
	}
	eng := New(newStore(flag), nil)

	result := eng.Evaluate("some_flag", EvaluationContext{TenantID: "other_tenant", SubjectID: "u1"})

	if result.Reason != ReasonNotFound {
		t.Errorf("expected not_found for wrong tenant, got %s", result.Reason)
	}
}

func TestEvaluateBatch(t *testing.T) {
	flags := []*FlagDefinition{
		{
			Key: "flag_a", Type: FlagTypeBoolean, Semantics: SemanticsDeterministic,
			Enabled: true,
			Rules:   []Rule{{Conditions: nil, RolloutPercentage: 100}},
		},
		{
			Key: "flag_b", Type: FlagTypeBoolean, Semantics: SemanticsDeterministic,
			Enabled: false,
		},
	}
	eng := New(newStore(flags...), nil)
	ctx := EvaluationContext{TenantID: "acme", SubjectID: "user_1"}

	results := eng.EvaluateBatch([]string{"flag_a", "flag_b", "nonexistent"}, ctx)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if !results[0].Enabled || results[0].Reason != ReasonRuleMatch {
		t.Errorf("flag_a: expected enabled/rule_match, got enabled=%v reason=%s", results[0].Enabled, results[0].Reason)
	}
	if results[1].Enabled || results[1].Reason != ReasonDisabled {
		t.Errorf("flag_b: expected disabled, got enabled=%v reason=%s", results[1].Enabled, results[1].Reason)
	}
	if results[2].Enabled || results[2].Reason != ReasonNotFound {
		t.Errorf("nonexistent: expected not_found, got enabled=%v reason=%s", results[2].Enabled, results[2].Reason)
	}
}

func TestEvaluate_RandomSemantics(t *testing.T) {
	flag := &FlagDefinition{
		Key:       "random_banner",
		Type:      FlagTypeBoolean,
		Semantics: SemanticsRandom,
		Enabled:   true,
		Rules: []Rule{
			{
				Conditions:        nil,
				RolloutPercentage: 50,
			},
		},
		DefaultResult: EvalResult{Enabled: false},
	}
	eng := New(newStore(flag), nil)
	ctx := EvaluationContext{TenantID: "acme", SubjectID: "user_123"}

	enabledCount := 0
	n := 1000
	for i := 0; i < n; i++ {
		result := eng.Evaluate("random_banner", ctx)
		if result.Enabled {
			enabledCount++
		}
	}

	pct := float64(enabledCount) / float64(n) * 100
	if pct < 30 || pct > 70 {
		t.Errorf("random 50%% rollout expected ~50%% enabled, got %.1f%%", pct)
	}
}

func TestEvaluate_RandomZeroPercent(t *testing.T) {
	flag := &FlagDefinition{
		Key:       "random_zero",
		Type:      FlagTypeBoolean,
		Semantics: SemanticsRandom,
		Enabled:   true,
		Rules: []Rule{
			{
				Conditions:        nil,
				RolloutPercentage: 0,
			},
		},
		DefaultResult: EvalResult{Enabled: false},
	}
	eng := New(newStore(flag), nil)
	ctx := EvaluationContext{TenantID: "acme", SubjectID: "user_123"}

	result := eng.Evaluate("random_zero", ctx)
	if result.Reason != ReasonDefault {
		t.Errorf("expected 0%% random to fall through to default, got %s", result.Reason)
	}
}

func TestEvaluate_RandomHundredPercent(t *testing.T) {
	flag := &FlagDefinition{
		Key:       "random_hundred",
		Type:      FlagTypeBoolean,
		Semantics: SemanticsRandom,
		Enabled:   true,
		Rules: []Rule{
			{
				Conditions:        nil,
				RolloutPercentage: 100,
			},
		},
		DefaultResult: EvalResult{Enabled: false},
	}
	eng := New(newStore(flag), nil)
	ctx := EvaluationContext{TenantID: "acme", SubjectID: "user_123"}

	for i := 0; i < 100; i++ {
		result := eng.Evaluate("random_hundred", ctx)
		if !result.Enabled {
			t.Fatal("expected 100% random to always be enabled")
		}
	}
}

func TestEvaluate_ResultFields(t *testing.T) {
	flag := &FlagDefinition{
		Key:       "my_flag",
		Type:      FlagTypeVariant,
		Semantics: SemanticsDeterministic,
		Enabled:   true,
		Rules: []Rule{
			{
				Conditions:        nil,
				RolloutPercentage: 100,
				Variant:           "variant_a",
			},
		},
	}
	eng := New(newStore(flag), nil)
	ctx := EvaluationContext{TenantID: "acme", SubjectID: "user_1"}

	result := eng.Evaluate("my_flag", ctx)

	if result.TenantID != "acme" {
		t.Errorf("expected tenantId acme, got %s", result.TenantID)
	}
	if result.FlagKey != "my_flag" {
		t.Errorf("expected flagKey my_flag, got %s", result.FlagKey)
	}
}

// stubAssignmentStore is an in-memory AssignmentStore for testing.
type stubAssignmentStore struct {
	data map[string]*Assignment // key: "tenant/subject/flag"
	err  error
}

func newStubAssignmentStore() *stubAssignmentStore {
	return &stubAssignmentStore{data: make(map[string]*Assignment)}
}

func (s *stubAssignmentStore) key(tid, sid, fk string) string {
	return tid + "/" + sid + "/" + fk
}

func (s *stubAssignmentStore) GetAssignment(tenantID, subjectID, flagKey string) (*Assignment, bool, error) {
	if s.err != nil {
		return nil, false, s.err
	}
	a, ok := s.data[s.key(tenantID, subjectID, flagKey)]
	return a, ok, nil
}

func (s *stubAssignmentStore) SaveAssignment(tenantID string, a *Assignment) error {
	if s.err != nil {
		return s.err
	}
	s.data[s.key(tenantID, a.SubjectID, a.FlagKey)] = a
	return nil
}

func TestEvaluate_PersistentFlag_NilStore(t *testing.T) {
	flag := &FlagDefinition{
		Key: "sticky_flag", Type: FlagTypeBoolean, Semantics: SemanticsPersistent,
		Enabled: true,
		Rules:   []Rule{{Conditions: nil, RolloutPercentage: 100}},
	}
	eng := New(newStore(flag), nil)
	ctx := EvaluationContext{TenantID: "acme", SubjectID: "user_1"}

	result := eng.Evaluate("sticky_flag", ctx)
	if !result.Enabled {
		t.Error("expected enabled")
	}
	if result.Reason != ReasonNoPersistence {
		t.Errorf("expected reason no_persistence, got %s", result.Reason)
	}
}

func TestEvaluate_PersistentFlag_SaveAndRetrieve(t *testing.T) {
	flag := &FlagDefinition{
		Key: "sticky_flag", Type: FlagTypeVariant, Semantics: SemanticsPersistent,
		Enabled: true,
		Rules:   []Rule{{Conditions: nil, RolloutPercentage: 100, Variant: "v1"}},
	}
	as := newStubAssignmentStore()
	eng := New(newStore(flag), as)
	ctx := EvaluationContext{TenantID: "acme", SubjectID: "user_1"}

	r1 := eng.Evaluate("sticky_flag", ctx)
	if r1.Reason != ReasonRuleMatch {
		t.Errorf("first eval expected rule_match, got %s", r1.Reason)
	}

	r2 := eng.Evaluate("sticky_flag", ctx)
	if r2.Reason != ReasonPersisted {
		t.Errorf("second eval expected persisted, got %s", r2.Reason)
	}
	if r2.Variant != r1.Variant {
		t.Errorf("expected same variant, got %s vs %s", r1.Variant, r2.Variant)
	}
}

func TestEvaluate_PersistentFlag_ExistingAssignment(t *testing.T) {
	flag := &FlagDefinition{
		Key: "sticky_flag", Type: FlagTypeVariant, Semantics: SemanticsPersistent,
		Enabled: true,
		Rules:   []Rule{{Conditions: nil, RolloutPercentage: 100, Variant: "new_variant"}},
	}
	as := newStubAssignmentStore()
	as.data["acme/user_1/sticky_flag"] = &Assignment{
		SubjectID: "user_1", FlagKey: "sticky_flag",
		Enabled: true, Variant: "old_variant",
		AssignedAt: time.Now().Add(-time.Hour),
	}
	eng := New(newStore(flag), as)
	ctx := EvaluationContext{TenantID: "acme", SubjectID: "user_1"}

	result := eng.Evaluate("sticky_flag", ctx)
	if result.Variant != "old_variant" {
		t.Errorf("expected persisted old_variant, got %s", result.Variant)
	}
	if result.Reason != ReasonPersisted {
		t.Errorf("expected reason persisted, got %s", result.Reason)
	}
}
