package scan

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/censys/scan-takehome/pkg/scanning"
	"github.com/transientvariable/anchor"

	"cloud.google.com/go/pubsub"
	"github.com/transientvariable/log-go"
	"golang.org/x/exp/rand"

	json "github.com/json-iterator/go"
	gcdpubsub "gocloud.dev/pubsub"
)

const (
	// DefaultEmulatedServices defines a comma-delimited list of services for creating emulated scan events.
	DefaultEmulatedServices = "HTTP,SSH,DNS"

	// DefaultSubscriptionName defines the default subscription name.
	DefaultSubscriptionName = "scan-sub"
)

var (
	subURLFmt = "gcppubsub://%s/%s"
)

// GCPDispatcher defines the properties for an event dispatcher that dispatches events using Google Cloud Pub/Sub.
type GCPDispatcher[T Message] struct {
	closed      bool
	handler     Handler[T]
	mutex       sync.Mutex
	numHandlers int
	projectId   string
	services    []string
	subName     string
	subURL      *url.URL
	topicId     string
}

// NewGCPDispatcher creates a new GCPDispatcher.
func NewGCPDispatcher[T Message](projectId string, topicId string, options ...func(*Option[T])) (*GCPDispatcher[T], error) {
	if projectId = strings.TrimSpace(projectId); projectId == "" {
		return nil, errors.New("gcp_dispatcher: project Id is required")
	}

	if topicId = strings.TrimSpace(topicId); topicId == "" {
		return nil, errors.New("gcp_dispatcher: topic Id is required")
	}

	d := &GCPDispatcher[T]{
		handler:   &LogHandler[T]{},
		projectId: projectId,
		topicId:   topicId,
		services:  strings.Split(DefaultEmulatedServices, ","),
		subName:   DefaultSubscriptionName,
	}

	opts := &Option[T]{}
	for _, opt := range options {
		opt(opts)
	}

	if opts.subName != "" {
		d.subName = opts.subName
	}

	if u, err := url.Parse(fmt.Sprintf(subURLFmt, d.projectId, d.subName)); err == nil {
		d.subURL = u
	} else {
		return nil, fmt.Errorf("gcp_dispatcher: failed to parse subscription url: %w", err)
	}

	if len(opts.services) > 0 {
		d.services = opts.services
	}
	return d, nil
}

// Add appends the provided handler to the chain of handlers for the GCPDispatcher.
func (d *GCPDispatcher[T]) Add(handler Handler[T]) bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	if d.handler != nil {
		last := d.handler
		for {
			if last.Next() == nil {
				break
			}
			last = last.Next()
		}
		last.SetNext(handler)
		d.numHandlers += 1
	} else {
		d.handler = handler
	}
	return true
}

// Publish publishes emulated scan events to a GCP pub/sub topic.
func (d *GCPDispatcher[T]) Publish(ctx context.Context) error {
	client, err := pubsub.NewClient(ctx, d.projectId)
	if err != nil {
		return err
	}

	topic := client.Topic(d.topicId)
	go func() {
		log.Info("[gcp_dispatcher:publish] publishing scans", log.String("topic", topic.String()))

		for range time.Tick(time.Second) {
			scan := &scanning.Scan{
				Ip:        fmt.Sprintf("1.1.1.%d", rand.Intn(255)),
				Port:      uint32(rand.Intn(65535)),
				Service:   d.services[rand.Intn(len(d.services))],
				Timestamp: time.Now().Unix(),
			}

			serviceResp := fmt.Sprintf("service response: %d", rand.Intn(100))
			if rand.Intn(2) == 0 {
				scan.DataVersion = scanning.V1
				scan.Data = &scanning.V1Data{ResponseBytesUtf8: []byte(serviceResp)}
			} else {
				scan.DataVersion = scanning.V2
				scan.Data = &scanning.V2Data{ResponseStr: serviceResp}
			}

			encoded, err := json.Marshal(scan)
			if err != nil {
				log.Error("[gcp_dispatcher:publish]", log.Err(err))
				return
			}

			select {
			case <-ctx.Done():
				log.Info("[gcp_dispatcher:publish] closing publisher", log.String("topic", topic.String()))
				return
			default:
				log.Debug(fmt.Sprintf("[gcp_dispatcher:publish] publishing emulated scan:\n%s", anchor.ToJSONFormatted(scan)))

				if _, err = topic.Publish(ctx, &pubsub.Message{Data: encoded}).Get(ctx); err != nil {
					log.Error("[gcp_dispatcher:publish]", log.Err(err))
					return
				}
			}
		}
	}()
	return nil
}

// Subscribe pulls emulated scan events from a GCP pub/sub topic and invokes the optional list of handlers.
func (d *GCPDispatcher[T]) Subscribe(ctx context.Context, handlers ...Handler[T]) error {
	for _, h := range handlers {
		d.Add(h)
	}

	log.Info("[gcp_dispatcher:subscribe] opening subscription", log.String("url", d.subURL.String()))

	sub, err := gcdpubsub.OpenSubscription(ctx, d.subURL.String())
	if err != nil {
		return err
	}

	go func() {
		defer func(ctx context.Context, sub *gcdpubsub.Subscription) {
			if err := sub.Shutdown(ctx); err != nil {
				log.Error("[gcp_dispatcher:subscribe]", log.Err(err))
			}
		}(ctx, sub)

		for {
			msg, err := sub.Receive(ctx)
			if err != nil {
				log.Error("[gcp_dispatcher:subscribe]", log.Err(err))
				break
			}

			select {
			case <-ctx.Done():
				log.Info("[gcp_dispatcher:subscribe] closing subscription")
				return
			default:
				if err := d.onMessage(ctx, msg); err != nil {
					log.Error("[gcp_dispatcher:subscribe]", log.Err(err))
					return
				}
			}
		}
	}()
	return nil
}

// ProjectID returns the pub/sub project ID configured for the GCPDispatcher.
func (d *GCPDispatcher[T]) ProjectID() string {
	return d.projectId
}

// SubscriptionName returns the name that uniquely identifies a subscription for the GCPDispatcher.
func (d *GCPDispatcher[T]) SubscriptionName() string {
	return d.subName
}

// TopicID returns the pub/sub topic ID configured for the GCPDispatcher.
func (d *GCPDispatcher[T]) TopicID() string {
	return d.topicId
}

// Close releases any resources used by the GCPDispatcher.
func (d *GCPDispatcher[T]) Close() error {
	if d == nil {
		return ErrInvalid
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	if !d.closed {
		d.closed = true
		return nil
	}
	return fmt.Errorf("gcp_dispatcher: %w", ErrClosed)
}

// String returns a string representing the current state of the GCPDispatcher.
func (d *GCPDispatcher[T]) String() string {
	m := make(map[string]any)
	m["gcp_dispatcher"] = map[string]any{
		"emulated_services": d.services,
		"project_id":        d.ProjectID(),
		"topic_id":          d.TopicID(),
		"subscription": map[string]any{
			"name": d.SubscriptionName(),
			"url":  d.subURL.String(),
		},
	}
	return string(anchor.ToJSONFormatted(m))
}

func (d *GCPDispatcher[T]) onMessage(ctx context.Context, msg *gcdpubsub.Message) error {
	defer msg.Ack() // ensures at-least-once semantics
	h := d.handler
	for i := 0; i < d.numHandlers; i++ {
		if err := h.OnMessage(ctx, any(msg)); err != nil {
			return err
		}
		h = h.Next()
	}
	return nil
}
