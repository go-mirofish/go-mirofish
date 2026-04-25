package artifactcontract

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EnvStatus is env_status.json (worker liveness and env metadata).
type EnvStatus struct {
	WorkerProtocolVersion string         `json:"worker_protocol_version,omitempty"`
	WorkerProtocolName    string         `json:"worker_protocol_name,omitempty"`
	TransportRole         string         `json:"transport_role,omitempty"`
	Status                string         `json:"status"`
	TwitterAvailable      bool           `json:"twitter_available,omitempty"`
	RedditAvailable       bool           `json:"reddit_available,omitempty"`
	Timestamp             string         `json:"timestamp,omitempty"`
	UpdatedAt             string         `json:"updated_at,omitempty"`
	Extra                 map[string]any `json:"-"`
}

type envStatusWire struct {
	WorkerProtocolVersion string `json:"worker_protocol_version,omitempty"`
	WorkerProtocolName    string `json:"worker_protocol_name,omitempty"`
	TransportRole         string `json:"transport_role,omitempty"`
	Status                string `json:"status"`
	TwitterAvailable      bool   `json:"twitter_available,omitempty"`
	RedditAvailable       bool   `json:"reddit_available,omitempty"`
	Timestamp             string `json:"timestamp,omitempty"`
	UpdatedAt             string `json:"updated_at,omitempty"`
}

// ValidateEnvStatus ensures minimum fields.
func ValidateEnvStatus(e *EnvStatus) error {
	if e == nil {
		return fmt.Errorf("env status: nil")
	}
	if strings.TrimSpace(e.Status) == "" {
		return fmt.Errorf("env status: missing status")
	}
	return nil
}

// ReadEnvStatusJSON parses env_status (additional keys preserved in Extra via raw map first pass).
func ReadEnvStatusJSON(raw []byte) (EnvStatus, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return EnvStatus{}, err
	}
	st, _ := m["status"].(string)
	up, _ := m["updated_at"].(string)
	ts, _ := m["timestamp"].(string)
	wv, _ := m["worker_protocol_version"].(string)
	wn, _ := m["worker_protocol_name"].(string)
	tr, _ := m["transport_role"].(string)
	e := EnvStatus{
		WorkerProtocolVersion: wv,
		WorkerProtocolName:    wn,
		TransportRole:         tr,
		Status:                st,
		TwitterAvailable:      boolValue(m["twitter_available"]),
		RedditAvailable:       boolValue(m["reddit_available"]),
		Timestamp:             ts,
		UpdatedAt:             up,
		Extra:                 m,
	}
	if e.UpdatedAt == "" {
		e.UpdatedAt = e.Timestamp
	}
	if err := ValidateEnvStatus(&e); err != nil {
		return e, err
	}
	return e, nil
}

// WriteEnvStatusJSON serializes a minimal contract view.
func WriteEnvStatusJSON(e EnvStatus) ([]byte, error) {
	if err := ValidateEnvStatus(&e); err != nil {
		return nil, err
	}
	w := envStatusWire{
		WorkerProtocolVersion: e.WorkerProtocolVersion,
		WorkerProtocolName:    e.WorkerProtocolName,
		TransportRole:         e.TransportRole,
		Status:                e.Status,
		TwitterAvailable:      e.TwitterAvailable,
		RedditAvailable:       e.RedditAvailable,
		Timestamp:             e.Timestamp,
		UpdatedAt:             e.UpdatedAt,
	}
	return json.MarshalIndent(w, "", "  ")
}

func boolValue(value any) bool {
	got, _ := value.(bool)
	return got
}
