package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	RuntimeWasm     = "wasm"
	RuntimeStarlark = "starlark"

	DefaultWasmModuleFilename     = "plugin.wasm"
	DefaultStarlarkModuleFilename = "plugin.star"

	CapabilityLog       = "log"
	CapabilityEmitEvent = "emit_event"
	CapabilityTimeNow   = "time_now"
	CapabilityRandom    = "random_bytes"
)

// Manifest is the portable plugin contract shared across SDK runtimes.
type Manifest struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Runtime      string   `json:"runtime"`
	APIVersion   uint64   `json:"api_version"`
	Module       string   `json:"module,omitempty"`
	Entry        string   `json:"entry"`
	Allocate     string   `json:"allocate,omitempty"`
	Deallocate   string   `json:"deallocate,omitempty"`
	ABIFunction  string   `json:"abi_function,omitempty"`
	DigestSHA256 string   `json:"digest_sha256,omitempty"`
	SignerID     string   `json:"signer_id,omitempty"`
	Signature    string   `json:"signature_ed25519,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
	Start        []string `json:"start,omitempty"`
}

// ParseManifest parses and validates a plugin manifest.
func ParseManifest(raw []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return m, err
	}
	return m, ValidateManifest(m)
}

// LoadManifestFile reads, parses, and validates a manifest from disk.
func LoadManifestFile(path string) (Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	return ParseManifest(raw)
}

// ValidateManifest validates the portable plugin manifest.
func ValidateManifest(m Manifest) error {
	if strings.TrimSpace(m.Name) == "" {
		return errors.New("plugin manifest: missing name")
	}
	if strings.TrimSpace(m.Version) == "" {
		return errors.New("plugin manifest: missing version")
	}
	if strings.TrimSpace(m.Runtime) == "" {
		return errors.New("plugin manifest: missing runtime")
	}
	if strings.TrimSpace(m.Entry) == "" {
		return errors.New("plugin manifest: missing entry")
	}
	if m.APIVersion == 0 {
		return errors.New("plugin manifest: missing api_version")
	}
	for _, capability := range m.Capabilities {
		switch capability {
		case CapabilityLog, CapabilityEmitEvent, CapabilityTimeNow, CapabilityRandom:
		default:
			return fmt.Errorf("plugin manifest: unsupported capability %q", capability)
		}
	}
	return nil
}

// NormalizeRuntime trims and lowercases a runtime name for comparison.
func NormalizeRuntime(runtime string) string {
	return strings.ToLower(strings.TrimSpace(runtime))
}

// DefaultModuleForRuntime returns the conventional module filename for a runtime.
func DefaultModuleForRuntime(runtime string) (string, error) {
	switch NormalizeRuntime(runtime) {
	case RuntimeWasm:
		return DefaultWasmModuleFilename, nil
	case RuntimeStarlark:
		return DefaultStarlarkModuleFilename, nil
	default:
		return "", fmt.Errorf("plugin manifest: runtime %q requires an explicit module", runtime)
	}
}
