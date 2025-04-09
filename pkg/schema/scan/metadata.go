package scan

import (
	"encoding/base64"
	"fmt"
	"github.com/transientvariable/log-go"
	"reflect"
	"strings"
	"time"

	"github.com/censys/scan-takehome/pkg/scanning"

	"github.com/minio/sha256-simd"
	"github.com/transientvariable/anchor"
	"github.com/transientvariable/cadre/ecs"
	"github.com/transientvariable/cadre/validation"
	"github.com/transientvariable/cadre/validation/constraint"

	json "github.com/json-iterator/go"
)

const (
	DataKeyV1 = "response_str"
	DataKeyV2 = "response_bytes_utf8"
)

// Metadata captures details for a scan.
type Metadata struct {
	*ecs.Base
	Network *ecs.Network `json:"network,omitempty"`
	Source  *ecs.Source  `json:"source,omitempty"`

	// ID is a SHA256 hash that unique identifies scan metadata.
	//
	// Input for the hash sum is in the format: <Ip>:<Port>:<Service>:<Timestamp>
	ID string `json:"id"`
}

// NewMetadata creates Metadata from the provided scan.
func NewMetadata(scan *scanning.Scan) (*Metadata, error) {
	// Timestamp of when the scan was performed/generated from the source.
	// This will be set as the Metadata.Timestamp.
	ts := time.Unix(scan.Timestamp, 0)

	m := &Metadata{
		ID: scanID(scan),
		Base: &ecs.Base{
			Timestamp: &ts,
		},
		Network: &ecs.Network{
			Protocol: scan.Service,
		},
		Source: &ecs.Source{
			IP:   scan.Ip,
			Port: int64(scan.Port),
		},
	}

	// Decode the response data for the event, metadata.Base.Message should contain the decoded response.
	if err := decodeScanMessage(scan, m); err != nil {
		return nil, fmt.Errorf("scan_metadata: could not decode response for scan %s: %w", scan.Ip, err)
	}

	if result := m.validate(); !result.IsValid() {
		return nil, result
	}
	return m, nil
}

// ToMap converts the Metadata fields and their values to a map.
func (m *Metadata) ToMap() (map[string]any, error) {
	var s map[string]any
	if err := json.Unmarshal(anchor.ToJSON(s), &s); err != nil {
		return nil, err
	}
	return s, nil
}

// String returns a string representation of the scan Metadata.
func (m *Metadata) String() string {
	return string(anchor.ToJSONFormatted(map[string]any{
		"id":       m.ID,
		"metadata": m,
	}))
}

// validate performs validation of a scan Metadata.
func (m *Metadata) validate() *validation.Result {
	var validators []validation.Validator
	validators = append(validators, constraint.NotBlank{
		Name:    "metadataID",
		Field:   m.ID,
		Message: "metadata: ID is required",
	})

	validators = append(validators, constraint.NotBlank{
		Name:    "metadataMessage",
		Field:   m.Message,
		Message: "metadata: metadata message is required",
	})
	return validation.Validate(validators...)
}

// decodeScanMessage decodes the raw message data for the scan and applies the result to Metadata.Base.Message.
func decodeScanMessage(scan *scanning.Scan, metadata *Metadata) error {
	var d any
	switch v := scan.Data.(type) {
	case map[string]any:
		if v1, ok := v[DataKeyV1]; ok {
			d = v1
		} else if v2, ok := v[DataKeyV2]; ok {
			d = v2
		}
		break
	default:
		return fmt.Errorf("scan_metadata: %w: data type is not support: %v", ErrInvalidData, reflect.TypeOf(d))
	}

	if s, ok := d.(string); ok {
		metadata.Network.Bytes = int64(len(s))
		s = strings.TrimSpace(s)
		if m, err := base64.StdEncoding.DecodeString(s); err == nil {
			s = string(m)
		}
		metadata.Message = s
	} else {
		return fmt.Errorf("scan_metadata: %w: unexpected data type: %v", ErrInvalidData, reflect.TypeOf(d))
	}

	log.Trace("[scan_metadata] decoded scan message",
		log.String("m", metadata.Message),
		log.Int("v", scan.DataVersion))
	return nil
}

// scanID generates a SHA256 digest from a combination of the IP, port, and service from the scan attributes.
//
// Decoded format: <Ip>:<Port>:<Service>:<Timestamp>
func scanID(s *scanning.Scan) string {
	seed := fmt.Sprintf("%s:%d:%s:%d", s.Ip, s.Port, s.Service, s.Timestamp)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(seed)))
}
