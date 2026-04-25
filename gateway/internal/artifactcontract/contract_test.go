package artifactcontract

import (
	"strings"
	"testing"
)

func TestRunStateRoundTrip(t *testing.T) {
	raw := `{
		"worker_protocol_version": "1.0",
		"simulation_id": "sim-fixture",
		"runner_status": "running",
		"current_round": 1,
		"total_rounds": 3,
		"simulated_hours": 4,
		"total_simulation_hours": 12,
		"progress_percent": 33.3,
		"twitter_current_round": 1,
		"reddit_current_round": 1,
		"twitter_running": true,
		"reddit_running": true,
		"twitter_actions_count": 2,
		"reddit_actions_count": 1,
		"total_actions_count": 3,
		"updated_at": "2026-01-01T00:00:00"
	}`
	rs, err := ReadRunStateJSON([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	out, err := WriteRunStateJSON(rs)
	if err != nil {
		t.Fatal(err)
	}
	rs2, err := ReadRunStateJSON(out)
	if err != nil {
		t.Fatal(err)
	}
	if rs2.SimulationID != "sim-fixture" || rs2.RunnerStatus != "running" {
		t.Fatalf("unexpected: %#v", rs2)
	}
	if !rs2.TwitterRunning || !rs2.RedditRunning {
		t.Fatalf("expected running platform flags to round-trip, got %#v", rs2)
	}
}

func TestActionsJSONL(t *testing.T) {
	const jl = `{"round_num":1,"timestamp":"t","platform":"twitter","agent_id":1,"agent_name":"a","action_type":"X","action_args":{},"success":true}
`
	evs, err := ParseActionsJSONL(strings.NewReader(jl))
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 1 || evs[0].Platform != "twitter" {
		t.Fatalf("got %#v", evs)
	}
	_, err = FormatActionsJSONL(evs)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIPCRoundTrip(t *testing.T) {
	cmd := `{"worker_protocol_version":"1.0","worker_protocol_name":"go-mirofish-worker","transport_role":"gateway_command","command_id":"c1","command_type":"interview","args":{},"timestamp":"t"}`
	c, err := ReadIPCCommandJSON([]byte(cmd))
	if err != nil {
		t.Fatal(err)
	}
	if c.CommandType != "interview" {
		t.Fatal(c.CommandType)
	}
	res := `{"worker_protocol_version":"1.0","worker_protocol_name":"go-mirofish-worker","transport_role":"worker_response","command_id":"c1","status":"completed","result":{},"timestamp":"t"}`
	r, err := ReadIPCResponseJSON([]byte(res))
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != "completed" {
		t.Fatal(r.Status)
	}
}

func TestEnvStatus(t *testing.T) {
	raw := `{"status":"alive","twitter_available":true,"reddit_available":false,"timestamp":"2026-01-01T00:00:00"}`
	e, err := ReadEnvStatusJSON([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if e.Status != "alive" {
		t.Fatal(e.Status)
	}
	if !e.TwitterAvailable || e.RedditAvailable {
		t.Fatalf("unexpected availability flags: %#v", e)
	}
	if e.UpdatedAt != "2026-01-01T00:00:00" {
		t.Fatalf("expected timestamp fallback into updated_at, got %q", e.UpdatedAt)
	}
	_, err = WriteEnvStatusJSON(e)
	if err != nil {
		t.Fatal(err)
	}
}
