package sovereign

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestInitializeTransitionAndAdvanceTick(t *testing.T) {
	t.Parallel()

	store := New(filepath.Join(t.TempDir(), "sovereign.db"))
	ctx := context.Background()

	state, err := store.InitializeSimulation(ctx, RuntimeState{
		SimulationID: "sim-1",
		Mode:         "sovereign",
		Profile:      "workstation",
		Status:       "created",
		CreatedAt:    "2026-05-09T00:00:00Z",
		UpdatedAt:    "2026-05-09T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("InitializeSimulation: %v", err)
	}
	if state.Status != "created" {
		t.Fatalf("expected created status, got %q", state.Status)
	}

	readyAt := time.Date(2026, 5, 9, 0, 1, 0, 0, time.UTC)
	state, err = store.TransitionStatus(ctx, "sim-1", []string{"created"}, "ready", readyAt, "")
	if err != nil {
		t.Fatalf("TransitionStatus: %v", err)
	}
	if state.Status != "ready" {
		t.Fatalf("expected ready status, got %q", state.Status)
	}

	state, err = store.AdvanceTick(ctx, "sim-1", time.Date(2026, 5, 9, 0, 2, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("AdvanceTick: %v", err)
	}
	if state.Status != "running" {
		t.Fatalf("expected running status, got %q", state.Status)
	}
	if state.CurrentTick != 1 {
		t.Fatalf("expected tick 1, got %d", state.CurrentTick)
	}
	if state.LastTickAt == "" {
		t.Fatal("expected last_tick_at to be set")
	}
}

func TestAdvanceTickRejectsNonRunnableState(t *testing.T) {
	t.Parallel()

	store := New(filepath.Join(t.TempDir(), "sovereign.db"))
	ctx := context.Background()

	if _, err := store.InitializeSimulation(ctx, RuntimeState{
		SimulationID: "sim-2",
		Mode:         "sovereign",
		Profile:      "workstation",
		Status:       "failed",
		CreatedAt:    "2026-05-09T00:00:00Z",
		UpdatedAt:    "2026-05-09T00:00:00Z",
	}); err != nil {
		t.Fatalf("InitializeSimulation: %v", err)
	}

	if _, err := store.AdvanceTick(ctx, "sim-2", time.Now().UTC()); err == nil {
		t.Fatal("expected advance tick error for failed state")
	}
}

func TestTruthClaimsAndMemorySummaries(t *testing.T) {
	t.Parallel()

	store := New(filepath.Join(t.TempDir(), "sovereign.db"))
	ctx := context.Background()

	if _, err := store.InitializeSimulation(ctx, RuntimeState{
		SimulationID: "sim-3",
		Mode:         "sovereign",
		Profile:      "workstation",
		Status:       "ready",
		CreatedAt:    "2026-05-09T00:00:00Z",
		UpdatedAt:    "2026-05-09T00:00:00Z",
	}); err != nil {
		t.Fatalf("InitializeSimulation: %v", err)
	}

	claim, err := store.UpsertTruthClaim(ctx, TruthClaim{
		SimulationID: "sim-3",
		ClaimID:      "claim-1",
		ClaimType:    "statement",
		Subject:      "market",
		Source:       "agent:alice",
		SourceKind:   "simulation",
		ClaimText:    "Market sentiment is weakening",
		TruthStatus:  "observed",
		Confidence:   45,
		EvidenceRefs: []string{"doc:1"},
		Version:      1,
		ValidFrom:    "2026-05-09T00:00:00Z",
		UpdatedBy:    "test",
	})
	if err != nil {
		t.Fatalf("UpsertTruthClaim: %v", err)
	}
	if claim.UpdatedAt == "" {
		t.Fatal("expected truth claim updated_at")
	}

	claims, err := store.ListTruthClaims(ctx, "sim-3")
	if err != nil {
		t.Fatalf("ListTruthClaims: %v", err)
	}
	if len(claims) != 1 || claims[0].ClaimID != "claim-1" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
	if claims[0].Subject != "market" || claims[0].SourceKind != "simulation" {
		t.Fatalf("expected richer claim fields, got %#v", claims[0])
	}
	if len(claims[0].EvidenceRefs) != 1 || claims[0].EvidenceRefs[0] != "doc:1" {
		t.Fatalf("expected evidence refs, got %#v", claims[0].EvidenceRefs)
	}

	summary, err := store.SaveMemorySummary(ctx, MemorySummary{
		SimulationID: "sim-3",
		SummaryID:    "summary-1",
		Scope:        "tick_window",
		StartTick:    0,
		EndTick:      1,
		Content:      "Tick 1 summary",
	})
	if err != nil {
		t.Fatalf("SaveMemorySummary: %v", err)
	}
	if summary.CreatedAt == "" {
		t.Fatal("expected summary created_at")
	}

	summaries, err := store.ListMemorySummaries(ctx, "sim-3")
	if err != nil {
		t.Fatalf("ListMemorySummaries: %v", err)
	}
	if len(summaries) != 1 || summaries[0].SummaryID != "summary-1" {
		t.Fatalf("unexpected summaries: %#v", summaries)
	}
}

func TestTruthClaimSchemaMigration(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "sovereign.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS simulation_runtime (
			simulation_id TEXT PRIMARY KEY,
			mode TEXT NOT NULL,
			profile TEXT NOT NULL,
			status TEXT NOT NULL,
			current_tick INTEGER NOT NULL DEFAULT 0,
			last_tick_at TEXT,
			last_error TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS truth_claim (
			simulation_id TEXT NOT NULL,
			claim_id TEXT NOT NULL,
			source TEXT NOT NULL,
			claim_text TEXT NOT NULL,
			truth_status TEXT NOT NULL,
			confidence INTEGER NOT NULL DEFAULT 0,
			evidence_refs TEXT,
			decay_at TEXT,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (simulation_id, claim_id)
		);
	`); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}

	store := New(path)
	ctx := context.Background()
	if _, err := store.InitializeSimulation(ctx, RuntimeState{
		SimulationID: "sim-migrate",
		Mode:         "sovereign",
		Profile:      "workstation",
		Status:       "ready",
		CreatedAt:    "2026-05-09T00:00:00Z",
		UpdatedAt:    "2026-05-09T00:00:00Z",
	}); err != nil {
		t.Fatalf("InitializeSimulation: %v", err)
	}
	if _, err := store.UpsertTruthClaim(ctx, ClaimRecord{
		SimulationID: "sim-migrate",
		ClaimID:      "claim-1",
		ClaimType:    "statement",
		Source:       "agent:migrate",
		SourceKind:   "simulation",
		ClaimText:    "migrated",
		TruthStatus:  "observed",
		Version:      1,
		UpdatedAt:    "2026-05-09T00:00:00Z",
	}); err != nil {
		t.Fatalf("UpsertTruthClaim after migration: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT OR REPLACE INTO truth_claim (simulation_id, claim_id, source, claim_text, truth_status, confidence, evidence_refs, decay_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"sim-migrate", "claim-legacy", "agent:legacy", "legacy", "observed", 10, "doc:legacy", "", "2026-05-09T00:00:00Z"); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}
	got, err := store.GetTruthClaim(ctx, "sim-migrate", "claim-1")
	if err != nil {
		t.Fatalf("GetTruthClaim: %v", err)
	}
	if got.ClaimType != "statement" || got.SourceKind != "simulation" {
		t.Fatalf("expected migrated columns populated, got %#v", got)
	}
	legacy, err := store.GetTruthClaim(ctx, "sim-migrate", "claim-legacy")
	if err != nil {
		t.Fatalf("GetTruthClaim legacy: %v", err)
	}
	if len(legacy.EvidenceRefs) != 1 || legacy.EvidenceRefs[0] != "doc:legacy" {
		t.Fatalf("expected legacy scalar evidence to survive, got %#v", legacy.EvidenceRefs)
	}
}

func TestDeleteSimulationCascadesAncillaryRows(t *testing.T) {
	t.Parallel()

	store := New(filepath.Join(t.TempDir(), "sovereign.db"))
	ctx := context.Background()

	if _, err := store.InitializeSimulation(ctx, RuntimeState{
		SimulationID: "sim-4",
		Mode:         "sovereign",
		Profile:      "workstation",
		Status:       "ready",
		CreatedAt:    "2026-05-09T00:00:00Z",
		UpdatedAt:    "2026-05-09T00:00:00Z",
	}); err != nil {
		t.Fatalf("InitializeSimulation: %v", err)
	}
	if _, err := store.UpsertTruthClaim(ctx, TruthClaim{
		SimulationID: "sim-4",
		ClaimID:      "claim-1",
		Source:       "agent:alice",
		ClaimText:    "something happened",
		TruthStatus:  "observed",
		Confidence:   40,
	}); err != nil {
		t.Fatalf("UpsertTruthClaim: %v", err)
	}
	if _, err := store.SaveMemorySummary(ctx, MemorySummary{
		SimulationID: "sim-4",
		SummaryID:    "summary-1",
		Scope:        "tick_window",
		StartTick:    0,
		EndTick:      1,
		Content:      "summary",
	}); err != nil {
		t.Fatalf("SaveMemorySummary: %v", err)
	}
	if _, err := store.AdvanceTick(ctx, "sim-4", time.Date(2026, 5, 9, 0, 1, 0, 0, time.UTC)); err != nil {
		t.Fatalf("AdvanceTick: %v", err)
	}
	db, err := store.open()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO agent_state (simulation_id, agent_id, state_json, updated_at) VALUES (?, ?, ?, ?)`, "sim-4", 1, `{"mood":"steady"}`, "2026-05-09T00:00:00Z"); err != nil {
		t.Fatalf("insert agent_state: %v", err)
	}
	if err := store.DeleteSimulation(ctx, "sim-4"); err != nil {
		t.Fatalf("DeleteSimulation: %v", err)
	}
	if _, err := store.GetSimulationRuntime(ctx, "sim-4"); !errors.Is(err, ErrSimulationRuntimeNotFound) {
		t.Fatalf("expected runtime not found after delete, got %v", err)
	}
	if _, err := store.ListTruthClaims(ctx, "sim-4"); !errors.Is(err, ErrSimulationRuntimeNotFound) {
		t.Fatalf("expected truth claims not found after delete, got %v", err)
	}
	if _, err := store.ListMemorySummaries(ctx, "sim-4"); !errors.Is(err, ErrSimulationRuntimeNotFound) {
		t.Fatalf("expected memory summaries not found after delete, got %v", err)
	}
	for _, table := range []string{"tick_log", "agent_state", "truth_claim", "memory_summary"} {
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table+` WHERE simulation_id = ?`, "sim-4").Scan(&count); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("expected %s rows to be purged, got %d", table, count)
		}
	}
}
