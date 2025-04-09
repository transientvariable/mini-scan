package gcp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/censys/scan-takehome/pkg/messaging"

	"github.com/transientvariable/anchor"
	"github.com/transientvariable/log-go"

	gcpubsub "cloud.google.com/go/pubsub"
	miniscan "github.com/censys/scan-takehome/pkg"
	gcdpubsub "gocloud.dev/pubsub"
)

var (
	_ messaging.Publisher[constr] = (*publisher[constr])(nil)
)

// Publisher is an implementation of a pubsub.Publisher for pushing messages to a GCP pub/sub topic.
type publisher[M Message] struct {
	closed    bool
	client    *gcpubsub.Client
	mutex     sync.Mutex
	ctx       context.Context
	ctxCancel context.CancelFunc
	projectID string
	topic     *gcpubsub.Topic
	topicID   string
}

// NewPublisher creates a new pubsub.Publisher for pushing messages to a GCP pub/sub topic.
func NewPublisher[M Message](projectID string, topicID string, options ...func(*Option)) (messaging.Publisher[M], error) {
	if projectID = strings.TrimSpace(projectID); projectID == "" {
		return nil, errors.New("gcp_publisher: project ID is required")
	}

	if topicID = strings.TrimSpace(topicID); topicID == "" {
		return nil, errors.New("gcp_publisher: topic ID is required")
	}

	opts := &Option{}
	for _, opt := range options {
		opt(opts)
	}

	p := &publisher[M]{projectID: projectID, topicID: topicID}
	if opts.ctx != nil {
		p.ctx, p.ctxCancel = context.WithCancel(opts.ctx)
	} else {
		p.ctx, p.ctxCancel = context.WithCancel(context.Background())
	}

	client, err := gcpubsub.NewClient(p.ctx, p.projectID)
	if err != nil {
		return nil, fmt.Errorf("gcp_publisher: unable to create client for publisher: %w", err)
	}
	p.client = client
	p.topic = client.Topic(p.topicID)

	log.Info("[gcp_publisher] created client", log.String("projectID", projectID), log.String("topicID", topicID))
	return p, nil
}

// Close releases any resources used by the  GCP publisher.
func (p *publisher[M]) Close() error {
	if p == nil {
		return miniscan.ErrInvalid
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.closed {
		defer p.ctxCancel()
		p.closed = true
		if err := p.client.Close(); err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("gcp_publisher: %w: %s/%s", messaging.ErrPublisherClosed, p.projectID, p.topicID)
}

// ProjectID returns the pub/sub project ID configured for the GCP publisher.
func (p *publisher[M]) ProjectID() string {
	return p.projectID
}

// Publish publishes a message to a GCP pub/sub topic.
func (p *publisher[M]) Publish(msg M) error {
	var d []byte
	switch m := any(msg).(type) {
	case []byte:
		d = m
	case *gcpubsub.Message:
		d = m.Data
	case *gcdpubsub.Message:
		d = m.Body
	default:
		return fmt.Errorf("gcp_publisher: unknown type %T", m)
	}

	log.Info(fmt.Sprintf("[gcp_publisher:publish] publishing message: %s", string(d)))

	_, err := p.topic.Publish(p.ctx, &gcpubsub.Message{Data: d}).Get(p.ctx)
	if err != nil {
		return fmt.Errorf("gcp_publisher: unable to publish message: %w", err)
	}
	return nil
}

// TopicID returns the pub/sub project ID configured for the GCP publisher.
func (p *publisher[M]) TopicID() string {
	return p.topicID
}

// String returns a string representing the current state of the GCP publisher.
func (p *publisher[M]) String() string {
	return string(anchor.ToJSONFormatted(map[string]any{
		"gcp_publisher": map[string]any{
			"project": p.ProjectID(),
			"topic": map[string]any{
				"id":   p.TopicID(),
				"name": p.topic.String(),
			},
		},
	}))
}
