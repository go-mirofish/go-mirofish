package runinstructions

import (
	"os"
	"testing"
)

func TestBuild_GatewayNativeShape(t *testing.T) {
	t.Setenv("GATEWAY_PUBLIC_BASE_URL", "http://example.test:4000")
	out := Build("sim-1", "/tmp/sim", "/tmp/cfg.json", "/tmp/scripts")
	if out["control_plane"] != "go_gateway" {
		t.Fatalf("control_plane: %v", out["control_plane"])
	}
	if out["worker_runtime"] != "native" {
		t.Fatalf("worker_runtime: %v", out["worker_runtime"])
	}
	if out["config_file"] != "/tmp/cfg.json" {
		t.Fatalf("config_file: %v", out["config_file"])
	}
	inst := out["instructions"].(string)
	if inst == "" {
		t.Fatal("empty instructions")
	}
}

func TestBuild_DefaultGatewayURL(t *testing.T) {
	_ = os.Unsetenv("GATEWAY_PUBLIC_BASE_URL")
	out := Build("x", "/", "/c", "/s")
	if out["gateway_base_url"] != "http://127.0.0.1:3000" {
		t.Fatalf("gateway_base_url: %v", out["gateway_base_url"])
	}
}
