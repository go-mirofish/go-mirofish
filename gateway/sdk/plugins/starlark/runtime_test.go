package starlark

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/go-mirofish/go-mirofish/gateway/sdk/plugins"
)

func TestLoadFromBytesAndInvoke(t *testing.T) {
	rt := NewRuntime()
	prog, err := rt.LoadFromBytes(plugins.Manifest{
		Name:         "starlark-greeter",
		Version:      "0.1.0",
		Runtime:      "starlark",
		APIVersion:   1,
		Module:       "plugin.star",
		Entry:        "run",
		Capabilities: []string{plugins.CapabilityLog, plugins.CapabilityEmitEvent, plugins.CapabilityTimeNow},
	}, []byte(`
def run(input):
    log("hello from starlark")
    emit_event("greeting.created", "payload")
    return "Hello, " + input
`))
	if err != nil {
		t.Fatalf("LoadFromBytes: %v", err)
	}
	result, err := prog.Invoke(context.Background(), []byte("SDK"))
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if got, want := string(result.Output), "Hello, SDK"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
	if len(result.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(result.Events))
	}
	if len(result.Logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(result.Logs))
	}
}

func TestLoadFromDir(t *testing.T) {
	rt := NewRuntime()
	prog, err := rt.LoadFromDir(filepath.Join("..", "..", "..", "..", "examples", "starlark-greeter"))
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	if prog.Manifest.Name != "starlark-greeter" {
		t.Fatalf("unexpected manifest name: %q", prog.Manifest.Name)
	}
}

func TestExecutionStepLimit(t *testing.T) {
	rt := NewRuntime()
	prog, err := rt.LoadFromBytesWithConfig(plugins.Manifest{
		Name:       "step-limit",
		Version:    "0.1.0",
		Runtime:    "starlark",
		APIVersion: 1,
		Module:     "plugin.star",
		Entry:      "run",
	}, []byte(`
def run(input):
    i = 0
    while True:
        i += 1
`), Config{MaxExecutionSteps: 1000})
	if err != nil {
		t.Fatalf("LoadFromBytesWithConfig: %v", err)
	}
	if _, err := prog.Invoke(context.Background(), []byte("SDK")); err == nil {
		t.Fatal("expected step limit error")
	}
}
