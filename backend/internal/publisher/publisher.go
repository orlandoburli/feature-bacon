package publisher

import (
	"context"
	"errors"
	"sync"

	pb "github.com/orlandoburli/feature-bacon/gen/proto/bacon/v1"
)

type Publisher interface {
	Publish(ctx context.Context, event *pb.Event) error
	Close() error
}

// Fanout sends events to multiple publishers in parallel.
// If publishers is nil or empty, Publish is a no-op (core works with zero publishers).
type Fanout struct {
	publishers []Publisher
}

func NewFanout(publishers ...Publisher) *Fanout {
	return &Fanout{publishers: publishers}
}

func (f *Fanout) Publish(ctx context.Context, event *pb.Event) error {
	if len(f.publishers) == 0 {
		return nil
	}

	errs := make([]error, len(f.publishers))
	var wg sync.WaitGroup
	for i, p := range f.publishers {
		wg.Add(1)
		go func(idx int, pub Publisher) {
			defer wg.Done()
			errs[idx] = pub.Publish(ctx, event)
		}(i, p)
	}
	wg.Wait()

	return errors.Join(errs...)
}

func (f *Fanout) Close() error {
	var errs []error
	for _, p := range f.publishers {
		if err := p.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
