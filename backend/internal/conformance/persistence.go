package conformance

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
	"github.com/orlandoburli/feature-bacon/modules/postgres/store"
)

const (
	flagTypeBoolean       = "boolean"
	flagSemanticsFlag     = "flag"
	experimentStatusDraft = "draft"
	apiKeyScopeEval       = "read:eval"
)

func ts(id string) *pb.TenantScope {
	return &pb.TenantScope{TenantId: id}
}

func RunPersistenceSuite(t *testing.T, db *sql.DB) {
	t.Helper()
	s := store.New(db)
	ctx := context.Background()

	t.Run("Flags", func(t *testing.T) {
		t.Run("CreateAndGetFlag", func(t *testing.T) {
			tid := t.Name()

			createResp, err := s.CreateFlag(ctx, &pb.CreateFlagRequest{
				Tenant: ts(tid),
				Flag: &pb.FlagDefinition{
					Key:         "my-flag",
					Type:        flagTypeBoolean,
					Semantics:   flagSemanticsFlag,
					Enabled:     true,
					Description: "a test flag",
					CreatedBy:   "tester",
				},
			})
			if err != nil {
				t.Fatalf("CreateFlag: %v", err)
			}

			f := createResp.GetFlag()
			if f.GetKey() != "my-flag" {
				t.Errorf("key = %q, want %q", f.GetKey(), "my-flag")
			}
			if f.GetType() != flagTypeBoolean {
				t.Errorf("type = %q, want %q", f.GetType(), flagTypeBoolean)
			}
			if f.GetSemantics() != flagSemanticsFlag {
				t.Errorf("semantics = %q, want %q", f.GetSemantics(), flagSemanticsFlag)
			}
			if !f.GetEnabled() {
				t.Error("enabled = false, want true")
			}
			if f.GetDescription() != "a test flag" {
				t.Errorf("description = %q, want %q", f.GetDescription(), "a test flag")
			}
			if f.GetCreatedBy() != "tester" {
				t.Errorf("created_by = %q, want %q", f.GetCreatedBy(), "tester")
			}
			if f.GetCreatedAt() == 0 {
				t.Error("created_at should be set")
			}

			getResp, err := s.GetFlag(ctx, &pb.GetFlagRequest{
				Tenant:  ts(tid),
				FlagKey: "my-flag",
			})
			if err != nil {
				t.Fatalf("GetFlag: %v", err)
			}
			got := getResp.GetFlag()
			if got == nil {
				t.Fatal("GetFlag returned nil flag")
			}
			if got.GetKey() != f.GetKey() {
				t.Errorf("got key = %q, want %q", got.GetKey(), f.GetKey())
			}
			if got.GetDescription() != f.GetDescription() {
				t.Errorf("got description = %q, want %q", got.GetDescription(), f.GetDescription())
			}
		})

		t.Run("ListFlags_Pagination", func(t *testing.T) {
			tid := t.Name()
			for _, key := range []string{"flag-a", "flag-b", "flag-c"} {
				_, err := s.CreateFlag(ctx, &pb.CreateFlagRequest{
					Tenant: ts(tid),
					Flag: &pb.FlagDefinition{
						Key:       key,
						Type:      flagTypeBoolean,
						Semantics: flagSemanticsFlag,
						Enabled:   true,
					},
				})
				if err != nil {
					t.Fatalf("CreateFlag(%s): %v", key, err)
				}
			}

			resp1, err := s.ListFlags(ctx, &pb.ListFlagsRequest{
				Tenant:     ts(tid),
				Pagination: &pb.PageRequest{Page: 1, PerPage: 2},
			})
			if err != nil {
				t.Fatalf("ListFlags page 1: %v", err)
			}
			if len(resp1.GetFlags()) != 2 {
				t.Errorf("page 1 count = %d, want 2", len(resp1.GetFlags()))
			}
			if resp1.GetPagination().GetTotal() != 3 {
				t.Errorf("total = %d, want 3", resp1.GetPagination().GetTotal())
			}

			resp2, err := s.ListFlags(ctx, &pb.ListFlagsRequest{
				Tenant:     ts(tid),
				Pagination: &pb.PageRequest{Page: 2, PerPage: 2},
			})
			if err != nil {
				t.Fatalf("ListFlags page 2: %v", err)
			}
			if len(resp2.GetFlags()) != 1 {
				t.Errorf("page 2 count = %d, want 1", len(resp2.GetFlags()))
			}
		})

		t.Run("UpdateFlag", func(t *testing.T) {
			tid := t.Name()
			_, err := s.CreateFlag(ctx, &pb.CreateFlagRequest{
				Tenant: ts(tid),
				Flag: &pb.FlagDefinition{
					Key:         "upd-flag",
					Type:        flagTypeBoolean,
					Semantics:   flagSemanticsFlag,
					Enabled:     true,
					Description: "before",
				},
			})
			if err != nil {
				t.Fatalf("CreateFlag: %v", err)
			}

			updResp, err := s.UpdateFlag(ctx, &pb.UpdateFlagRequest{
				Tenant: ts(tid),
				Flag: &pb.FlagDefinition{
					Key:         "upd-flag",
					Type:        flagTypeBoolean,
					Semantics:   flagSemanticsFlag,
					Enabled:     false,
					Description: "after",
					UpdatedBy:   "editor",
				},
			})
			if err != nil {
				t.Fatalf("UpdateFlag: %v", err)
			}
			f := updResp.GetFlag()
			if f.GetDescription() != "after" {
				t.Errorf("description = %q, want %q", f.GetDescription(), "after")
			}
			if f.GetEnabled() {
				t.Error("enabled = true, want false")
			}
			if f.GetUpdatedBy() != "editor" {
				t.Errorf("updated_by = %q, want %q", f.GetUpdatedBy(), "editor")
			}
		})

		t.Run("DeleteFlag", func(t *testing.T) {
			tid := t.Name()
			_, err := s.CreateFlag(ctx, &pb.CreateFlagRequest{
				Tenant: ts(tid),
				Flag: &pb.FlagDefinition{
					Key:       "del-flag",
					Type:      flagTypeBoolean,
					Semantics: flagSemanticsFlag,
					Enabled:   true,
				},
			})
			if err != nil {
				t.Fatalf("CreateFlag: %v", err)
			}

			_, err = s.DeleteFlag(ctx, &pb.DeleteFlagRequest{
				Tenant:  ts(tid),
				FlagKey: "del-flag",
			})
			if err != nil {
				t.Fatalf("DeleteFlag: %v", err)
			}

			getResp, err := s.GetFlag(ctx, &pb.GetFlagRequest{
				Tenant:  ts(tid),
				FlagKey: "del-flag",
			})
			if err != nil {
				t.Fatalf("GetFlag after delete: %v", err)
			}
			if getResp.GetFlag() != nil {
				t.Errorf("expected nil flag after delete, got key=%q", getResp.GetFlag().GetKey())
			}
		})

		t.Run("TenantIsolation_Flags", func(t *testing.T) {
			tidA := t.Name() + "-A"
			tidB := t.Name() + "-B"

			_, err := s.CreateFlag(ctx, &pb.CreateFlagRequest{
				Tenant: ts(tidA),
				Flag: &pb.FlagDefinition{
					Key:       "isolated",
					Type:      flagTypeBoolean,
					Semantics: flagSemanticsFlag,
					Enabled:   true,
				},
			})
			if err != nil {
				t.Fatalf("CreateFlag tenant A: %v", err)
			}

			getResp, err := s.GetFlag(ctx, &pb.GetFlagRequest{
				Tenant:  ts(tidB),
				FlagKey: "isolated",
			})
			if err != nil {
				t.Fatalf("GetFlag tenant B: %v", err)
			}
			if getResp.GetFlag() != nil {
				t.Error("tenant B should not see tenant A flag")
			}

			listResp, err := s.ListFlags(ctx, &pb.ListFlagsRequest{
				Tenant: ts(tidB),
			})
			if err != nil {
				t.Fatalf("ListFlags tenant B: %v", err)
			}
			if len(listResp.GetFlags()) != 0 {
				t.Errorf("tenant B flag count = %d, want 0", len(listResp.GetFlags()))
			}
		})
	})

	t.Run("Assignments", func(t *testing.T) {
		t.Run("SaveAndGetAssignment", func(t *testing.T) {
			tid := t.Name()

			_, err := s.SaveAssignment(ctx, &pb.SaveAssignmentRequest{
				Tenant: ts(tid),
				Assignment: &pb.Assignment{
					SubjectId: "user-1",
					FlagKey:   "feature-x",
					Enabled:   true,
					Variant:   "control",
				},
			})
			if err != nil {
				t.Fatalf("SaveAssignment: %v", err)
			}

			resp, err := s.GetAssignment(ctx, &pb.GetAssignmentRequest{
				Tenant:    ts(tid),
				SubjectId: "user-1",
				FlagKey:   "feature-x",
			})
			if err != nil {
				t.Fatalf("GetAssignment: %v", err)
			}
			if !resp.GetFound() {
				t.Fatal("expected found=true")
			}
			a := resp.GetAssignment()
			if a.GetSubjectId() != "user-1" {
				t.Errorf("subject_id = %q, want %q", a.GetSubjectId(), "user-1")
			}
			if a.GetFlagKey() != "feature-x" {
				t.Errorf("flag_key = %q, want %q", a.GetFlagKey(), "feature-x")
			}
			if !a.GetEnabled() {
				t.Error("enabled = false, want true")
			}
			if a.GetVariant() != "control" {
				t.Errorf("variant = %q, want %q", a.GetVariant(), "control")
			}
			if a.GetAssignedAt() == 0 {
				t.Error("assigned_at should be set")
			}
		})

		t.Run("AssignmentUpsert", func(t *testing.T) {
			tid := t.Name()
			save := func(variant string) {
				t.Helper()
				_, err := s.SaveAssignment(ctx, &pb.SaveAssignmentRequest{
					Tenant: ts(tid),
					Assignment: &pb.Assignment{
						SubjectId: "user-1",
						FlagKey:   "feature-x",
						Enabled:   true,
						Variant:   variant,
					},
				})
				if err != nil {
					t.Fatalf("SaveAssignment(%s): %v", variant, err)
				}
			}

			save("control")
			save("treatment")

			resp, err := s.GetAssignment(ctx, &pb.GetAssignmentRequest{
				Tenant:    ts(tid),
				SubjectId: "user-1",
				FlagKey:   "feature-x",
			})
			if err != nil {
				t.Fatalf("GetAssignment: %v", err)
			}
			if resp.GetAssignment().GetVariant() != "treatment" {
				t.Errorf("variant = %q, want %q", resp.GetAssignment().GetVariant(), "treatment")
			}
		})

		t.Run("AssignmentExpiry", func(t *testing.T) {
			tid := t.Name()
			past := time.Now().Add(-1 * time.Hour).Unix()

			_, err := s.SaveAssignment(ctx, &pb.SaveAssignmentRequest{
				Tenant: ts(tid),
				Assignment: &pb.Assignment{
					SubjectId: "user-1",
					FlagKey:   "feature-x",
					Enabled:   true,
					Variant:   "control",
					ExpiresAt: past,
				},
			})
			if err != nil {
				t.Fatalf("SaveAssignment: %v", err)
			}

			resp, err := s.GetAssignment(ctx, &pb.GetAssignmentRequest{
				Tenant:    ts(tid),
				SubjectId: "user-1",
				FlagKey:   "feature-x",
			})
			if err != nil {
				t.Fatalf("GetAssignment: %v", err)
			}
			if resp.GetFound() {
				t.Error("expected found=false for expired assignment")
			}
		})

		t.Run("AssignmentNotFound", func(t *testing.T) {
			tid := t.Name()

			resp, err := s.GetAssignment(ctx, &pb.GetAssignmentRequest{
				Tenant:    ts(tid),
				SubjectId: "no-such-user",
				FlagKey:   "no-such-flag",
			})
			if err != nil {
				t.Fatalf("GetAssignment: %v", err)
			}
			if resp.GetFound() {
				t.Error("expected found=false for non-existent assignment")
			}
		})
	})

	t.Run("Experiments", func(t *testing.T) {
		t.Run("CreateAndGetExperiment", func(t *testing.T) {
			tid := t.Name()

			createResp, err := s.CreateExperiment(ctx, &pb.CreateExperimentRequest{
				Tenant: ts(tid),
				Experiment: &pb.Experiment{
					Key:              "exp-1",
					Name:             "Experiment One",
					Status:           experimentStatusDraft,
					StickyAssignment: true,
					Variants: []*pb.Variant{
						{Key: "control", Description: "baseline"},
						{Key: "treatment", Description: "new flow"},
					},
					Allocation: []*pb.Allocation{
						{VariantKey: "control", Percentage: 50},
						{VariantKey: "treatment", Percentage: 50},
					},
				},
			})
			if err != nil {
				t.Fatalf("CreateExperiment: %v", err)
			}

			e := createResp.GetExperiment()
			if e.GetKey() != "exp-1" {
				t.Errorf("key = %q, want %q", e.GetKey(), "exp-1")
			}
			if e.GetName() != "Experiment One" {
				t.Errorf("name = %q, want %q", e.GetName(), "Experiment One")
			}
			if e.GetStatus() != experimentStatusDraft {
				t.Errorf("status = %q, want %q", e.GetStatus(), experimentStatusDraft)
			}
			if !e.GetStickyAssignment() {
				t.Error("sticky_assignment = false, want true")
			}
			if len(e.GetVariants()) != 2 {
				t.Errorf("variants count = %d, want 2", len(e.GetVariants()))
			}
			if len(e.GetAllocation()) != 2 {
				t.Errorf("allocation count = %d, want 2", len(e.GetAllocation()))
			}
			if e.GetCreatedAt() == 0 {
				t.Error("created_at should be set")
			}

			getResp, err := s.GetExperiment(ctx, &pb.GetExperimentRequest{
				Tenant:        ts(tid),
				ExperimentKey: "exp-1",
			})
			if err != nil {
				t.Fatalf("GetExperiment: %v", err)
			}
			got := getResp.GetExperiment()
			if got == nil {
				t.Fatal("GetExperiment returned nil")
			}
			if got.GetKey() != e.GetKey() {
				t.Errorf("got key = %q, want %q", got.GetKey(), e.GetKey())
			}
			if got.GetName() != e.GetName() {
				t.Errorf("got name = %q, want %q", got.GetName(), e.GetName())
			}
			if len(got.GetVariants()) != 2 {
				t.Errorf("got variants count = %d, want 2", len(got.GetVariants()))
			}
			if len(got.GetAllocation()) != 2 {
				t.Errorf("got allocation count = %d, want 2", len(got.GetAllocation()))
			}
		})

		t.Run("ListExperiments_Pagination", func(t *testing.T) {
			tid := t.Name()
			for _, key := range []string{"exp-a", "exp-b", "exp-c"} {
				_, err := s.CreateExperiment(ctx, &pb.CreateExperimentRequest{
					Tenant: ts(tid),
					Experiment: &pb.Experiment{
						Key:    key,
						Name:   key,
						Status: experimentStatusDraft,
					},
				})
				if err != nil {
					t.Fatalf("CreateExperiment(%s): %v", key, err)
				}
			}

			resp1, err := s.ListExperiments(ctx, &pb.ListExperimentsRequest{
				Tenant:     ts(tid),
				Pagination: &pb.PageRequest{Page: 1, PerPage: 2},
			})
			if err != nil {
				t.Fatalf("ListExperiments page 1: %v", err)
			}
			if len(resp1.GetExperiments()) != 2 {
				t.Errorf("page 1 count = %d, want 2", len(resp1.GetExperiments()))
			}
			if resp1.GetPagination().GetTotal() != 3 {
				t.Errorf("total = %d, want 3", resp1.GetPagination().GetTotal())
			}

			resp2, err := s.ListExperiments(ctx, &pb.ListExperimentsRequest{
				Tenant:     ts(tid),
				Pagination: &pb.PageRequest{Page: 2, PerPage: 2},
			})
			if err != nil {
				t.Fatalf("ListExperiments page 2: %v", err)
			}
			if len(resp2.GetExperiments()) != 1 {
				t.Errorf("page 2 count = %d, want 1", len(resp2.GetExperiments()))
			}
		})

		t.Run("UpdateExperiment", func(t *testing.T) {
			tid := t.Name()
			_, err := s.CreateExperiment(ctx, &pb.CreateExperimentRequest{
				Tenant: ts(tid),
				Experiment: &pb.Experiment{
					Key:    "upd-exp",
					Name:   "Before",
					Status: experimentStatusDraft,
				},
			})
			if err != nil {
				t.Fatalf("CreateExperiment: %v", err)
			}

			updResp, err := s.UpdateExperiment(ctx, &pb.UpdateExperimentRequest{
				Tenant: ts(tid),
				Experiment: &pb.Experiment{
					Key:    "upd-exp",
					Name:   "After",
					Status: "running",
				},
			})
			if err != nil {
				t.Fatalf("UpdateExperiment: %v", err)
			}
			e := updResp.GetExperiment()
			if e.GetStatus() != "running" {
				t.Errorf("status = %q, want %q", e.GetStatus(), "running")
			}
			if e.GetName() != "After" {
				t.Errorf("name = %q, want %q", e.GetName(), "After")
			}
		})
	})

	t.Run("APIKeys", func(t *testing.T) {
		t.Run("CreateAndGetAPIKeyByHash", func(t *testing.T) {
			tid := t.Name()

			createResp, err := s.CreateAPIKey(ctx, &pb.CreateAPIKeyRequest{
				Tenant: ts(tid),
				ApiKey: &pb.APIKey{
					Id:        tid + "-key-1",
					KeyHash:   "hash-" + tid,
					KeyPrefix: "fb_",
					Scope:     apiKeyScopeEval,
					Name:      "test key",
					CreatedBy: "admin",
				},
			})
			if err != nil {
				t.Fatalf("CreateAPIKey: %v", err)
			}
			k := createResp.GetApiKey()
			if k.GetId() != tid+"-key-1" {
				t.Errorf("id = %q, want %q", k.GetId(), tid+"-key-1")
			}
			if k.GetKeyHash() != "hash-"+tid {
				t.Errorf("key_hash = %q, want %q", k.GetKeyHash(), "hash-"+tid)
			}
			if k.GetScope() != apiKeyScopeEval {
				t.Errorf("scope = %q, want %q", k.GetScope(), apiKeyScopeEval)
			}
			if k.GetCreatedBy() != "admin" {
				t.Errorf("created_by = %q, want %q", k.GetCreatedBy(), "admin")
			}
			if k.GetCreatedAt() == 0 {
				t.Error("created_at should be set")
			}

			getResp, err := s.GetAPIKeyByHash(ctx, &pb.GetAPIKeyByHashRequest{
				KeyHash: "hash-" + tid,
			})
			if err != nil {
				t.Fatalf("GetAPIKeyByHash: %v", err)
			}
			if !getResp.GetFound() {
				t.Fatal("expected found=true")
			}
			got := getResp.GetApiKey()
			if got.GetId() != k.GetId() {
				t.Errorf("got id = %q, want %q", got.GetId(), k.GetId())
			}
			if getResp.GetTenantId() != tid {
				t.Errorf("tenant_id = %q, want %q", getResp.GetTenantId(), tid)
			}
		})

		t.Run("ListAPIKeys_Pagination", func(t *testing.T) {
			tid := t.Name()
			for i := range 3 {
				_, err := s.CreateAPIKey(ctx, &pb.CreateAPIKeyRequest{
					Tenant: ts(tid),
					ApiKey: &pb.APIKey{
						Id:        fmt.Sprintf("%s-key-%d", tid, i),
						KeyHash:   fmt.Sprintf("hash-%s-%d", tid, i),
						KeyPrefix: "fb_",
						Scope:     apiKeyScopeEval,
						Name:      fmt.Sprintf("key-%d", i),
					},
				})
				if err != nil {
					t.Fatalf("CreateAPIKey(%d): %v", i, err)
				}
			}

			resp1, err := s.ListAPIKeys(ctx, &pb.ListAPIKeysRequest{
				Tenant:     ts(tid),
				Pagination: &pb.PageRequest{Page: 1, PerPage: 2},
			})
			if err != nil {
				t.Fatalf("ListAPIKeys page 1: %v", err)
			}
			if len(resp1.GetApiKeys()) != 2 {
				t.Errorf("page 1 count = %d, want 2", len(resp1.GetApiKeys()))
			}
			if resp1.GetPagination().GetTotal() != 3 {
				t.Errorf("total = %d, want 3", resp1.GetPagination().GetTotal())
			}

			resp2, err := s.ListAPIKeys(ctx, &pb.ListAPIKeysRequest{
				Tenant:     ts(tid),
				Pagination: &pb.PageRequest{Page: 2, PerPage: 2},
			})
			if err != nil {
				t.Fatalf("ListAPIKeys page 2: %v", err)
			}
			if len(resp2.GetApiKeys()) != 1 {
				t.Errorf("page 2 count = %d, want 1", len(resp2.GetApiKeys()))
			}
		})

		t.Run("RevokeAPIKey", func(t *testing.T) {
			tid := t.Name()

			_, err := s.CreateAPIKey(ctx, &pb.CreateAPIKeyRequest{
				Tenant: ts(tid),
				ApiKey: &pb.APIKey{
					Id:        tid + "-key",
					KeyHash:   "hash-" + tid,
					KeyPrefix: "fb_",
					Scope:     apiKeyScopeEval,
					Name:      "revoke-me",
				},
			})
			if err != nil {
				t.Fatalf("CreateAPIKey: %v", err)
			}

			_, err = s.RevokeAPIKey(ctx, &pb.RevokeAPIKeyRequest{
				Tenant: ts(tid),
				KeyId:  tid + "-key",
			})
			if err != nil {
				t.Fatalf("RevokeAPIKey: %v", err)
			}

			getResp, err := s.GetAPIKeyByHash(ctx, &pb.GetAPIKeyByHashRequest{
				KeyHash: "hash-" + tid,
			})
			if err != nil {
				t.Fatalf("GetAPIKeyByHash: %v", err)
			}
			if !getResp.GetFound() {
				t.Fatal("expected found=true after revoke")
			}
			if getResp.GetApiKey().GetRevokedAt() == 0 {
				t.Error("revoked_at should be set after revocation")
			}
		})

		t.Run("GetAPIKeyByHash_NotFound", func(t *testing.T) {
			resp, err := s.GetAPIKeyByHash(ctx, &pb.GetAPIKeyByHashRequest{
				KeyHash: "nonexistent-hash",
			})
			if err != nil {
				t.Fatalf("GetAPIKeyByHash: %v", err)
			}
			if resp.GetFound() {
				t.Error("expected found=false for non-existent key")
			}
		})
	})
}
