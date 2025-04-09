package gcp

import (
	"context"
	"strings"
)

// Option defines optional properties for configuring GCP pub/sub services.
type Option struct {
	ctx     context.Context
	subName string
}

// WithContext sets the context.Context for GCP pub/sub services.
func WithContext(ctx context.Context) func(*Option) {
	return func(o *Option) {
		o.ctx = ctx
	}
}

// WithSubscriptionName sets the subscription name to use when consuming pub/sub events.
//
// The default subscription name is generated using DefaultSubscriptionPrefix and a random 32-bit suffix.
func WithSubscriptionName(name string) func(*Option) {
	return func(o *Option) {
		o.subName = strings.TrimSpace(name)
	}
}
