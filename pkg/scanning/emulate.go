package scanning

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/censys/scan-takehome/pkg/messaging"

	"github.com/transientvariable/anchor"
	"github.com/transientvariable/log-go"

	miniscan "github.com/censys/scan-takehome/pkg"
	json "github.com/json-iterator/go"
)

const (
	// EmulatorServices defines the default services to use for generating emulated scans.
	EmulatorServices = "HTTP,SSH,DNS"

	// EmulatorInterval defines the default time duration to wait between generating emulated scans.
	EmulatorInterval = time.Second
)

type encoded struct {
	scan []byte
	err  error
}

type noopPublisher struct{}

func (p *noopPublisher) Publish(msg any) error {
	var s string
	switch m := msg.(type) {
	case []byte:
		s = string(m)
	case fmt.Stringer:
		s = m.String()
	default:
		s = fmt.Sprintf("%v", msg)
	}
	log.Info(fmt.Sprintf("[scan:emulator] received message: %s", s))
	return nil
}

func (p *noopPublisher) Close() error {
	return nil
}

// Emulator can be used for generating emulated scans.
type Emulator struct {
	interval  time.Duration
	publisher messaging.Publisher[any]
	services  []string
}

// NewEmulator creates a new Emulator.
func NewEmulator(options ...func(*Emulator)) *Emulator {
	e := &Emulator{}
	for _, opt := range options {
		opt(e)
	}

	if e.interval <= 0 {
		e.interval = EmulatorInterval
	}

	if len(e.services) == 0 {
		e.services = strings.Split(EmulatorServices, ",")
	}

	if e.publisher == nil {
		e.publisher = &noopPublisher{}
	}
	return e
}

// Emulate runs the Emulator.
func (e *Emulator) Emulate(ctx context.Context) error {
	scans, err := miniscan.Generate(ctx, e.next, e.interval)
	if err != nil {
		return err
	}
	errors := make(chan error)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		for s := range scans {
			err := s.err
			if err == nil {
				err = e.publisher.Publish(s.scan)
			}
			select {
			case errors <- err:
			case <-ctx.Done():
				break
			}
		}
		wg.Done()
	}()

	go func() {
		wg.Wait()
		close(errors)
	}()

	go func() {
		for err := range errors {
			if err != nil {
				log.Error("[scan:emulator]", log.Err(err))
				return
			}
		}
	}()
	return nil
}

// String returns a string representing the current state of the Emulator.
func (e *Emulator) String() string {
	return string(anchor.ToJSONFormatted(map[string]any{
		"scan_emulator": map[string]any{
			"interval": e.interval.String(),
			"services": e.services,
		}}))
}

func (e *Emulator) next() encoded {
	scan := &Scan{
		Ip:        fmt.Sprintf("1.1.1.%d", rand.Intn(255)),
		Port:      uint32(rand.Intn(65535)),
		Service:   e.services[rand.Intn(len(e.services))],
		Timestamp: time.Now().Unix(),
	}

	serviceResp := fmt.Sprintf("service response: %d", rand.Intn(100))
	if rand.Intn(2) == 0 {
		scan.DataVersion = V1
		scan.Data = &V1Data{ResponseBytesUtf8: []byte(serviceResp)}
	} else {
		scan.DataVersion = V2
		scan.Data = &V2Data{ResponseStr: serviceResp}
	}

	s, err := json.Marshal(scan)
	if err != nil {
		return encoded{err: err}
	}
	return encoded{scan: s}
}

// WithInterval sets the time duration to wait between generating emulated scans.
func WithInterval(interval time.Duration) func(*Emulator) {
	return func(e *Emulator) {
		e.interval = interval
	}
}

// WithPublisher sets the pubsub.Publisher to use for publishing emulated scans.
func WithPublisher(publisher messaging.Publisher[any]) func(*Emulator) {
	return func(e *Emulator) {
		e.publisher = publisher
	}
}

// WithServices sets the list of services to use when generating emulated scans.
//
// The default list of services is defined by EmulatorServices.
func WithServices(services ...string) func(*Emulator) {
	return func(e *Emulator) {
		for _, s := range services {
			if s = strings.TrimSpace(s); s != "" {
				e.services = append(e.services, strings.ToUpper(s))
			}
		}
	}
}
