package miniscan

import (
	"context"
	"errors"
	"time"
)

// StreamOf returns a sequential ordered stream whose elements are the specified values.
func StreamOf[E any](values ...E) <-chan E {
	stream := make(chan E)
	go func() {
		defer close(stream)
		for _, i := range values {
			stream <- i
		}
	}()
	return stream
}

// Generate produces a continuous stream of elements using the specified function for producing elements and a
// time.Ticker defining the delay between producing elements.
func Generate[E any](ctx context.Context, supplier func() E, interval time.Duration) (<-chan E, error) {
	if supplier == nil {
		return nil, errors.New("pipeline: supplier is required")
	}

	if interval <= 0 || interval > time.Minute {
		return nil, errors.New("pipeline: time interval is required")
	}

	stream := make(chan E)
	go func() {
		defer close(stream)
		for range time.Tick(interval) {
			select {
			case stream <- supplier():
			case <-ctx.Done():
				return
			}
		}
	}()
	return stream, nil
}
