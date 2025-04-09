package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/transientvariable/config-go"
	opensearch "github.com/transientvariable/repository-opensearch-go"
	"sync"

	"github.com/censys/scan-takehome/pkg/messaging"
	"github.com/censys/scan-takehome/pkg/messaging/gcp"
	"github.com/censys/scan-takehome/pkg/messaging/handler"
	"github.com/censys/scan-takehome/pkg/scanning"

	"github.com/transientvariable/log-go"

	miniscan "github.com/censys/scan-takehome/pkg"
)

type dispatcher struct {
	closed     bool
	ctx        context.Context
	ctxCancel  context.CancelFunc
	emulator   *scanning.Emulator
	mutex      sync.Mutex
	publisher  messaging.Publisher[any]
	subscriber messaging.Subscriber[any]
}

func newDispatcher(projectID string, topicID string, subscriptionName string) (*dispatcher, error) {
	d := &dispatcher{}
	d.ctx, d.ctxCancel = context.WithCancel(context.Background())

	// Create publisher
	p, err := gcp.NewPublisher[any](
		projectID,
		topicID,
		gcp.WithContext(d.ctx))
	if err != nil {
		return nil, fmt.Errorf("cmd_dispatch: %w", err)
	}
	d.publisher = p
	log.Info(fmt.Sprintf("[cmd:dispatch] publisher:\n%s", p))

	// Create emulator
	d.emulator = scanning.NewEmulator(scanning.WithPublisher(p))
	log.Info(fmt.Sprintf("[cmd:dispatch] emulator:\n%s", d.emulator))

	// Emulate scans
	if err := d.emulator.Emulate(d.ctx); err != nil {
		return nil, fmt.Errorf("cmd_dispatch: %w", err)
	}

	// Create subscriber
	s, err := gcp.NewSubscriber(
		projectID,
		gcp.WithContext(d.ctx),
		gcp.WithSubscriptionName(subscriptionName))
	if err != nil {
		return nil, fmt.Errorf("cmd_dispatch: %w", err)
	}
	d.subscriber = s

	h, err := handler.NewScanEventHandler(
		handler.WithDocsRepository(opensearch.New(
			opensearch.WithAddresses(config.ValueMustResolve(miniscan.Addresses)),
			opensearch.WithMappingTemplatePath(config.ValueMustResolve(miniscan.MappingTemplatePath)),
			opensearch.WithIndicesPath(config.ValueMustResolve(miniscan.MappingIndicesPath)),
			opensearch.WithMappingCreate(true)),
		))
	if err != nil {
		return nil, fmt.Errorf("cmd_dispatch: %w", err)
	}

	if ok := d.subscriber.Add(h); !ok {
		return nil, fmt.Errorf("cmd_dispatch: failed to add handler to subscriber")
	}

	log.Info(fmt.Sprintf("[cmd:dispatch] subscriber:\n%s", s))
	return d, nil
}

func (d *dispatcher) Run() error {
	log.Info("[cmd:dispatch] running dispatcher...")

	// Listen for emulated scans
	if err := d.subscriber.Subscribe(); err != nil {
		log.Error("[cmd:dispatch]", log.Err(err))
	}
	return nil
}

func (d *dispatcher) Close() error {
	if d == nil {
		return miniscan.ErrInvalid
	}

	d.mutex.Lock()
	defer d.mutex.Unlock()

	if !d.closed {
		defer d.ctxCancel()
		d.closed = true
		err := errors.Join(d.publisher.Close(), d.subscriber.Close())
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("cmd_dispatch: %w", miniscan.ErrClosed)
}
