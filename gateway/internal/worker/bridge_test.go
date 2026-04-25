package worker

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func assertWorkerErrorKind(t *testing.T, err error, want error) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error kind %v, got nil", want)
	}

	var workerErr *Error
	if !errors.As(err, &workerErr) {
		t.Fatalf("expected worker error, got %T: %v", err, err)
	}
	if !errors.Is(err, want) {
		t.Fatalf("expected error kind %v, got %v", want, workerErr.Kind)
	}
}

// startIPCResponder watches ipc_commands/ for a command file and writes a response.
func startIPCResponder(t *testing.T, simulationDir string, respond func(t *testing.T, cmd ipcCommand) []byte) {
	t.Helper()
	commandsDir := filepath.Join(simulationDir, "ipc_commands")
	responsesDir := filepath.Join(simulationDir, "ipc_responses")
	if err := os.MkdirAll(commandsDir, 0o755); err != nil {
		t.Fatalf("mkdir commands dir: %v", err)
	}
	if err := os.MkdirAll(responsesDir, 0o755); err != nil {
		t.Fatalf("mkdir responses dir: %v", err)
	}
	go func() {
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			entries, err := os.ReadDir(commandsDir)
			if err != nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
					continue
				}
				raw, err := os.ReadFile(filepath.Join(commandsDir, entry.Name()))
				if err != nil {
					continue
				}
				var cmd ipcCommand
				if err := json.Unmarshal(raw, &cmd); err != nil {
					t.Errorf("decode ipc command: %v", err)
					return
				}
				responseRaw := respond(t, cmd)
				if err := os.WriteFile(filepath.Join(responsesDir, cmd.CommandID+".json"), responseRaw, 0o644); err != nil {
					t.Errorf("write ipc response: %v", err)
				}
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
}

func TestIntValue(t *testing.T) {
	cases := []struct {
		input any
		want  int
	}{
		{float64(3.7), 3},
		{int(5), 5},
		{int64(99), 99},
		{"nope", 0},
		{nil, 0},
	}
	for _, tc := range cases {
		got := intValue(tc.input)
		if got != tc.want {
			t.Fatalf("intValue(%v) = %d; want %d", tc.input, got, tc.want)
		}
	}
}

func TestTernary(t *testing.T) {
	if got := ternary(true, "yes", "no"); got != "yes" {
		t.Fatalf("ternary(true) = %q; want %q", got, "yes")
	}
	if got := ternary(false, "yes", "no"); got != "no" {
		t.Fatalf("ternary(false) = %q; want %q", got, "no")
	}
}
