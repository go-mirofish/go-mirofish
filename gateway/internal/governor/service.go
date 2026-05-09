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

func (s *Service) ListTruthClaims(ctx context.Context, simulationID string) ([]sovereignstore.TruthClaim, error) {
	if !s.Enabled() {
		return nil, errors.New("governor is not enabled")
	}
	return s.store.ListTruthClaims(ctx, simulationID)
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
