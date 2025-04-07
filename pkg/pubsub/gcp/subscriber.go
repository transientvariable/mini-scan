package gcp

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"sync"

	"github.com/censys/scan-takehome/pkg"
	"github.com/censys/scan-takehome/pkg/pubsub"

	"github.com/transientvariable/anchor"
	"github.com/transientvariable/log-go"

	gcdpubsub "gocloud.dev/pubsub"
)

const (
	// DefaultSubscriptionPrefix defines the default subscription name prefix.
	DefaultSubscriptionPrefix = "sub"

	// SubscriptionURLFormat ...
	SubscriptionURLFormat = "gcppubsub://%s/%s"
)

var (
	_ pubsub.Subscriber[constr] = (*Subscriber[constr])(nil)
)

// messageACK is used to perform a type assertion for a Message that supports acknowledgement callback.
type messageACK interface {
	Ack()
}

// Subscriber implementation of a pubsub.Subscriber for consuming messages from a GCP pub/sub topic.
type Subscriber[M Message] struct {
	closed      bool
	handler     pubsub.Handler[M]
	mutex       sync.Mutex
	numHandlers int
	projectId   string
	subName     string
	subURL      *url.URL
}

// NewSubscriber creates a new pubsub.Subscriber implementation of a pubsub.Subscriber for consuming messages from a GCP
// pub/sub topic.
func NewSubscriber[M Message](projectId string, options ...func(*Subscriber[M])) (*Subscriber[M], error) {
	if projectId = strings.TrimSpace(projectId); projectId == "" {
		return nil, errors.New("gcp_subscriber: project Id is required")
	}

	s := &Subscriber[M]{handler: &LogHandler[M]{}, projectId: projectId}
	for _, opt := range options {
		opt(s)
	}

	if s.subName == "" {
		s.subName = fmt.Sprintf("%s-%d", DefaultSubscriptionPrefix, int32(rand.Uint32()))
	}

	if u, err := url.Parse(fmt.Sprintf(SubscriptionURLFormat, s.projectId, s.subName)); err == nil {
		s.subURL = u
	} else {
		return nil, fmt.Errorf("gcp_subscriber: failed to parse subscription url: %w", err)
	}
	return s, nil
}

// Subscribe pulls scan events from a GCP pub/sub topic and invokes the optional list of handlers.
func (s *Subscriber[M]) Subscribe(ctx context.Context, handlers ...pubsub.Handler[M]) error {
	if err := s.stat(); err != nil {
		return err
	}

	log.Info("[gcp_subscriber:subscribe] opening subscription", log.String("url", s.subURL.String()))

	for _, h := range handlers {
		s.Add(h)
	}

	sub, err := gcdpubsub.OpenSubscription(ctx, s.subURL.String())
	if err != nil {
		return err
	}

	go func() {
		defer func(ctx context.Context, sub *gcdpubsub.Subscription) {
			if err := sub.Shutdown(ctx); err != nil {
				log.Error("[gcp_subscriber:subscribe]", log.Err(err))
			}
		}(ctx, sub)

		for {
			msg, err := sub.Receive(ctx)
			if err != nil {
				log.Error("[gcp_subscriber:subscribe]", log.Err(err))
				break
			}

			select {
			case <-ctx.Done():
				log.Info("[gcp_subscriber:subscribe] closing subscription")
				return
			default:
				if err := s.onMessage(ctx, any(msg)); err != nil {
					log.Error("[gcp_subscriber:subscribe]", log.Err(err))
					return
				}
			}
		}
	}()
	return nil
}

// Add appends the provided handler to the chain of handlers for the Subscriber.
func (s *Subscriber[M]) Add(handler pubsub.Handler[M]) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.handler != nil {
		last := s.handler
		for {
			if last.Next() == nil {
				break
			}
			last = last.Next()
		}
		last.SetNext(handler)
		s.numHandlers += 1
	} else {
		s.handler = handler
	}
	return true
}

// ProjectID returns the pub/sub project ID configured for the Subscriber.
func (s *Subscriber[M]) ProjectID() string {
	return s.projectId
}

// SubscriptionName returns the name that uniquely identifies a subscription for the Subscriber.
func (s *Subscriber[M]) SubscriptionName() string {
	return s.subName
}

// Close releases any resources used by the Subscriber.
func (s *Subscriber[M]) Close() error {
	if s == nil {
		return miniscan.ErrInvalid
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.closed {
		s.closed = true
		return nil
	}
	return fmt.Errorf("gcp_subscriber: %w: %s", pubsub.ErrSubscriberClosed, s.subName)
}

// String returns a string representing the current state of the Subscriber.
func (s *Subscriber[M]) String() string {
	m := make(map[string]any)
	m["gcp_subscriber"] = map[string]any{
		"project_id": s.ProjectID(),
		"subscription": map[string]any{
			"name": s.SubscriptionName(),
			"url":  s.subURL.String(),
		},
	}
	return string(anchor.ToJSONFormatted(m))
}

func (s *Subscriber[M]) onMessage(ctx context.Context, msg M) error {
	defer func() { // ensures at-least-once semantics
		if m, ok := any(msg).(messageACK); ok {
			m.Ack()
		}
	}()
	h := s.handler
	for i := 0; i < s.numHandlers; i++ {
		if err := h.OnMessage(ctx, msg); err != nil {
			return err
		}
		h = h.Next()
	}
	return nil
}

func (s *Subscriber[M]) stat() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.closed {
		return fmt.Errorf("gcp_subscriber: %w: %s", pubsub.ErrSubscriberClosed, s.subName)
	}
	return nil
}

// WithSubscriptionName sets the subscription name to use when consuming pub/sub events.
//
// The default subscription name is generated using DefaultSubscriptionPrefix and a random 32-bit suffix.
func WithSubscriptionName[M Message](name string) func(*Subscriber[M]) {
	return func(s *Subscriber[M]) {
		s.subName = strings.TrimSpace(name)
	}
}
