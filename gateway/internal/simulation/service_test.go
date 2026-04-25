package simulation

import "testing"

func TestNativeActionRoundParity(t *testing.T) {
	got := toAction(map[string]any{
		"round_num":   3,
		"timestamp":   "2026-01-01T00:00:00Z",
		"platform":    "twitter",
		"agent_id":    7,
		"agent_name":  "Agent 7",
		"action_type": "POST_TWEET",
		"action_args": map[string]any{"content": "hello"},
		"success":     true,
	}, "")
	if got.RoundNum != 3 {
		t.Fatalf("expected round_num parity for native artifacts, got %d", got.RoundNum)
	}
}

func TestNormalizeRunStatusIncludesPlatformProgress(t *testing.T) {
	got := NormalizeRunStatus("sim-1", map[string]any{
		"simulation_id":          "sim-1",
		"runner_status":          "running",
		"twitter_current_round":  2,
		"reddit_current_round":   1,
		"twitter_running":        true,
		"reddit_running":         false,
		"twitter_completed":      false,
		"reddit_completed":       true,
		"twitter_actions_count":  4,
		"reddit_actions_count":   3,
		"total_actions_count":    7,
		"progress_percent":       50,
		"total_simulation_hours": 4,
	})
	if got["twitter_current_round"] != 2 {
		t.Fatalf("expected twitter_current_round to survive normalization, got %#v", got["twitter_current_round"])
	}
	if got["reddit_completed"] != true {
		t.Fatalf("expected reddit_completed to survive normalization, got %#v", got["reddit_completed"])
	}
}
