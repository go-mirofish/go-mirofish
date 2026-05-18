package sovereign

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

var ErrSimulationRuntimeNotFound = errors.New("sovereign simulation runtime not found")
var ErrTruthClaimNotFound = errors.New("sovereign truth claim not found")

type RuntimeState struct {
	SimulationID string `json:"simulation_id"`
	Mode         string `json:"mode"`
	Profile      string `json:"profile"`
	Status       string `json:"status"`
	CurrentTick  int    `json:"current_tick"`
	LastTickAt   string `json:"last_tick_at,omitempty"`
	LastError    string `json:"last_error,omitempty"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type ClaimRecord struct {
	SimulationID string   `json:"simulation_id"`
	ClaimID      string   `json:"claim_id"`
	ClaimType    string   `json:"claim_type"`
	Subject      string   `json:"subject,omitempty"`
	Source       string   `json:"source"`
	SourceKind   string   `json:"source_kind"`
	ClaimText    string   `json:"claim_text"`
	TruthStatus  string   `json:"truth_status"`
	Confidence   int      `json:"confidence"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
	Version      int      `json:"version"`
	ValidFrom    string   `json:"valid_from,omitempty"`
	ValidTo      string   `json:"valid_to,omitempty"`
	DecayAt      string   `json:"decay_at,omitempty"`
	UpdatedAt    string   `json:"updated_at"`
	UpdatedBy    string   `json:"updated_by,omitempty"`
}

type TruthClaim = ClaimRecord

type MemorySummary struct {
	SimulationID string `json:"simulation_id"`
	SummaryID    string `json:"summary_id"`
	Scope        string `json:"scope"`
	StartTick    int    `json:"start_tick"`
	EndTick      int    `json:"end_tick"`
	Content      string `json:"content"`
	CreatedAt    string `json:"created_at"`
}

type Store struct {
	path    string
	once    sync.Once
	db      *sql.DB
	initErr error
}

func New(path string) *Store {
	return &Store{path: filepath.Clean(path)}
}

func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *Store) InitializeSimulation(ctx context.Context, state RuntimeState) (RuntimeState, error) {
	db, err := s.open()
	if err != nil {
		return RuntimeState{}, err
	}
	if state.SimulationID == "" {
		return RuntimeState{}, fmt.Errorf("sovereign runtime missing simulation_id")
	}
	if state.Mode == "" {
		state.Mode = "sovereign"
	}
	if state.Profile == "" {
		state.Profile = "workstation"
	}
	if state.Status == "" {
		state.Status = "created"
	}
	now := chooseTimestamp(state.CreatedAt, time.Now().UTC())
	if state.CreatedAt == "" {
		state.CreatedAt = now
	}
	if state.UpdatedAt == "" {
		state.UpdatedAt = state.CreatedAt
	}
	_, err = db.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO simulation_runtime
		 (simulation_id, mode, profile, status, current_tick, last_tick_at, last_error, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		state.SimulationID,
		state.Mode,
		state.Profile,
		state.Status,
		state.CurrentTick,
		nullableString(state.LastTickAt),
		nullableString(state.LastError),
		state.CreatedAt,
		state.UpdatedAt,
	)
	if err != nil {
		return RuntimeState{}, err
	}
	return s.GetSimulationRuntime(ctx, state.SimulationID)
}

