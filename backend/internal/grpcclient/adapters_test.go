package grpcclient

import (
	"context"
	"testing"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

func TestFlagManagerAdapter(t *testing.T) {
	conn := startMockServer(t)
	pc := NewPersistenceClient(conn)
	a := NewFlagManagerAdapter(pc)

	flag, err := a.GetFlag(context.Background(), tenantDefault, flagDarkMode)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if flag == nil {
		t.Fatal("expected flag, got nil")
	}
	if flag.Key != flagDarkMode {
		t.Errorf(fmtExpectedKeyGot, flagDarkMode, flag.Key)
	}

	flags, total, err := a.ListFlags(context.Background(), tenantDefault, 1, 20)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if len(flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(flags))
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}

	created, err := a.CreateFlag(context.Background(), tenantDefault, &pb.FlagDefinition{
		Key: flagDarkMode, Type: "boolean", Semantics: "flag",
	})
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if created.Key != flagDarkMode {
		t.Errorf(fmtExpectedKeyGot, flagDarkMode, created.Key)
	}

	updated, err := a.UpdateFlag(context.Background(), tenantDefault, &pb.FlagDefinition{
		Key: flagDarkMode, Type: "boolean", Semantics: "flag",
	})
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if updated.Key != flagDarkMode {
		t.Errorf(fmtExpectedKeyGot, flagDarkMode, updated.Key)
	}

	if err := a.DeleteFlag(context.Background(), tenantDefault, flagDarkMode); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
}

func TestExperimentManagerAdapter(t *testing.T) {
	conn := startMockServer(t)
	pc := NewPersistenceClient(conn)
	a := NewExperimentManagerAdapter(pc)

	exp, err := a.GetExperiment(context.Background(), tenantDefault, experimentKeyOnboarding)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if exp == nil {
		t.Fatal("expected experiment, got nil")
	}
	if exp.Key != experimentKeyOnboarding {
		t.Errorf(fmtExpectedKeyGot, experimentKeyOnboarding, exp.Key)
	}

	exps, total, err := a.ListExperiments(context.Background(), tenantDefault, 1, 20)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if len(exps) != 1 {
		t.Errorf("expected 1 experiment, got %d", len(exps))
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}

	created, err := a.CreateExperiment(context.Background(), tenantDefault, &pb.Experiment{
		Key: experimentKeyOnboarding, Name: "Onboarding",
	})
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if created.Key != experimentKeyOnboarding {
		t.Errorf(fmtExpectedKeyGot, experimentKeyOnboarding, created.Key)
	}

	updated, err := a.UpdateExperiment(context.Background(), tenantDefault, &pb.Experiment{
		Key: experimentKeyOnboarding, Name: "Updated",
	})
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if updated.Key != experimentKeyOnboarding {
		t.Errorf(fmtExpectedKeyGot, experimentKeyOnboarding, updated.Key)
	}
}

func TestAPIKeyManagerAdapter(t *testing.T) {
	conn := startMockServer(t)
	pc := NewPersistenceClient(conn)
	a := NewAPIKeyManagerAdapter(pc)

	keys, total, err := a.ListAPIKeys(context.Background(), tenantDefault, 1, 20)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if len(keys) != 1 {
		t.Errorf("expected 1 key, got %d", len(keys))
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}

	created, err := a.CreateAPIKey(context.Background(), tenantDefault, &pb.APIKey{
		KeyHash: "abc123", KeyPrefix: "ba_eval_", Scope: "evaluation", Name: "test",
	})
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if created.Id != apiKeyIDTest {
		t.Errorf(fmtExpectedKeyGot, apiKeyIDTest, created.Id)
	}

	if err := a.RevokeAPIKey(context.Background(), tenantDefault, apiKeyIDTest); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
}
