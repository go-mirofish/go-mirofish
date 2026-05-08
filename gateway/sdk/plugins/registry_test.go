package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverDirectoryAndRegistry(t *testing.T) {
	dir := t.TempDir()
	manifest := `{
		"name":"example-greeter",
		"version":"0.1.0",
		"runtime":"wasm",
		"api_version":1,
		"module":"guest.wasm",
		"entry":"run"
	}`
	if err := os.WriteFile(filepath.Join(dir, DefaultManifestFilename), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "guest.wasm"), []byte{0x00, 0x61, 0x73, 0x6d}, 0o644); err != nil {
		t.Fatalf("write guest: %v", err)
	}

	reg, err := DiscoverDirectory(dir)
	if err != nil {
		t.Fatalf("DiscoverDirectory: %v", err)
	}
	if reg.Manifest.Name != "example-greeter" {
		t.Fatalf("unexpected name: %q", reg.Manifest.Name)
	}
	if filepath.Base(reg.ModulePath) != "guest.wasm" {
		t.Fatalf("unexpected module path: %q", reg.ModulePath)
	}

	registry := NewRegistry()
	if _, err := registry.RegisterDirectory(dir); err != nil {
		t.Fatalf("RegisterDirectory: %v", err)
	}
	got, ok := registry.Get("example-greeter")
	if !ok {
		t.Fatalf("expected registered plugin")
	}
	if got.Directory != dir {
		t.Fatalf("unexpected directory: %q", got.Directory)
	}
}

func TestDiscoverDirectoryDefaultsModuleByRuntime(t *testing.T) {
	t.Run("wasm", func(t *testing.T) {
		dir := t.TempDir()
		manifest := `{
			"name":"example-greeter",
			"version":"0.1.0",
			"runtime":"wasm",
			"api_version":1,
			"entry":"run"
		}`
		if err := os.WriteFile(filepath.Join(dir, DefaultManifestFilename), []byte(manifest), 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, DefaultWasmModuleFilename), []byte{0x00, 0x61, 0x73, 0x6d}, 0o644); err != nil {
			t.Fatalf("write guest: %v", err)
		}

		reg, err := DiscoverDirectory(dir)
		if err != nil {
			t.Fatalf("DiscoverDirectory: %v", err)
		}
		if reg.Manifest.Module != DefaultWasmModuleFilename {
			t.Fatalf("expected normalized module %q, got %q", DefaultWasmModuleFilename, reg.Manifest.Module)
		}
	})

	t.Run("starlark", func(t *testing.T) {
		dir := t.TempDir()
		manifest := `{
			"name":"example-rules",
			"version":"0.1.0",
			"runtime":"starlark",
			"api_version":1,
			"entry":"run"
		}`
		if err := os.WriteFile(filepath.Join(dir, DefaultManifestFilename), []byte(manifest), 0o644); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, DefaultStarlarkModuleFilename), []byte("def run(input):\n    return input\n"), 0o644); err != nil {
			t.Fatalf("write plugin.star: %v", err)
		}

		reg, err := DiscoverDirectory(dir)
		if err != nil {
			t.Fatalf("DiscoverDirectory: %v", err)
		}
		if reg.Manifest.Module != DefaultStarlarkModuleFilename {
			t.Fatalf("expected normalized module %q, got %q", DefaultStarlarkModuleFilename, reg.Manifest.Module)
		}
	})
}

func TestDiscoverDirectoryRequiresExplicitModuleForUnknownRuntime(t *testing.T) {
	dir := t.TempDir()
	manifest := `{
		"name":"example-unknown",
		"version":"0.1.0",
		"runtime":"custom",
		"api_version":1,
		"entry":"run"
	}`
	if err := os.WriteFile(filepath.Join(dir, DefaultManifestFilename), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if _, err := DiscoverDirectory(dir); err == nil {
		t.Fatal("expected unknown runtime error")
	}
}
