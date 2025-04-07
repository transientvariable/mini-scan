package gcp

import (
	"context"
	"sync"

	"github.com/censys/scan-takehome/pkg/pubsub"
)

var (
	_ pubsub.Publisher[constr] = (*Publisher[constr])(nil)
)

type Publisher[M Message] struct {
	closed bool
	mutex  sync.Mutex
}

func NewPublisher[M Message]() (*Publisher[M], error) {
	return &Publisher[M]{}, nil
}

func (p *Publisher[M]) Close() {

}

func (p *Publisher[M]) Publish(ctx context.Context, msg M) error {
	return nil
}
