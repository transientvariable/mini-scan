package scan

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/censys/scan-takehome/pkg/scanning"

	"github.com/transientvariable/anchor"
	"github.com/transientvariable/cadre/ecs"
	"github.com/transientvariable/cadre/validation"
	"github.com/transientvariable/cadre/validation/constraint"
)

// Event represents scan events generated by a scanning agent.
type Event struct {
	*ecs.Base
	Event   *ecs.Event   `json:"event,omitempty"`
	Host    *ecs.Host    `json:"host,omitempty"`
	Network *ecs.Network `json:"network,omitempty"`
	Service *ecs.Service `json:"service,omitempty"`

	eventType string
}

// NewEvent creates a new Event using the provided event type and scan data.
func NewEvent(eventType string, scan *scanning.Scan) (*Event, error) {
	// Timestamp of when the scan was performed/generated from the source.
	// This will be set as the scanEvent.Timestamp.
	ts := time.Unix(scan.Timestamp, 0)

	// Timestamp to indicate when an event was created and is distinct from the event.Timestamp.
	created := time.Now().UTC()

	event := &Event{
		Base: &ecs.Base{
			Timestamp: &ts,
		},
		Event: &ecs.Event{
			Created:  &created,
			Kind:     ecs.EventKindEvent,
			Category: []string{ecs.EventCategoryHost},
			Type:     []string{eventType},
		},
		Host: &ecs.Host{
			IP: scan.Ip,
		},
		Network: &ecs.Network{
			Protocol: scan.Service,
		},
		eventType: eventType,
	}

	// Decode the response data for the event, event.Message should contain the decoded response.
	if err := decodeResponse(scan, event); err != nil {
		return nil, fmt.Errorf("scan_event: could not decode response for scan %s: %w", scan.Ip, err)
	}

	if result := event.validate(); !result.IsValid() {
		return nil, result
	}
	return event, nil
}

// Type returns the event type for the Event.
func (e *Event) Type() string {
	return e.eventType
}

// ID returns unique identifier for the Event.
func (e *Event) ID() string {
	return e.Event.ID
}

// String returns a string representation of the Event.
func (e *Event) String() string {
	em := make(map[string]any)
	em["event"] = e
	em["id"] = e.ID()
	em["type"] = e.Type()
	return string(anchor.ToJSONFormatted(em))
}

// validate performs validation of a Event.
func (e *Event) validate() *validation.Result {
	var validators []validation.Validator
	validators = append(validators, constraint.NotBlank{
		Name:    "eventType",
		Field:   e.eventType,
		Message: "scan_event: event type is required",
	})

	validators = append(validators, constraint.NotBlank{
		Name:    "eventMessage",
		Field:   e.Message,
		Message: "scan_event: event message is required",
	})

	if e.eventType == ecs.EventTypeChange {
		validators = append(validators, constraint.NotBlank{
			Name:    "eventID",
			Field:   e.Event.ID,
			Message: "scan_event: ID is required",
		})
	}
	return validation.Validate(validators...)
}

func decodeResponse(scan *scanning.Scan, event *Event) error {
	switch scan.DataVersion {
	case scanning.V1:
		d := scan.Data.([]byte)
		event.Network.Bytes = int64(len(d))
		event.Message = strings.TrimSpace(string(d))
	case scanning.V2:
		d := scan.Data.(string)
		event.Network.Bytes = int64(len([]byte(d)))
		event.Message = strings.TrimSpace(d)
	default:
		return fmt.Errorf("%w: %d", ErrDataVersionUnsupported, scan.DataVersion)
	}

	if m, err := base64.StdEncoding.DecodeString(event.Message); err == nil {
		event.Message = string(m)
	}
	return nil
}
