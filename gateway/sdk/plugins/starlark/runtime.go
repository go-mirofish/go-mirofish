package starlark

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-mirofish/go-mirofish/gateway/sdk/plugins"
	starlarklib "go.starlark.net/starlark"
)

type Runtime struct{}

type Program struct {
	Manifest plugins.Manifest
	source   []byte
	config   Config
}

type Config struct {
	MaxExecutionSteps uint64
}

type executionState struct {
	events       []plugins.Event
	logs         []plugins.LogEntry
	capabilities plugins.CapabilitySet
}

func NewRuntime() *Runtime {
	return &Runtime{}
}

func (r *Runtime) LoadFromDir(dir string) (*Program, error) {
	reg, err := plugins.DiscoverDirectory(dir)
	if err != nil {
		return nil, err
	}
	if reg.Manifest.Runtime != "starlark" {
		return nil, fmt.Errorf("plugin runtime mismatch: got %q", reg.Manifest.Runtime)
	}
	return r.LoadFromFiles(reg.ManifestPath, reg.ModulePath)
}

func (r *Runtime) LoadFromFiles(manifestPath, sourcePath string) (*Program, error) {
	manifest, err := plugins.LoadManifestFile(manifestPath)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}
	return r.LoadFromBytes(manifest, raw)
}

func (r *Runtime) LoadFromBytes(manifest plugins.Manifest, source []byte) (*Program, error) {
	return r.LoadFromBytesWithConfig(manifest, source, Config{MaxExecutionSteps: 100000})
}

func (r *Runtime) LoadFromBytesWithConfig(manifest plugins.Manifest, source []byte, cfg Config) (*Program, error) {
	if manifest.Runtime != "starlark" {
		return nil, fmt.Errorf("plugin runtime mismatch: got %q", manifest.Runtime)
	}
	if err := plugins.ValidateManifest(manifest); err != nil {
		return nil, err
	}
	if cfg.MaxExecutionSteps == 0 {
		cfg.MaxExecutionSteps = 100000
	}
	return &Program{Manifest: manifest, source: source, config: cfg}, nil
}

func (p *Program) Invoke(ctx context.Context, input []byte) (plugins.Result, error) {
	return p.InvokeWithCapabilities(ctx, input, p.Manifest.Capabilities)
}

func (p *Program) InvokeWithCapabilities(ctx context.Context, input []byte, capabilities []string) (plugins.Result, error) {
	state := &executionState{capabilities: plugins.NewCapabilitySet(capabilities)}
	predeclared := starlarklib.StringDict{
		"log":              starlarklib.NewBuiltin("log", state.logBuiltin),
		"emit_event":       starlarklib.NewBuiltin("emit_event", state.emitEventBuiltin),
		"time_now_unix_ms": starlarklib.NewBuiltin("time_now_unix_ms", state.timeNowBuiltin),
		"random_hex":       starlarklib.NewBuiltin("random_hex", state.randomHexBuiltin),
	}
	thread := &starlarklib.Thread{Name: "go-mirofish-starlark"}
	thread.SetMaxExecutionSteps(p.config.MaxExecutionSteps)
	if ctx != nil {
		done := make(chan struct{})
		defer close(done)
		go func() {
			select {
			case <-ctx.Done():
				thread.Cancel(ctx.Err().Error())
			case <-done:
			}
		}()
	}
	globals, err := starlarklib.ExecFile(thread, p.Manifest.Module, p.source, predeclared)
	if err != nil {
		return plugins.Result{}, err
	}
	entry := globals[p.Manifest.Entry]
	if entry == nil {
		return plugins.Result{}, fmt.Errorf("missing entry %q", p.Manifest.Entry)
	}
	result, err := starlarklib.Call(thread, entry, starlarklib.Tuple{starlarklib.String(input)}, nil)
	if err != nil {
		return plugins.Result{}, err
	}
	var output []byte
	if result != starlarklib.None {
		if s, ok := result.(starlarklib.String); ok {
			output = []byte(string(s))
		} else {
			output = []byte(result.String())
		}
	}
	return plugins.Result{
		Output: output,
		Events: append([]plugins.Event(nil), state.events...),
		Logs:   append([]plugins.LogEntry(nil), state.logs...),
	}, nil
}

func (s *executionState) logBuiltin(_ *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, _ []starlarklib.Tuple) (starlarklib.Value, error) {
	if !s.capabilities.Has(plugins.CapabilityLog) {
		return starlarklib.None, nil
	}
	var msg string
	if err := starlarklib.UnpackArgs("log", args, nil, "message", &msg); err != nil {
		return nil, err
	}
	s.logs = append(s.logs, plugins.LogEntry{Level: "info", Message: msg})
	return starlarklib.None, nil
}

func (s *executionState) emitEventBuiltin(_ *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, _ []starlarklib.Tuple) (starlarklib.Value, error) {
	if !s.capabilities.Has(plugins.CapabilityEmitEvent) {
		return starlarklib.None, nil
	}
	var typ, payload string
	if err := starlarklib.UnpackArgs("emit_event", args, nil, "type", &typ, "payload", &payload); err != nil {
		return nil, err
	}
	s.events = append(s.events, plugins.Event{Type: typ, Payload: payload})
	return starlarklib.None, nil
}

func (s *executionState) timeNowBuiltin(_ *starlarklib.Thread, _ *starlarklib.Builtin, _ starlarklib.Tuple, _ []starlarklib.Tuple) (starlarklib.Value, error) {
	if !s.capabilities.Has(plugins.CapabilityTimeNow) {
		return starlarklib.MakeInt64(0), nil
	}
	return starlarklib.MakeInt64(time.Now().UnixMilli()), nil
}

func (s *executionState) randomHexBuiltin(_ *starlarklib.Thread, _ *starlarklib.Builtin, args starlarklib.Tuple, _ []starlarklib.Tuple) (starlarklib.Value, error) {
	var n int
	if err := starlarklib.UnpackArgs("random_hex", args, nil, "length", &n); err != nil {
		return nil, err
	}
	if n < 0 {
		return nil, fmt.Errorf("random_hex length must be non-negative")
	}
	if n > 4096 {
		return nil, fmt.Errorf("random_hex length too large: %d", n)
	}
	if !s.capabilities.Has(plugins.CapabilityRandom) {
		return starlarklib.String(""), nil
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	return starlarklib.String(hex.EncodeToString(buf)), nil
}

func ExampleDir() string {
	return filepath.Join("examples", "starlark-greeter")
}
