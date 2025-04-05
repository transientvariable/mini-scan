package scan

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"cloud.google.com/go/pubsub"
	"github.com/transientvariable/log-go"

	opensearch "github.com/transientvariable/repository-opensearch-go"
)

type Watcher struct {
	closed bool
	docs   *opensearch.Repository
	mutex  sync.Mutex
	sub    *pubsub.Subscription
}

// NewEventWatcher ...
func NewEventWatcher(sub *pubsub.Subscription, options ...func(*Option)) (*Watcher, error) {
	return nil, nil
}

// Run ...
func (w *Watcher) Run(ctx context.Context) (<-chan *Event, error) {
	log.Info("[scan:watcher] begin listening for events")

	eventStream := make(chan *Event)
	go func() {
		defer close(eventStream)

		for {
			err := w.sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {

			})
			if err != nil && !errors.Is(err, context.Canceled) {
				log.Warn("[event:watcher] terminating event stream")
				// Handle error.
			}

			select {
			case eventStream <- nil:
			case <-ctx.Done():
				return
			}
		}
	}()
	return eventStream, nil
}

// Close releases any resources used by the watcher.
func (w *Watcher) Close() error {
	if w == nil {
		return ErrInvalid
	}

	w.mutex.Lock()
	defer w.mutex.Unlock()

	if !w.closed {
		w.closed = true
		//if w.weed != nil {
		//	if err := w.weed.Close(); err != nil && !errors.Is(err, gofs.ErrClosed) {
		//		return err
		//	}
		//}
		return nil
	}
	return fmt.Errorf("even_watcher: %w", ErrClosed)
}
