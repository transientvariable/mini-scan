package gcp

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/transientvariable/log-go"

	gcpubsub "cloud.google.com/go/pubsub"
	gcdpubsub "gocloud.dev/pubsub"
)

const (
	readinessProbeMaxRetries    = 20
	readinessProbeRetryInterval = 1 * time.Second
)

type ReadyFn func() (any, error)

// Message defines the constraints for message types.
type Message interface {
	[]byte | *gcpubsub.Message | *gcdpubsub.Message | constr
}

// constr is used for compile-time assertion for types that handle the Message constraint.
type constr interface{}

// Ready is a readiness probe for GCP pub/sub services.
func Ready(ctx context.Context, resource string, fn ReadyFn) (any, error) {
	ready := make(chan any)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		pollReadyState(ctx, fn, ready)
		wg.Done()
	}()

	go func() {
		wg.Wait()
		close(ready)
	}()

	retries := 0
	for resp := range ready {
		if retries > readinessProbeMaxRetries {
			break
		}
		if _, ok := resp.(error); ok {
			log.Info("[gcp:pubsub] waiting for resource", log.String("name", resource))
			retries++
			continue
		}
		return resp, nil
	}
	return nil, errors.New("gcp_pubsub: maximum number of retries exceeded")
}

func pollReadyState(ctx context.Context, fn ReadyFn, ready chan<- any) {
	ticker := backoff.NewTicker(backoff.NewConstantBackOff(readinessProbeRetryInterval))
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r, err := fn()
			if err != nil {
				ready <- err
				continue
			}
			ready <- r
		case <-ctx.Done():
			return
		}
	}
}
