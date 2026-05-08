package plugins

// Event is emitted by a guest plugin through the host runtime.
type Event struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

// LogEntry is a guest log line captured by the host runtime.
type LogEntry struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

// Result is the normalized output from a plugin invocation.
type Result struct {
	Output []byte     `json:"output,omitempty"`
	Events []Event    `json:"events,omitempty"`
	Logs   []LogEntry `json:"logs,omitempty"`
}

// CapabilitySet is a normalized set of plugin capabilities.
type CapabilitySet map[string]struct{}

// NewCapabilitySet normalizes a list of capabilities into a set.
func NewCapabilitySet(capabilities []string) CapabilitySet {
	out := CapabilitySet{}
	for _, capability := range capabilities {
		if capability == "" {
			continue
		}
		out[capability] = struct{}{}
	}
	return out
}

// Has returns true when the capability is present.
func (c CapabilitySet) Has(capability string) bool {
	_, ok := c[capability]
	return ok
}
