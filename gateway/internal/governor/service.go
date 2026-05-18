package governor

import (
	"context"
	"errors"
	"fmt"
	"time"

	sovereignstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/sovereign"
)

const (
	ModeSovereign           = "sovereign"
	DefaultProfile          = "workstation"
	ProfileConstrainedLocal = "constrained_local"
	ProfileARM64Edge        = "arm64_edge"
	StatusCreated           = "created"
	StatusReady             = "ready"
	StatusRunning           = "running"
	StatusPaused            = "paused"
	StatusStopping          = "stopping"
	StatusStopped           = "stopped"
	StatusCompleted         = "completed"
	StatusFailed            = "failed"

	StatusObserved    = "observed"
	StatusGrounded    = "grounded"
	StatusSpeculative = "speculative"
	StatusContested   = "contested"
	StatusInvalidated = "invalidated"
)

type Profile struct {
	Name              string `json:"name"`
	TickIntervalMs    int    `json:"tick_interval_ms"`
	MaxParallelAgents int    `json:"max_parallel_agents"`
	TruthMode         string `json:"truth_mode"`
	CompactionMode    string `json:"compaction_mode"`
}

type Service struct {
	store   *sovereignstore.Store
	profile string
	now     func() time.Time
}

type ClaimInput struct {
	ClaimID      string
	ClaimType    string
	Subject      string
	Source       string
	SourceKind   string
	ClaimText    string
	EvidenceRefs []string
	ValidFrom    string
	ValidTo      string
	DecayAt      string
	UpdatedBy    string
}

type AuditHook interface {
	OnClaimClassified(context.Context, sovereignstore.ClaimRecord) error
}

