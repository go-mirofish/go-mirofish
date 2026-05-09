package governor

import (
	"context"
	"path/filepath"
	"testing"

	sovereignstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/sovereign"
)

func TestInitializeAndAdvanceTick(t *testing.T) {
	t.Parallel()

	store := sovereignstore.New(filepath.Join(t.TempDir(), "sovereign.db"))
	service := NewService(store, DefaultProfile)
	ctx := context.Background()

	state, err := service.InitializeSimulation(ctx, "sim-1")
	if err != nil {
		t.Fatalf("InitializeSimulation: %v", err)
	}
	if state.Status != StatusReady {
		t.Fatalf("expected ready status, got %q", state.Status)
	}
	if state.Profile != DefaultProfile {
		t.Fatalf("expected profile %q, got %q", DefaultProfile, state.Profile)
	}

	state, err = service.AdvanceTick(ctx, "sim-1")
	if err != nil {
		t.Fatalf("AdvanceTick: %v", err)
	}
	if state.Status != StatusRunning {
		t.Fatalf("expected running status, got %q", state.Status)
	}
	if state.CurrentTick != 1 {
		t.Fatalf("expected tick 1, got %d", state.CurrentTick)
	}
}

func TestProfileResolutionAndTruthMemoryFlows(t *testing.T) {
	t.Parallel()

	store := sovereignstore.New(filepath.Join(t.TempDir(), "sovereign.db"))
	service := NewService(store, ProfileARM64Edge)
	ctx := context.Background()

	profile := service.Profile()
	if profile.Name != ProfileARM64Edge {
		t.Fatalf("expected arm64 edge profile, got %#v", profile)
	}
	if profile.MaxParallelAgents != 2 {
		t.Fatalf("expected edge parallel cap 2, got %d", profile.MaxParallelAgents)
	}

	if _, err := service.InitializeSimulation(ctx, "sim-2"); err != nil {
		t.Fatalf("InitializeSimulation: %v", err)
	}
	if _, err := service.AdvanceTick(ctx, "sim-2"); err != nil {
		t.Fatalf("AdvanceTick: %v", err)
	}

	claims, err := service.ListTruthClaims(ctx, "sim-2")
	if err != nil {
		t.Fatalf("ListTruthClaims empty: %v", err)
	}
	if len(claims) != 0 {
		t.Fatalf("expected empty claims, got %#v", claims)
	}

	claim, err := service.UpsertTruthClaim(ctx, "sim-2", sovereignstore.TruthClaim{
		ClaimID:     "claim-1",
		Source:      "agent:bob",
		ClaimText:   "Sentiment is negative",
		TruthStatus: "observed",
		Confidence:  35,
	})
	if err != nil {
		t.Fatalf("UpsertTruthClaim: %v", err)
	}
	if claim.SimulationID != "sim-2" {
		t.Fatalf("unexpected simulation id on claim: %#v", claim)
	}

	summary, err := service.Compact(ctx, "sim-2")
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}
	if summary.EndTick != 1 {
		t.Fatalf("expected compacted end tick 1, got %d", summary.EndTick)
	}
}
