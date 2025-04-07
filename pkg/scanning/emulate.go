package scanning

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/censys/scan-takehome/pkg/pubsub"

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

type ScanEmulator struct {
	interval time.Duration
	services []string
}

func NewScanEmulator[T any](publisher pubsub.Publisher[T], options ...func(*ScanEmulator)) *ScanEmulator {
	e := &ScanEmulator{}
	for _, opt := range options {
		opt(e)
	}

	if e.interval <= 0 {
		e.interval = EmulatorInterval
	}

	if len(e.services) == 0 {
		e.services = strings.Split(EmulatorServices, ",")
	}
	return &ScanEmulator{}
}

func (e *ScanEmulator) Emulate(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err := miniscan.Generate(ctx, e.next, e.interval)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)

	return nil
}

func (e *ScanEmulator) next() encoded {
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
func WithInterval(interval time.Duration) func(*ScanEmulator) {
	return func(e *ScanEmulator) {
		e.interval = interval
	}
}

// WithServices sets the list of services to use when generating emulated scans.
//
// The default list of services is defined by EmulatorServices.
func WithServices(services ...string) func(*ScanEmulator) {
	return func(e *ScanEmulator) {
		for _, s := range services {
			if s = strings.TrimSpace(s); s != "" {
				e.services = append(e.services, strings.ToUpper(s))
			}
		}
	}
}
