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
	"github.com/censys/scan-takehome/pkg/messaging"
	"github.com/censys/scan-takehome/pkg/messaging/handler"
	"github.com/transientvariable/anchor"
	"github.com/transientvariable/log-go"

	gcdpubsub "gocloud.dev/pubsub"

	_ "gocloud.dev/pubsub/gcppubsub"
)

const (
	// DefaultSubscriptionPrefix defines the default subscription name prefix.
	DefaultSubscriptionPrefix = "sub"

	// SubscriptionURLFormat ...
	SubscriptionURLFormat = "gcppubsub://%s/%s"
)

var (
	_ messaging.Subscriber[any] = (*subscriber)(nil)
)

// subscriber implementation of a pubsub.Subscriber for consuming messages from a GCP pub/sub topic.
type subscriber struct {
	closed      bool
	ctx         context.Context
	ctxCancel   context.CancelFunc
	handler     messaging.Handler[any]
	mutex       sync.Mutex
	numHandlers int
	projectID   string
	sub         *gcdpubsub.Subscription
	subName     string
	subURL      *url.URL
}

// NewSubscriber creates a new pubsub.Subscriber consuming messages from a GCP pub/sub topic.
func NewSubscriber(projectID string, options ...func(*Option)) (messaging.Subscriber[any], error) {
	if projectID = strings.TrimSpace(projectID); projectID == "" {
		return nil, errors.New("gcp_subscriber: project ID is required")
	}

	opts := &Option{}
	for _, opt := range options {
		opt(opts)
	}

	s := &subscriber{projectID: projectID}
	s.Add(handler.NewLogHandler())

	if opts.ctx != nil {
		s.ctx, s.ctxCancel = context.WithCancel(opts.ctx)
	} else {
		s.ctx, s.ctxCancel = context.WithCancel(context.Background())
	}

	if opts.subName == "" {
		s.subName = fmt.Sprintf("%s-%d", DefaultSubscriptionPrefix, int32(rand.Uint32()))
	} else {
		s.subName = opts.subName
	}

	if u, err := url.Parse(fmt.Sprintf(SubscriptionURLFormat, s.projectID, s.subName)); err == nil {
		s.subURL = u
	} else {
		return nil, fmt.Errorf("gcp_subscriber: failed to parse subscription url: %w", err)
	}

	sub, err := gcdpubsub.OpenSubscription(s.ctx, s.subURL.String())
	if err != nil {
		return nil, fmt.Errorf("gcp_subscriber: %w", err)
	}
	s.sub = sub

	_, err = Ready(s.ctx, s.subName, func() (any, error) {
		msg, err := s.sub.Receive(s.ctx)
		if err != nil {
			return nil, err
		}
		msg.Nack()
		return msg, nil
	})
	if err != nil {
		return nil, fmt.Errorf("gcp_subscriber: %w: %s", err, s.subURL.String())
	}
	return s, nil
}

// Add appends the provided handler to the chain of handlers for the GCP subscriber.
func (s *subscriber) Add(handler messaging.Handler[any]) bool {
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
	} else {
		s.handler = handler
	}
	s.numHandlers += 1
	return true
}

// Close releases any resources used by the GCP subscriber.
func (s *subscriber) Close() error {
	if s == nil {
		return miniscan.ErrInvalid
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.closed {
		defer s.ctxCancel()
		s.closed = true
		if err := s.sub.Shutdown(s.ctx); err != nil {
			return fmt.Errorf("gcp_subscriber: failed to shutdown subscription: %w", err)
		}
		return nil
	}
	return fmt.Errorf("gcp_subscriber: %w: %s", messaging.ErrSubscriberClosed, s.subName)
}

// ProjectID returns the pub/sub project ID configured for the GCP subscriber.
func (s *subscriber) ProjectID() string {
	return s.projectID
}

// Subscribe consumes messages from a GCP pub/sub topic and invokes the optional list of handlers.
func (s *subscriber) Subscribe(handlers ...messaging.Handler[any]) error {
	if err := s.stat(); err != nil {
		return err
	}

	for _, h := range handlers {
		s.Add(h)
	}

	log.Info("[gcp_subscriber:subscribe] consuming messages", log.String("url", s.subURL.String()))

	go func() {

		for {
			msg, err := s.sub.Receive(s.ctx)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					log.Error("[gcp_subscriber:subscribe]", log.Err(err))
					msg.Nack()
				}
				break
			}

			select {
			case <-s.ctx.Done():
				if msg != nil {
					msg.Nack()
				}
				log.Info("[gcp_subscriber:subscribe] closing subscription")
				return
			default:
				if err := s.onMessage(s.ctx, msg); err != nil {
					log.Error("[gcp_subscriber:subscribe]", log.Err(err))
					msg.Nack()
					return
				}
				msg.Ack()
			}
		}
	}()
	return nil
}

// SubscriptionName returns the name that uniquely identifies a subscription for the GCP subscriber.
func (s *subscriber) SubscriptionName() string {
	return s.subName
}

// String returns a string representing the current state of the GCP subscriber.
func (s *subscriber) String() string {
	var handlers []string
	if s.numHandlers > 0 {
		h := s.handler
		for i := 0; i < s.numHandlers; i++ {
			handlers = append(handlers, h.Name())
			if h = h.Next(); h == nil {
				break
			}
		}
	}
	return string(anchor.ToJSONFormatted(map[string]any{
		"gcp_subscriber": map[string]any{
			"project_id": s.ProjectID(),
			"subscription": map[string]any{
				"name": s.SubscriptionName(),
				"url":  s.subURL.String(),
			},
			"handlers": map[string]any{
				"count": len(handlers),
				"names": handlers,
			},
		},
	}))
}

func (s *subscriber) onMessage(ctx context.Context, msg *gcdpubsub.Message) error {
	h := s.handler
	for i := 0; i < s.numHandlers; i++ {
		if err := h.OnMessage(ctx, msg); err != nil {
			return err
		}

		if h = h.Next(); h == nil {
			break
		}
	}
	return nil
}

func (s *subscriber) stat() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.closed {
		return fmt.Errorf("gcp_subscriber: %w: %s", messaging.ErrSubscriberClosed, s.subName)
	}
	return nil
}
