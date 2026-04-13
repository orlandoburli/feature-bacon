package handlers

import (
	"context"
	"errors"
	"testing"
	"time"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

const (
	tenantTest      = "tenant-1"
	flagKeyDM       = "dark-mode"
	expKeyOB        = "onboarding"
	fmtUnexpectedPE = "unexpected error: %v"
	fmtEventTypeW   = "EventType = %q, want %q"
)

var errPub = errors.New("publish failed")

type recordingPublisher struct {
	events chan *pb.Event
	err    error
}

func newRecordingPublisher() *recordingPublisher {
	return &recordingPublisher{events: make(chan *pb.Event, 10)}
}

func (p *recordingPublisher) Publish(_ context.Context, event *pb.Event) error {
	if p.err != nil {
		return p.err
	}
	p.events <- event
	return nil
}

func (p *recordingPublisher) Close() error { return nil }

func (p *recordingPublisher) waitEvent(t *testing.T) *pb.Event {
	t.Helper()
	select {
	case e := <-p.events:
		return e
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for published event")
		return nil
	}
}

// -- PublishingFlagManager tests --

func TestPublishingFlagManager_GetFlag_Delegates(t *testing.T) {
	fm := &mockFlagManager{
		getFunc: func(_ context.Context, _, _ string) (*pb.FlagDefinition, error) {
			return sampleFlag(), nil
		},
	}
	pub := newRecordingPublisher()
	pfm := NewPublishingFlagManager(fm, pub)

	flag, err := pfm.GetFlag(context.Background(), tenantTest, flagKeyDM)
	if err != nil {
		t.Fatalf(fmtUnexpectedPE, err)
	}
	if flag.Key != flagKeyMyFlag {
		t.Errorf(fmtKeyWant, flag.Key, flagKeyMyFlag)
	}
}

func TestPublishingFlagManager_ListFlags_Delegates(t *testing.T) {
	fm := &mockFlagManager{
		listFunc: func(_ context.Context, _ string, _, _ int) ([]*pb.FlagDefinition, int, error) {
			return []*pb.FlagDefinition{sampleFlag()}, 1, nil
		},
	}
	pub := newRecordingPublisher()
	pfm := NewPublishingFlagManager(fm, pub)

	flags, total, err := pfm.ListFlags(context.Background(), tenantTest, 1, 10)
	if err != nil {
		t.Fatalf(fmtUnexpectedPE, err)
	}
	if len(flags) != 1 {
		t.Errorf("flags len = %d, want 1", len(flags))
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
}

func TestPublishingFlagManager_CreateFlag_PublishesEvent(t *testing.T) {
	fm := &mockFlagManager{
		createFunc: func(_ context.Context, _ string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error) {
			flag.CreatedAt = 1700000000
			return flag, nil
		},
	}
	pub := newRecordingPublisher()
	pfm := NewPublishingFlagManager(fm, pub)

	flag, err := pfm.CreateFlag(context.Background(), tenantTest, &pb.FlagDefinition{
		Key: flagKeyDM, Type: "boolean", Semantics: "flag",
	})
	if err != nil {
		t.Fatalf(fmtUnexpectedPE, err)
	}
	if flag.Key != flagKeyDM {
		t.Errorf(fmtKeyWant, flag.Key, flagKeyDM)
	}

	event := pub.waitEvent(t)
	if event.EventType != "flag.created" {
		t.Errorf(fmtEventTypeW, event.EventType, "flag.created")
	}
	if event.TenantId != tenantTest {
		t.Errorf("TenantId = %q, want %q", event.TenantId, tenantTest)
	}
}

func TestPublishingFlagManager_UpdateFlag_PublishesEvent(t *testing.T) {
	fm := &mockFlagManager{
		updateFunc: func(_ context.Context, _ string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error) {
			flag.UpdatedAt = 1700000001
			return flag, nil
		},
	}
	pub := newRecordingPublisher()
	pfm := NewPublishingFlagManager(fm, pub)

	_, err := pfm.UpdateFlag(context.Background(), tenantTest, &pb.FlagDefinition{Key: flagKeyDM})
	if err != nil {
		t.Fatalf(fmtUnexpectedPE, err)
	}

	event := pub.waitEvent(t)
	if event.EventType != "flag.updated" {
		t.Errorf(fmtEventTypeW, event.EventType, "flag.updated")
	}
}

func TestPublishingFlagManager_DeleteFlag_PublishesEvent(t *testing.T) {
	fm := &mockFlagManager{
		deleteFunc: func(_ context.Context, _, _ string) error { return nil },
	}
	pub := newRecordingPublisher()
	pfm := NewPublishingFlagManager(fm, pub)

	if err := pfm.DeleteFlag(context.Background(), tenantTest, flagKeyDM); err != nil {
		t.Fatalf(fmtUnexpectedPE, err)
	}

	event := pub.waitEvent(t)
	if event.EventType != "flag.deleted" {
		t.Errorf(fmtEventTypeW, event.EventType, "flag.deleted")
	}
}

func TestPublishingFlagManager_CreateFlag_InnerError(t *testing.T) {
	fm := &mockFlagManager{
		createFunc: func(_ context.Context, _ string, _ *pb.FlagDefinition) (*pb.FlagDefinition, error) {
			return nil, errStore
		},
	}
	pub := newRecordingPublisher()
	pfm := NewPublishingFlagManager(fm, pub)

	_, err := pfm.CreateFlag(context.Background(), tenantTest, &pb.FlagDefinition{Key: flagKeyDM})
	if !errors.Is(err, errStore) {
		t.Errorf("expected errStore, got %v", err)
	}
}

func TestPublishingFlagManager_PublishFailure_DoesNotAffectCaller(t *testing.T) {
	fm := &mockFlagManager{
		createFunc: func(_ context.Context, _ string, flag *pb.FlagDefinition) (*pb.FlagDefinition, error) {
			return flag, nil
		},
	}
	pub := &recordingPublisher{events: make(chan *pb.Event, 10), err: errPub}
	pfm := NewPublishingFlagManager(fm, pub)

	flag, err := pfm.CreateFlag(context.Background(), tenantTest, &pb.FlagDefinition{Key: flagKeyDM})
	if err != nil {
		t.Fatalf("expected no error from caller, got %v", err)
	}
	if flag.Key != flagKeyDM {
		t.Errorf(fmtKeyWant, flag.Key, flagKeyDM)
	}
}

// -- PublishingExperimentManager tests --

func TestPublishingExperimentManager_GetExperiment_Delegates(t *testing.T) {
	em := &mockExperimentManager{
		getFunc: func(_ context.Context, _, _ string) (*pb.Experiment, error) {
			return sampleExperiment(statusDraft), nil
		},
	}
	pub := newRecordingPublisher()
	pem := NewPublishingExperimentManager(em, pub)

	exp, err := pem.GetExperiment(context.Background(), tenantTest, expKeyOB)
	if err != nil {
		t.Fatalf(fmtUnexpectedPE, err)
	}
	if exp.Key != experimentKeyOnboarding {
		t.Errorf(fmtKeyWant, exp.Key, experimentKeyOnboarding)
	}
}

func TestPublishingExperimentManager_ListExperiments_Delegates(t *testing.T) {
	em := &mockExperimentManager{
		listFunc: func(_ context.Context, _ string, _, _ int) ([]*pb.Experiment, int, error) {
			return []*pb.Experiment{sampleExperiment(statusDraft)}, 1, nil
		},
	}
	pub := newRecordingPublisher()
	pem := NewPublishingExperimentManager(em, pub)

	exps, total, err := pem.ListExperiments(context.Background(), tenantTest, 1, 10)
	if err != nil {
		t.Fatalf(fmtUnexpectedPE, err)
	}
	if len(exps) != 1 {
		t.Errorf("experiments len = %d, want 1", len(exps))
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
}

func TestPublishingExperimentManager_CreateExperiment_PublishesEvent(t *testing.T) {
	em := &mockExperimentManager{
		createFunc: func(_ context.Context, _ string, exp *pb.Experiment) (*pb.Experiment, error) {
			exp.CreatedAt = 1700000000
			exp.Status = statusDraft
			return exp, nil
		},
	}
	pub := newRecordingPublisher()
	pem := NewPublishingExperimentManager(em, pub)

	exp, err := pem.CreateExperiment(context.Background(), tenantTest, &pb.Experiment{
		Key: expKeyOB, Name: "Onboarding",
	})
	if err != nil {
		t.Fatalf(fmtUnexpectedPE, err)
	}
	if exp.Key != expKeyOB {
		t.Errorf(fmtKeyWant, exp.Key, expKeyOB)
	}

	event := pub.waitEvent(t)
	if event.EventType != "experiment.created" {
		t.Errorf(fmtEventTypeW, event.EventType, "experiment.created")
	}
	if event.TenantId != tenantTest {
		t.Errorf("TenantId = %q, want %q", event.TenantId, tenantTest)
	}
}

func TestPublishingExperimentManager_UpdateExperiment_PublishesEvent(t *testing.T) {
	em := &mockExperimentManager{
		updateFunc: func(_ context.Context, _ string, exp *pb.Experiment) (*pb.Experiment, error) {
			exp.UpdatedAt = 1700000001
			return exp, nil
		},
	}
	pub := newRecordingPublisher()
	pem := NewPublishingExperimentManager(em, pub)

	_, err := pem.UpdateExperiment(context.Background(), tenantTest, &pb.Experiment{Key: expKeyOB})
	if err != nil {
		t.Fatalf(fmtUnexpectedPE, err)
	}

	event := pub.waitEvent(t)
	if event.EventType != "experiment.updated" {
		t.Errorf(fmtEventTypeW, event.EventType, "experiment.updated")
	}
}

func TestPublishingExperimentManager_CreateExperiment_InnerError(t *testing.T) {
	em := &mockExperimentManager{
		createFunc: func(_ context.Context, _ string, _ *pb.Experiment) (*pb.Experiment, error) {
			return nil, errStore
		},
	}
	pub := newRecordingPublisher()
	pem := NewPublishingExperimentManager(em, pub)

	_, err := pem.CreateExperiment(context.Background(), tenantTest, &pb.Experiment{Key: expKeyOB})
	if !errors.Is(err, errStore) {
		t.Errorf("expected errStore, got %v", err)
	}
}
