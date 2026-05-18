package governor

import (
	"context"
	"path/filepath"
	"testing"
	"time"

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

	claim, err := service.RecordClaim(ctx, "sim-2", ClaimInput{
		ClaimID:   "claim-1",
		Source:    "agent:bob",
		ClaimText: "Sentiment is negative",
	})
	if err != nil {
		t.Fatalf("RecordClaim: %v", err)
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

func TestRecordClassifyAndDecayClaims(t *testing.T) {
	t.Parallel()

	store := sovereignstore.New(filepath.Join(t.TempDir(), "sovereign.db"))
	service := NewService(store, DefaultProfile)
	ctx := context.Background()
	now := time.Date(2026, 5, 18, 7, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	if _, err := service.InitializeSimulation(ctx, "sim-claims"); err != nil {
		t.Fatalf("InitializeSimulation: %v", err)
	}

	grounded, err := service.RecordClaim(ctx, "sim-claims", ClaimInput{
		ClaimID:      "claim-grounded",
		Source:       "agent:alice",
		ClaimText:    "Document-backed finding",
		EvidenceRefs: []string{"doc:1"},
	})
	if err != nil {
		t.Fatalf("RecordClaim grounded: %v", err)
	}
	if grounded.TruthStatus != StatusGrounded || grounded.Confidence != 80 {
		t.Fatalf("unexpected grounded claim: %#v", grounded)
	}
	if grounded.DecayAt == "" {
		t.Fatalf("expected default decay schedule on grounded claim, got %#v", grounded)
	}

	contested, err := service.RecordClaim(ctx, "sim-claims", ClaimInput{
		ClaimID:      "claim-contested",
		Source:       "agent:bob",
		ClaimText:    "Conflicting claim",
		EvidenceRefs: []string{"conflict:doc:2"},
	})
	if err != nil {
		t.Fatalf("RecordClaim contested: %v", err)
	}
	if contested.TruthStatus != StatusContested {
		t.Fatalf("unexpected contested status: %#v", contested)
	}

	speculative, err := service.RecordClaim(ctx, "sim-claims", ClaimInput{
		ClaimID:   "claim-speculative",
		Source:    "agent:carol",
		ClaimText: "Ungrounded claim",
	})
	if err != nil {
		t.Fatalf("RecordClaim speculative: %v", err)
	}
	if speculative.TruthStatus != StatusSpeculative {
		t.Fatalf("unexpected speculative status: %#v", speculative)
	}

	groundedLate, err := service.RecordClaim(ctx, "sim-claims", ClaimInput{
		ClaimID:      "claim-grounded-late",
		Source:       "agent:dana",
		ClaimText:    "Grounded long ago",
		EvidenceRefs: []string{"doc:2"},
	})
	if err != nil {
		t.Fatalf("RecordClaim grounded late: %v", err)
	}
	if groundedLate.TruthStatus != StatusGrounded {
		t.Fatalf("unexpected grounded late status: %#v", groundedLate)
	}
	if _, err := service.ScheduleClaimDecay(ctx, "sim-claims", "claim-contested", now.Add(-time.Minute)); err != nil {
		t.Fatalf("ScheduleClaimDecay contested: %v", err)
	}
	if _, err := service.ScheduleClaimDecay(ctx, "sim-claims", "claim-speculative", now.Add(-time.Minute)); err != nil {
		t.Fatalf("ScheduleClaimDecay speculative: %v", err)
	}
	if _, err := service.ScheduleClaimDecay(ctx, "sim-claims", "claim-grounded-late", now.Add(-2*time.Second)); err != nil {
		t.Fatalf("ScheduleClaimDecay grounded late: %v", err)
	}
	if _, err := service.RecordClaim(ctx, "sim-claims", ClaimInput{
		ClaimID:   "claim-speculative",
		Source:    "agent:carol",
		ClaimText: "duplicate",
	}); err == nil {
		t.Fatal("expected duplicate claim rejection")
	}

	decayed, err := service.DecayClaims(ctx, "sim-claims", now)
	if err != nil {
		t.Fatalf("DecayClaims: %v", err)
	}
	if len(decayed) != 3 {
		t.Fatalf("expected 3 decayed claims, got %d", len(decayed))
	}

	observed, err := service.ObserveTruthClaims(ctx, "sim-claims", now.Add(2*time.Second))
	if err != nil {
		t.Fatalf("ObserveTruthClaims: %v", err)
	}
	foundInvalidated := false
	for _, item := range observed {
		if item.ClaimID == "claim-grounded-late" && item.TruthStatus == StatusInvalidated {
			foundInvalidated = true
		}
	}
	if !foundInvalidated {
		t.Fatalf("expected overdue grounded claim to project invalidated on read, got %#v", observed)
	}

	updated, err := service.UpdateClaimConfidence(ctx, "sim-claims", "claim-grounded", 65)
	if err != nil {
		t.Fatalf("UpdateClaimConfidence: %v", err)
	}
	if updated.Confidence != 65 {
		t.Fatalf("expected confidence 65, got %#v", updated.Confidence)
	}
	if updated.Version != 3 {
		t.Fatalf("expected version 3, got %#v", updated.Version)
	}
}
