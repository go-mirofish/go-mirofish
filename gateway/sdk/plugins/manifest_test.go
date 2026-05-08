package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseManifest(t *testing.T) {
	raw := []byte(`{
		"name":"example-greeter",
		"version":"0.1.0",
		"runtime":"wasm",
		"api_version":1,
		"entry":"greeting",
		"allocate":"allocate",
		"deallocate":"deallocate",
		"capabilities":["log","emit_event"]
	}`)

	manifest, err := ParseManifest(raw)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if manifest.Name != "example-greeter" {
		t.Fatalf("unexpected name: %q", manifest.Name)
	}
}

func TestValidateManifestRejectsUnknownCapability(t *testing.T) {
	err := ValidateManifest(Manifest{
		Name:         "bad",
		Version:      "0.1.0",
		Runtime:      "wasm",
		APIVersion:   1,
		Entry:        "run",
		Capabilities: []string{"database"},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLoadManifestFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.json")
	raw := []byte(`{
		"name":"file-plugin",
		"version":"0.1.0",
		"runtime":"wasm",
		"api_version":1,
		"entry":"run"
	}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	manifest, err := LoadManifestFile(path)
	if err != nil {
		t.Fatalf("LoadManifestFile: %v", err)
	}
	if manifest.Name != "file-plugin" {
		t.Fatalf("unexpected name: %q", manifest.Name)
	}
}