func (s *Store) DeleteSimulation(ctx context.Context, simulationID string) error {
	db, err := s.open()
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, stmt := range []string{
		`DELETE FROM tick_log WHERE simulation_id = ?`,
		`DELETE FROM agent_state WHERE simulation_id = ?`,
		`DELETE FROM truth_claim WHERE simulation_id = ?`,
		`DELETE FROM memory_summary WHERE simulation_id = ?`,
		`DELETE FROM simulation_runtime WHERE simulation_id = ?`,
	} {
		if _, err := tx.ExecContext(ctx, stmt, simulationID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) GetSimulationRuntime(ctx context.Context, simulationID string) (RuntimeState, error) {
	db, err := s.open()
	if err != nil {
		return RuntimeState{}, err
	}
	var (
		state      RuntimeState
		lastTickAt sql.NullString
		lastError  sql.NullString
	)
	row := db.QueryRowContext(
		ctx,
		`SELECT simulation_id, mode, profile, status, current_tick, last_tick_at, last_error, created_at, updated_at
		   FROM simulation_runtime
		  WHERE simulation_id = ?`,
		simulationID,
	)
	if err := row.Scan(
		&state.SimulationID,
		&state.Mode,
		&state.Profile,
		&state.Status,
		&state.CurrentTick,
		&lastTickAt,
		&lastError,
		&state.CreatedAt,
		&state.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RuntimeState{}, ErrSimulationRuntimeNotFound
		}
		return RuntimeState{}, err
	}
	if lastTickAt.Valid {
		state.LastTickAt = lastTickAt.String
	}
	if lastError.Valid {
		state.LastError = lastError.String
	}
	return state, nil
}

func (s *Store) TransitionStatus(ctx context.Context, simulationID string, from []string, to string, now time.Time, lastError string) (RuntimeState, error) {
	db, err := s.open()
	if err != nil {
		return RuntimeState{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return RuntimeState{}, err
	}
	defer tx.Rollback()

	current, err := getSimulationRuntimeTx(ctx, tx, simulationID)
	if err != nil {
		return RuntimeState{}, err
	}
	if len(from) > 0 && !matchesStatus(current.Status, from) {
		return RuntimeState{}, fmt.Errorf("sovereign runtime status transition denied: %s -> %s", current.Status, to)
	}
	current.Status = to
	current.UpdatedAt = now.UTC().Format(time.RFC3339)
	current.LastError = lastError
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE simulation_runtime
		    SET status = ?, last_error = ?, updated_at = ?
		  WHERE simulation_id = ?`,
		current.Status,
		nullableString(current.LastError),
		current.UpdatedAt,
		simulationID,
	); err != nil {
		return RuntimeState{}, err
	}
	if err := tx.Commit(); err != nil {
		return RuntimeState{}, err
	}
	return current, nil
}

func (s *Store) SyncRuntimeState(ctx context.Context, state RuntimeState) (RuntimeState, error) {
	db, err := s.open()
	if err != nil {
		return RuntimeState{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return RuntimeState{}, err
	}
	defer tx.Rollback()

	current, err := getSimulationRuntimeTx(ctx, tx, state.SimulationID)
	if err != nil {
		return RuntimeState{}, err
	}
	current.Status = state.Status
	current.CurrentTick = state.CurrentTick
	current.LastError = state.LastError
	if state.LastTickAt != "" {
		current.LastTickAt = state.LastTickAt
	}
	current.UpdatedAt = chooseTimestamp(state.UpdatedAt, time.Now().UTC())
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE simulation_runtime
		    SET status = ?, current_tick = ?, last_tick_at = ?, last_error = ?, updated_at = ?
		  WHERE simulation_id = ?`,
		current.Status,
		current.CurrentTick,
		nullableString(current.LastTickAt),
		nullableString(current.LastError),
		current.UpdatedAt,
		current.SimulationID,
	); err != nil {
		return RuntimeState{}, err
	}
	if err := tx.Commit(); err != nil {
		return RuntimeState{}, err
	}
	return current, nil
}

func (s *Store) AdvanceTick(ctx context.Context, simulationID string, now time.Time) (RuntimeState, error) {
	db, err := s.open()
	if err != nil {
		return RuntimeState{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return RuntimeState{}, err
	}
	defer tx.Rollback()

	current, err := getSimulationRuntimeTx(ctx, tx, simulationID)
	if err != nil {
		return RuntimeState{}, err
	}
	if current.Status != "ready" && current.Status != "running" {
		return RuntimeState{}, fmt.Errorf("sovereign runtime cannot advance tick while status=%s", current.Status)
	}
	nextTick := current.CurrentTick + 1
	timestamp := now.UTC().Format(time.RFC3339)
	result, err := tx.ExecContext(
		ctx,
		`UPDATE simulation_runtime
		    SET status = ?, current_tick = ?, last_tick_at = ?, updated_at = ?
		  WHERE simulation_id = ? AND current_tick = ?`,
		"running",
		nextTick,
		timestamp,
		timestamp,
		simulationID,
		current.CurrentTick,
	)
	if err != nil {
		return RuntimeState{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return RuntimeState{}, err
	}
	if affected != 1 {
		return RuntimeState{}, fmt.Errorf("sovereign runtime tick advance conflict")
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO tick_log (simulation_id, tick, phase, status, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		simulationID,
		nextTick,
		"committed",
		"ok",
		timestamp,
	); err != nil {
		return RuntimeState{}, err
	}
	if err := tx.Commit(); err != nil {
		return RuntimeState{}, err
	}
	return s.GetSimulationRuntime(ctx, simulationID)
}

func (s *Store) UpsertTruthClaim(ctx context.Context, claim ClaimRecord) (ClaimRecord, error) {
	db, err := s.open()
	if err != nil {
		return ClaimRecord{}, err
	}
	if claim.SimulationID == "" || claim.ClaimID == "" {
		return ClaimRecord{}, fmt.Errorf("truth claim requires simulation_id and claim_id")
	}
	if _, err := s.GetSimulationRuntime(ctx, claim.SimulationID); err != nil {
		return ClaimRecord{}, err
	}
	if claim.ClaimType == "" {
		claim.ClaimType = "statement"
	}
	if claim.SourceKind == "" {
		claim.SourceKind = "simulation"
	}
	if claim.TruthStatus == "" {
		claim.TruthStatus = "observed"
	}
	if claim.Version == 0 {
		claim.Version = 1
	}
	if claim.ValidFrom == "" {
		claim.ValidFrom = time.Now().UTC().Format(time.RFC3339)
	}
	if claim.UpdatedAt == "" {
		claim.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	evidenceRefs, err := json.Marshal(claim.EvidenceRefs)
	if err != nil {
		return ClaimRecord{}, err
	}
	_, err = db.ExecContext(
		ctx,
		`INSERT INTO truth_claim
		 (simulation_id, claim_id, claim_type, subject, source, source_kind, claim_text, truth_status, confidence, evidence_refs, version, valid_from, valid_to, decay_at, updated_at, updated_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(simulation_id, claim_id) DO UPDATE SET
		   claim_type = excluded.claim_type,
		   subject = excluded.subject,
		   source = excluded.source,
		   source_kind = excluded.source_kind,
		   claim_text = excluded.claim_text,
		   truth_status = excluded.truth_status,
		   confidence = excluded.confidence,
		   evidence_refs = excluded.evidence_refs,
		   version = excluded.version,
		   valid_from = excluded.valid_from,
		   valid_to = excluded.valid_to,
		   decay_at = excluded.decay_at,
		   updated_at = excluded.updated_at,
		   updated_by = excluded.updated_by`,
		claim.SimulationID,
		claim.ClaimID,
		claim.ClaimType,
		nullableString(claim.Subject),
		claim.Source,
		claim.SourceKind,
		claim.ClaimText,
		claim.TruthStatus,
		claim.Confidence,
		string(evidenceRefs),
		claim.Version,
		nullableString(claim.ValidFrom),
		nullableString(claim.ValidTo),
		nullableString(claim.DecayAt),
		claim.UpdatedAt,
		nullableString(claim.UpdatedBy),
	)
	if err != nil {
		return ClaimRecord{}, err
	}
	return claim, nil
}

func (s *Store) CreateTruthClaim(ctx context.Context, claim ClaimRecord) (ClaimRecord, error) {
	db, err := s.open()
	if err != nil {
		return ClaimRecord{}, err
	}
	if claim.SimulationID == "" || claim.ClaimID == "" {
		return ClaimRecord{}, fmt.Errorf("truth claim requires simulation_id and claim_id")
	}
	if _, err := s.GetSimulationRuntime(ctx, claim.SimulationID); err != nil {
		return ClaimRecord{}, err
	}
	if claim.ClaimType == "" {
		claim.ClaimType = "statement"
	}
	if claim.SourceKind == "" {
		claim.SourceKind = "simulation"
	}
	if claim.TruthStatus == "" {
		claim.TruthStatus = "observed"
	}
	if claim.Version == 0 {
		claim.Version = 1
	}
	if claim.ValidFrom == "" {
		claim.ValidFrom = time.Now().UTC().Format(time.RFC3339)
	}
	if claim.UpdatedAt == "" {
		claim.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	evidenceRefs, err := json.Marshal(claim.EvidenceRefs)
	if err != nil {
		return ClaimRecord{}, err
	}
	result, err := db.ExecContext(
		ctx,
		`INSERT INTO truth_claim
		 (simulation_id, claim_id, claim_type, subject, source, source_kind, claim_text, truth_status, confidence, evidence_refs, version, valid_from, valid_to, decay_at, updated_at, updated_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		claim.SimulationID,
		claim.ClaimID,
		claim.ClaimType,
		nullableString(claim.Subject),
		claim.Source,
		claim.SourceKind,
		claim.ClaimText,
		claim.TruthStatus,
		claim.Confidence,
		string(evidenceRefs),
		claim.Version,
		nullableString(claim.ValidFrom),
		nullableString(claim.ValidTo),
		nullableString(claim.DecayAt),
		claim.UpdatedAt,
		nullableString(claim.UpdatedBy),
	)
	if err != nil {
		return ClaimRecord{}, err
	}
	affected, err := result.RowsAffected()
	if err == nil && affected != 1 {
		return ClaimRecord{}, fmt.Errorf("claim %q was not created", claim.ClaimID)
	}
	return claim, nil
}

func (s *Store) GetTruthClaim(ctx context.Context, simulationID, claimID string) (ClaimRecord, error) {
	db, err := s.open()
	if err != nil {
		return ClaimRecord{}, err
	}
	if _, err := s.GetSimulationRuntime(ctx, simulationID); err != nil {
		return ClaimRecord{}, err
	}
	var (
		item         ClaimRecord
		subject      sql.NullString
		evidenceRefs string
		validFrom    sql.NullString
		validTo      sql.NullString
		decayAt      sql.NullString
		updatedBy    sql.NullString
	)
	row := db.QueryRowContext(
		ctx,
		`SELECT simulation_id, claim_id, claim_type, subject, source, source_kind, claim_text, truth_status, confidence, evidence_refs, version, valid_from, valid_to, decay_at, updated_at, updated_by
		   FROM truth_claim
		  WHERE simulation_id = ? AND claim_id = ?`,
		simulationID, claimID,
	)
	if err := row.Scan(
		&item.SimulationID,
		&item.ClaimID,
		&item.ClaimType,
		&subject,
		&item.Source,
		&item.SourceKind,
		&item.ClaimText,
		&item.TruthStatus,
		&item.Confidence,
		&evidenceRefs,
		&item.Version,
		&validFrom,
		&validTo,
		&decayAt,
		&item.UpdatedAt,
		&updatedBy,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ClaimRecord{}, ErrTruthClaimNotFound
		}
		return ClaimRecord{}, err
	}
	decodeClaimRecord(&item, subject, evidenceRefs, validFrom, validTo, decayAt, updatedBy)
	return item, nil
}

func (s *Store) ListTruthClaims(ctx context.Context, simulationID string) ([]ClaimRecord, error) {
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	if _, err := s.GetSimulationRuntime(ctx, simulationID); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(
		ctx,
		`SELECT simulation_id, claim_id, claim_type, subject, source, source_kind, claim_text, truth_status, confidence, evidence_refs, version, valid_from, valid_to, decay_at, updated_at, updated_by
		   FROM truth_claim
		  WHERE simulation_id = ?
		  ORDER BY updated_at DESC, claim_id ASC`,
		simulationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var claims []ClaimRecord
	for rows.Next() {
		var (
			item         ClaimRecord
			subject      sql.NullString
			evidenceRefs string
			validFrom    sql.NullString
			validTo      sql.NullString
			decayAt      sql.NullString
			updatedBy    sql.NullString
		)
		if err := rows.Scan(
			&item.SimulationID,
			&item.ClaimID,
			&item.ClaimType,
			&subject,
			&item.Source,
			&item.SourceKind,
			&item.ClaimText,
			&item.TruthStatus,
			&item.Confidence,
			&evidenceRefs,
			&item.Version,
			&validFrom,
			&validTo,
			&decayAt,
			&item.UpdatedAt,
			&updatedBy,
		); err != nil {
			return nil, err
		}
		decodeClaimRecord(&item, subject, evidenceRefs, validFrom, validTo, decayAt, updatedBy)
		claims = append(claims, item)
	}
	return claims, rows.Err()
}

func (s *Store) ListTruthClaimsByStatus(ctx context.Context, simulationID, status string) ([]ClaimRecord, error) {
	items, err := s.ListTruthClaims(ctx, simulationID)
	if err != nil {
		return nil, err
	}
	if status == "" {
		return items, nil
	}
	filtered := make([]ClaimRecord, 0, len(items))
	for _, item := range items {
		if item.TruthStatus == status {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (s *Store) ListDecayedTruthClaims(ctx context.Context, simulationID string, now time.Time) ([]ClaimRecord, error) {
	items, err := s.ListTruthClaims(ctx, simulationID)
	if err != nil {
		return nil, err
	}
	filtered := make([]ClaimRecord, 0, len(items))
	for _, item := range items {
		if item.DecayAt == "" {
			continue
		}
		decayAt, err := time.Parse(time.RFC3339, item.DecayAt)
		if err != nil {
			continue
		}
		if !decayAt.After(now) && item.TruthStatus != "invalidated" {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (s *Store) SaveMemorySummary(ctx context.Context, summary MemorySummary) (MemorySummary, error) {
	db, err := s.open()
	if err != nil {
		return MemorySummary{}, err
	}
	if summary.SimulationID == "" || summary.SummaryID == "" {
		return MemorySummary{}, fmt.Errorf("memory summary requires simulation_id and summary_id")
	}
	if _, err := s.GetSimulationRuntime(ctx, summary.SimulationID); err != nil {
		return MemorySummary{}, err
	}
	if summary.Scope == "" {
		summary.Scope = "tick_window"
	}
	if summary.CreatedAt == "" {
		summary.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err = db.ExecContext(
		ctx,
		`INSERT INTO memory_summary
		 (simulation_id, summary_id, scope, start_tick, end_tick, content, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(simulation_id, summary_id) DO UPDATE SET
		   scope = excluded.scope,
		   start_tick = excluded.start_tick,
		   end_tick = excluded.end_tick,
		   content = excluded.content,
		   created_at = excluded.created_at`,
		summary.SimulationID,
		summary.SummaryID,
		summary.Scope,
		summary.StartTick,
		summary.EndTick,
		summary.Content,
		summary.CreatedAt,
	)
	if err != nil {
		return MemorySummary{}, err
	}
	return summary, nil
}

func (s *Store) ListMemorySummaries(ctx context.Context, simulationID string) ([]MemorySummary, error) {
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	if _, err := s.GetSimulationRuntime(ctx, simulationID); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(
		ctx,
		`SELECT simulation_id, summary_id, scope, start_tick, end_tick, content, created_at
		   FROM memory_summary
		  WHERE simulation_id = ?
		  ORDER BY end_tick DESC, created_at DESC`,
		simulationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var summaries []MemorySummary
	for rows.Next() {
		var item MemorySummary
		if err := rows.Scan(
			&item.SimulationID,
			&item.SummaryID,
			&item.Scope,
			&item.StartTick,
			&item.EndTick,
			&item.Content,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		summaries = append(summaries, item)
	}
	return summaries, rows.Err()
}

func (s *Store) open() (*sql.DB, error) {
	if s == nil {
		return nil, fmt.Errorf("sovereign store is nil")
	}
	s.once.Do(func() {
		if s.path == "" {
			s.initErr = fmt.Errorf("sovereign sqlite path is empty")
			return
		}
		if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
			s.initErr = err
			return
		}
		db, err := sql.Open("sqlite", s.path)
		if err != nil {
			s.initErr = err
			return
		}
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		if _, err := db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
			_ = db.Close()
			s.initErr = err
			return
		}
		if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
			_ = db.Close()
			s.initErr = err
			return
		}
		if _, err := db.Exec(schemaSQL); err != nil {
			_ = db.Close()
			s.initErr = err
			return
		}
		if err := migrateTruthClaimSchema(db); err != nil {
			_ = db.Close()
			s.initErr = err
			return
		}
		s.db = db
	})
	if s.initErr != nil {
		return nil, s.initErr
	}
	return s.db, nil
}

func getSimulationRuntimeTx(ctx context.Context, tx *sql.Tx, simulationID string) (RuntimeState, error) {
	var (
		state      RuntimeState
		lastTickAt sql.NullString
		lastError  sql.NullString
	)
	row := tx.QueryRowContext(
		ctx,
		`SELECT simulation_id, mode, profile, status, current_tick, last_tick_at, last_error, created_at, updated_at
		   FROM simulation_runtime
		  WHERE simulation_id = ?`,
		simulationID,
	)
	if err := row.Scan(
		&state.SimulationID,
		&state.Mode,
		&state.Profile,
		&state.Status,
		&state.CurrentTick,
		&lastTickAt,
		&lastError,
		&state.CreatedAt,
		&state.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RuntimeState{}, ErrSimulationRuntimeNotFound
		}
		return RuntimeState{}, err
	}
	if lastTickAt.Valid {
		state.LastTickAt = lastTickAt.String
	}
	if lastError.Valid {
		state.LastError = lastError.String
	}
	return state, nil
}

func matchesStatus(current string, allowed []string) bool {
	for _, candidate := range allowed {
		if current == candidate {
			return true
		}
	}
	return false
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func chooseTimestamp(existing string, fallback time.Time) string {
	if existing != "" {
		return existing
	}
	return fallback.Format(time.RFC3339)
}

func decodeClaimRecord(item *ClaimRecord, subject sql.NullString, evidenceRefs string, validFrom, validTo, decayAt, updatedBy sql.NullString) {
	if item == nil {
		return
	}
	if subject.Valid {
		item.Subject = subject.String
	}
	if evidenceRefs != "" {
		if err := json.Unmarshal([]byte(evidenceRefs), &item.EvidenceRefs); err != nil {
			item.EvidenceRefs = []string{evidenceRefs}
		}
	}
	if validFrom.Valid {
		item.ValidFrom = validFrom.String
	}
	if validTo.Valid {
		item.ValidTo = validTo.String
	}
	if decayAt.Valid {
		item.DecayAt = decayAt.String
	}
	if updatedBy.Valid {
		item.UpdatedBy = updatedBy.String
	}
}

func migrateTruthClaimSchema(db *sql.DB) error {
	if db == nil {
		return nil
	}
	for _, column := range []struct {
		name       string
		definition string
	}{
		{name: "claim_type", definition: "TEXT NOT NULL DEFAULT 'statement'"},
		{name: "subject", definition: "TEXT"},
		{name: "source_kind", definition: "TEXT NOT NULL DEFAULT 'simulation'"},
		{name: "version", definition: "INTEGER NOT NULL DEFAULT 1"},
		{name: "valid_from", definition: "TEXT"},
		{name: "valid_to", definition: "TEXT"},
		{name: "updated_by", definition: "TEXT"},
	} {
		if err := ensureColumn(db, "truth_claim", column.name, column.definition); err != nil {
			return err
		}
	}
	return nil
}

func ensureColumn(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid       int
			name      string
			typ       string
			notnull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	_, err = db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + definition)
	return err
}

const schemaSQL = `
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

CREATE TABLE IF NOT EXISTS agent_state (
	simulation_id TEXT NOT NULL,
	agent_id INTEGER NOT NULL,
	state_json TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	PRIMARY KEY (simulation_id, agent_id),
	FOREIGN KEY (simulation_id) REFERENCES simulation_runtime(simulation_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tick_log (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	simulation_id TEXT NOT NULL,
	tick INTEGER NOT NULL,
	phase TEXT NOT NULL,
	status TEXT NOT NULL,
	created_at TEXT NOT NULL,
	FOREIGN KEY (simulation_id) REFERENCES simulation_runtime(simulation_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS truth_claim (
	simulation_id TEXT NOT NULL,
	claim_id TEXT NOT NULL,
	claim_type TEXT NOT NULL DEFAULT 'statement',
	subject TEXT,
	source TEXT NOT NULL,
	source_kind TEXT NOT NULL DEFAULT 'simulation',
	claim_text TEXT NOT NULL,
	truth_status TEXT NOT NULL,
	confidence INTEGER NOT NULL DEFAULT 0,
	evidence_refs TEXT,
	version INTEGER NOT NULL DEFAULT 1,
	valid_from TEXT,
	valid_to TEXT,
	decay_at TEXT,
	updated_at TEXT NOT NULL,
	updated_by TEXT,
	PRIMARY KEY (simulation_id, claim_id),
	FOREIGN KEY (simulation_id) REFERENCES simulation_runtime(simulation_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS memory_summary (
	simulation_id TEXT NOT NULL,
	summary_id TEXT NOT NULL,
	scope TEXT NOT NULL,
	start_tick INTEGER NOT NULL,
	end_tick INTEGER NOT NULL,
	content TEXT NOT NULL,
	created_at TEXT NOT NULL,
	PRIMARY KEY (simulation_id, summary_id),
	FOREIGN KEY (simulation_id) REFERENCES simulation_runtime(simulation_id) ON DELETE CASCADE
);
`