func NewService(store *sovereignstore.Store, profile string) *Service {
	if profile == "" {
		profile = DefaultProfile
	}
	return &Service{
		store:   store,
		profile: profile,
		now:     func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Enabled() bool {
	return s != nil && s.store != nil
}

func (s *Service) Profile() Profile {
	return ResolveProfile(s.profile)
}

func (s *Service) InitializeSimulation(ctx context.Context, simulationID string) (sovereignstore.RuntimeState, error) {
	if !s.Enabled() {
		return sovereignstore.RuntimeState{}, errors.New("governor is not enabled")
	}
	now := s.now().Format(time.RFC3339)
	_, err := s.store.InitializeSimulation(ctx, sovereignstore.RuntimeState{
		SimulationID: simulationID,
		Mode:         ModeSovereign,
		Profile:      s.profile,
		Status:       StatusCreated,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return sovereignstore.RuntimeState{}, err
	}
	return s.store.TransitionStatus(ctx, simulationID, []string{StatusCreated}, StatusReady, s.now(), "")
}

func (s *Service) AdoptSimulation(ctx context.Context, state sovereignstore.RuntimeState) (sovereignstore.RuntimeState, error) {
	if !s.Enabled() {
		return sovereignstore.RuntimeState{}, errors.New("governor is not enabled")
	}
	if state.Mode == "" {
		state.Mode = ModeSovereign
	}
	if state.Profile == "" {
		state.Profile = s.profile
	}
	return s.store.InitializeSimulation(ctx, state)
}

func (s *Service) Status(ctx context.Context, simulationID string) (sovereignstore.RuntimeState, error) {
	if !s.Enabled() {
		return sovereignstore.RuntimeState{}, errors.New("governor is not enabled")
	}
	return s.store.GetSimulationRuntime(ctx, simulationID)
}

func (s *Service) AdvanceTick(ctx context.Context, simulationID string) (sovereignstore.RuntimeState, error) {
	if !s.Enabled() {
		return sovereignstore.RuntimeState{}, errors.New("governor is not enabled")
	}
	return s.store.AdvanceTick(ctx, simulationID, s.now())
}

func (s *Service) SetStatus(ctx context.Context, simulationID string, from []string, to string, lastError string) (sovereignstore.RuntimeState, error) {
	if !s.Enabled() {
		return sovereignstore.RuntimeState{}, errors.New("governor is not enabled")
	}
	return s.store.TransitionStatus(ctx, simulationID, from, to, s.now(), lastError)
}

func (s *Service) DeleteSimulation(ctx context.Context, simulationID string) error {
	if !s.Enabled() {
		return nil
	}
	return s.store.DeleteSimulation(ctx, simulationID)
}

func (s *Service) SyncRuntimeState(ctx context.Context, state sovereignstore.RuntimeState) (sovereignstore.RuntimeState, error) {
	if !s.Enabled() {
		return sovereignstore.RuntimeState{}, errors.New("governor is not enabled")
	}
	return s.store.SyncRuntimeState(ctx, state)
}

func (s *Service) UpsertTruthClaim(ctx context.Context, simulationID string, claim sovereignstore.TruthClaim) (sovereignstore.TruthClaim, error) {
	if !s.Enabled() {
		return sovereignstore.TruthClaim{}, errors.New("governor is not enabled")
	}
	claim.SimulationID = simulationID
	return s.store.UpsertTruthClaim(ctx, claim)
}

func (s *Service) RecordClaim(ctx context.Context, simulationID string, input ClaimInput) (sovereignstore.ClaimRecord, error) {
	if !s.Enabled() {
		return sovereignstore.ClaimRecord{}, errors.New("governor is not enabled")
	}
	record := sovereignstore.ClaimRecord{
		SimulationID: simulationID,
		ClaimID:      input.ClaimID,
		ClaimType:    input.ClaimType,
		Subject:      input.Subject,
		Source:       input.Source,
		SourceKind:   input.SourceKind,
		ClaimText:    input.ClaimText,
		EvidenceRefs: append([]string(nil), input.EvidenceRefs...),
		ValidFrom:    input.ValidFrom,
		ValidTo:      input.ValidTo,
		DecayAt:      input.DecayAt,
		UpdatedBy:    input.UpdatedBy,
		Version:      1,
		UpdatedAt:    s.now().Format(time.RFC3339),
	}
	if _, err := s.store.GetTruthClaim(ctx, simulationID, input.ClaimID); err == nil {
		return sovereignstore.ClaimRecord{}, fmt.Errorf("claim %q already exists", input.ClaimID)
	} else if err != nil && !errors.Is(err, sovereignstore.ErrTruthClaimNotFound) {
		return sovereignstore.ClaimRecord{}, err
	}
	classified := s.classifyClaimRecord(record)
	return s.store.CreateTruthClaim(ctx, classified)
}

func (s *Service) UpdateClaimConfidence(ctx context.Context, simulationID, claimID string, confidence int) (sovereignstore.ClaimRecord, error) {
	if !s.Enabled() {
		return sovereignstore.ClaimRecord{}, errors.New("governor is not enabled")
	}
	record, err := s.store.GetTruthClaim(ctx, simulationID, claimID)
	if err != nil {
		return sovereignstore.ClaimRecord{}, err
	}
	record.Confidence = confidence
	record.Version++
	record.UpdatedAt = s.now().Format(time.RFC3339)
	return s.store.UpsertTruthClaim(ctx, record)
}

func (s *Service) ClassifyClaim(ctx context.Context, record sovereignstore.ClaimRecord) (sovereignstore.ClaimRecord, error) {
	if !s.Enabled() {
		return sovereignstore.ClaimRecord{}, errors.New("governor is not enabled")
	}
	record = s.classifyClaimRecord(record)
	return s.store.UpsertTruthClaim(ctx, record)
}

func (s *Service) classifyClaimRecord(record sovereignstore.ClaimRecord) sovereignstore.ClaimRecord {
	if record.ClaimType == "" {
		record.ClaimType = "statement"
	}
	if record.SourceKind == "" {
		record.SourceKind = "simulation"
	}
	switch {
	case hasConflictingEvidence(record.EvidenceRefs):
		record.TruthStatus = StatusContested
		record.Confidence = 20
	case len(record.EvidenceRefs) > 0:
		record.TruthStatus = StatusGrounded
		record.Confidence = 80
	default:
		record.TruthStatus = StatusSpeculative
		record.Confidence = 40
	}
	if record.ValidFrom == "" {
		record.ValidFrom = s.now().Format(time.RFC3339)
	}
	if record.Version == 0 {
		record.Version = 1
	}
	record.UpdatedAt = s.now().Format(time.RFC3339)
	return record
}

func (s *Service) DecayClaims(ctx context.Context, simulationID string, now time.Time) ([]sovereignstore.ClaimRecord, error) {
	if !s.Enabled() {
		return nil, errors.New("governor is not enabled")
	}
	candidates, err := s.store.ListDecayedTruthClaims(ctx, simulationID, now)
	if err != nil {
		return nil, err
	}
	updated := make([]sovereignstore.ClaimRecord, 0, len(candidates))
	for _, item := range candidates {
		item = s.projectClaimDecay(item, now, false)
		item.Version++
		item.UpdatedAt = now.Format(time.RFC3339)
		next, err := s.store.UpsertTruthClaim(ctx, item)
		if err != nil {
			return nil, err
		}
		updated = append(updated, next)
	}
	return updated, nil
}

func (s *Service) ListTruthClaims(ctx context.Context, simulationID string) ([]sovereignstore.TruthClaim, error) {
	if !s.Enabled() {
		return nil, errors.New("governor is not enabled")
	}
	return s.store.ListTruthClaims(ctx, simulationID)
}

func (s *Service) ObserveTruthClaims(ctx context.Context, simulationID string, now time.Time) ([]sovereignstore.ClaimRecord, error) {
	if !s.Enabled() {
		return nil, errors.New("governor is not enabled")
	}
	claims, err := s.store.ListTruthClaims(ctx, simulationID)
	if err != nil {
		return nil, err
	}
	out := make([]sovereignstore.ClaimRecord, 0, len(claims))
	for _, item := range claims {
		out = append(out, s.projectClaimDecay(item, now, true))
	}
	return out, nil
}

func (s *Service) SaveMemorySummary(ctx context.Context, simulationID string, summary sovereignstore.MemorySummary) (sovereignstore.MemorySummary, error) {
	if !s.Enabled() {
		return sovereignstore.MemorySummary{}, errors.New("governor is not enabled")
	}
	summary.SimulationID = simulationID
	return s.store.SaveMemorySummary(ctx, summary)
}

func (s *Service) ListMemorySummaries(ctx context.Context, simulationID string) ([]sovereignstore.MemorySummary, error) {
	if !s.Enabled() {
		return nil, errors.New("governor is not enabled")
	}
	return s.store.ListMemorySummaries(ctx, simulationID)
}

func (s *Service) Compact(ctx context.Context, simulationID string) (sovereignstore.MemorySummary, error) {
	if !s.Enabled() {
		return sovereignstore.MemorySummary{}, errors.New("governor is not enabled")
	}
	runtime, err := s.store.GetSimulationRuntime(ctx, simulationID)
	if err != nil {
		return sovereignstore.MemorySummary{}, err
	}
	endTick := runtime.CurrentTick
	summaryID := fmt.Sprintf("summary-%d", endTick)
	return s.store.SaveMemorySummary(ctx, sovereignstore.MemorySummary{
		SimulationID: simulationID,
		SummaryID:    summaryID,
		Scope:        "tick_window",
		StartTick:    0,
		EndTick:      endTick,
		Content:      fmt.Sprintf("Compacted sovereign summary through tick %d", endTick),
	})
}

func ResolveProfile(name string) Profile {
	switch name {
	case ProfileConstrainedLocal:
		return Profile{
			Name:              ProfileConstrainedLocal,
			TickIntervalMs:    1500,
			MaxParallelAgents: 4,
			TruthMode:         "reduced",
			CompactionMode:    "aggressive",
		}
	case ProfileARM64Edge:
		return Profile{
			Name:              ProfileARM64Edge,
			TickIntervalMs:    2500,
			MaxParallelAgents: 2,
			TruthMode:         "minimal",
			CompactionMode:    "aggressive",
		}
	default:
		return Profile{
			Name:              DefaultProfile,
			TickIntervalMs:    500,
			MaxParallelAgents: 16,
			TruthMode:         "standard",
			CompactionMode:    "normal",
		}
	}
}

func (s *Service) projectClaimDecay(item sovereignstore.ClaimRecord, now time.Time, projected bool) sovereignstore.ClaimRecord {
	if item.DecayAt == "" {
		return item
	}
	decayAt, err := time.Parse(time.RFC3339, item.DecayAt)
	if err != nil || decayAt.After(now) || item.TruthStatus == StatusInvalidated {
		return item
	}
	switch item.TruthStatus {
	case StatusGrounded:
		item.TruthStatus = StatusContested
		if item.Confidence > 50 {
			item.Confidence = 50
		}
		item.DecayAt = now.Add(time.Second).Format(time.RFC3339Nano)
	case StatusSpeculative, StatusContested, StatusObserved:
		item.TruthStatus = StatusInvalidated
		item.Confidence = 0
		item.ValidTo = now.Format(time.RFC3339)
		item.DecayAt = ""
	}
	return item
}

func hasConflictingEvidence(evidenceRefs []string) bool {
	for _, ref := range evidenceRefs {
		if len(ref) >= len("conflict:") && ref[:len("conflict:")] == "conflict:" {
			return true
		}
	}
	return false
}
