package publisher

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

const (
	fmtExpectedNilErr = "expected nil error, got %v"
	fmtUnexpectedErr  = "unexpected error: %v"
)

var errPublish = errors.New("publish failed")

type spyPublisher struct {
	calls  atomic.Int32
	err    error
	closed atomic.Bool
}

func (s *spyPublisher) Publish(_ context.Context, _ *pb.Event) error {
	s.calls.Add(1)
	return s.err
}

func (s *spyPublisher) Close() error {
	s.closed.Store(true)
	return s.err
}

func sampleEvent() *pb.Event {
	return &pb.Event{
		EventId:   "evt-1",
		EventType: EventFlagCreated,
		TenantId:  "t1",
		Timestamp: 1700000000,
	}
}

func TestFanout_ZeroPublishers(t *testing.T) {
	f := NewFanout()
	if err := f.Publish(context.Background(), sampleEvent()); err != nil {
		t.Fatalf(fmtExpectedNilErr, err)
	}
}

func TestFanout_SinglePublisher(t *testing.T) {
	spy := &spyPublisher{}
	f := NewFanout(spy)

	if err := f.Publish(context.Background(), sampleEvent()); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if spy.calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", spy.calls.Load())
	}
}

func TestFanout_MultiplePublishers(t *testing.T) {
	spies := []*spyPublisher{{}, {}, {}}
	pubs := make([]Publisher, len(spies))
	for i, s := range spies {
		pubs[i] = s
	}
	f := NewFanout(pubs...)

	if err := f.Publish(context.Background(), sampleEvent()); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	for i, s := range spies {
		if s.calls.Load() != 1 {
			t.Errorf("publisher[%d] calls = %d, want 1", i, s.calls.Load())
		}
	}
}

func TestFanout_FailingPublisher(t *testing.T) {
	good := &spyPublisher{}
	bad := &spyPublisher{err: errPublish}
	f := NewFanout(good, bad)

	err := f.Publish(context.Background(), sampleEvent())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errPublish) {
		t.Errorf("expected errPublish in error chain, got %v", err)
	}
	if good.calls.Load() != 1 {
		t.Errorf("good publisher calls = %d, want 1", good.calls.Load())
	}
	if bad.calls.Load() != 1 {
		t.Errorf("bad publisher calls = %d, want 1", bad.calls.Load())
	}
}

func TestFanout_Close(t *testing.T) {
	spies := []*spyPublisher{{}, {}}
	f := NewFanout(spies[0], spies[1])

	if err := f.Close(); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	for i, s := range spies {
		if !s.closed.Load() {
			t.Errorf("publisher[%d] not closed", i)
		}
	}
}

func TestFanout_Close_WithError(t *testing.T) {
	bad := &spyPublisher{err: errPublish}
	f := NewFanout(bad)

	err := f.Close()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errPublish) {
		t.Errorf("expected errPublish in error chain, got %v", err)
	}
}

func TestFanout_Close_ZeroPublishers(t *testing.T) {
	f := NewFanout()
	if err := f.Close(); err != nil {
		t.Fatalf(fmtExpectedNilErr, err)
	}
}

func TestFanout_NilSlice(t *testing.T) {
	f := &Fanout{publishers: nil}
	if err := f.Publish(context.Background(), sampleEvent()); err != nil {
		t.Fatalf(fmtExpectedNilErr, err)
	}
}
