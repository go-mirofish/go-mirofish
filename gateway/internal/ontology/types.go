package ontology

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
)

//go:embed schemas/*.json
var Schemas embed.FS

type EntityAttribute struct {
	Name        string `json:"name" validate:"required"`
	Type        string `json:"type" validate:"required"`
	Description string `json:"description" validate:"required"`
}

// SourceTarget tolerates both {"source":"X","target":"Y"} and ["X","Y"] from LLMs.
type SourceTarget struct {
	Source string `json:"source" validate:"required"`
	Target string `json:"target" validate:"required"`
}

func (st *SourceTarget) UnmarshalJSON(data []byte) error {
	// Try object form first.
	type plain struct {
		Source string `json:"source"`
		Target string `json:"target"`
	}
	var p plain
	if err := json.Unmarshal(data, &p); err == nil {
		st.Source = p.Source
		st.Target = p.Target
		return nil
	}
	// Try 2-element array form: ["Source","Target"].
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("source_target: expected object or 2-element array, got: %s", string(data))
	}
	if len(arr) < 2 {
		return fmt.Errorf("source_target: array must have at least 2 elements, got %d", len(arr))
	}
	st.Source = arr[0]
	st.Target = arr[1]
	return nil
}

type EntityType struct {
	Name        string            `json:"name" validate:"required"`
	Description string            `json:"description" validate:"required"`
	Attributes  []EntityAttribute `json:"attributes" validate:"required"`
	Examples    []string          `json:"examples,omitempty"`
}

// EdgeType tolerates source_targets as either []object or []array.
// It also accepts source_targets as a single object (not array) by wrapping it.
type EdgeType struct {
	Name          string            `json:"name" validate:"required"`
	Description   string            `json:"description" validate:"required"`
	Attributes    []EntityAttribute `json:"attributes,omitempty"`
	SourceTargets []SourceTarget    `json:"source_targets" validate:"required"`
}

func (et *EdgeType) UnmarshalJSON(data []byte) error {
	type edgePlain struct {
		Name          string            `json:"name"`
		Description   string            `json:"description"`
		Attributes    []EntityAttribute `json:"attributes"`
		SourceTargets json.RawMessage   `json:"source_targets"`
	}
	var p edgePlain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	et.Name = p.Name
	et.Description = p.Description
	et.Attributes = p.Attributes

	if len(p.SourceTargets) == 0 || string(p.SourceTargets) == "null" {
		return nil
	}

	// Try as array first.
	var arr []SourceTarget
	if err := json.Unmarshal(p.SourceTargets, &arr); err == nil {
		et.SourceTargets = arr
		return nil
	}
	// Try as single object.
	var single SourceTarget
	if err := json.Unmarshal(p.SourceTargets, &single); err == nil {
		et.SourceTargets = []SourceTarget{single}
		return nil
	}
	return fmt.Errorf("edge source_targets: cannot parse %s", string(p.SourceTargets))
}

type Ontology struct {
	EntityTypes     []EntityType `json:"entity_types" validate:"required,min=1"`
	EdgeTypes       []EdgeType   `json:"edge_types" validate:"required,min=1"`
	AnalysisSummary string       `json:"analysis_summary"`
}

type BuildInput struct {
	SimulationRequirement string `json:"simulation_requirement" validate:"required"`
	SourceText            string `json:"source_text" validate:"required"`
	AdditionalContext     string `json:"additional_context,omitempty"`
}

type AgentProfile struct {
	UserID           int      `json:"user_id"`
	Username         string   `json:"username"`
	Name             string   `json:"name"`
	Bio              string   `json:"bio"`
	Persona          string   `json:"persona"`
	Age              int      `json:"age,omitempty"`
	Gender           string   `json:"gender,omitempty"`
	MBTI             string   `json:"mbti,omitempty"`
	Country          string   `json:"country,omitempty"`
	Profession       string   `json:"profession,omitempty"`
	InterestedTopics []string `json:"interested_topics,omitempty"`
	Karma            int      `json:"karma,omitempty"`
	FriendCount      int      `json:"friend_count,omitempty"`
	FollowerCount    int      `json:"follower_count,omitempty"`
	StatusesCount    int      `json:"statuses_count,omitempty"`
	CreatedAt        string   `json:"created_at"`
}

type Entity struct {
	UUID       string         `json:"uuid"`
	Name       string         `json:"name"`
	Labels     []string       `json:"labels"`
	Summary    string         `json:"summary"`
	Attributes map[string]any `json:"attributes"`
}

type SimulationTimeConfig struct {
	TotalSimulationHours      int     `json:"total_simulation_hours"`
	MinutesPerRound           int     `json:"minutes_per_round"`
	AgentsPerHourMin          int     `json:"agents_per_hour_min"`
	AgentsPerHourMax          int     `json:"agents_per_hour_max"`
	PeakHours                 []int   `json:"peak_hours"`
	PeakActivityMultiplier    float64 `json:"peak_activity_multiplier"`
	OffPeakHours              []int   `json:"off_peak_hours"`
	OffPeakActivityMultiplier float64 `json:"off_peak_activity_multiplier"`
	MorningHours              []int   `json:"morning_hours"`
	MorningActivityMultiplier float64 `json:"morning_activity_multiplier"`
	WorkHours                 []int   `json:"work_hours"`
	WorkActivityMultiplier    float64 `json:"work_activity_multiplier"`
}

type AgentActivityConfig struct {
	AgentID          int     `json:"agent_id"`
	EntityUUID       string  `json:"entity_uuid"`
	EntityName       string  `json:"entity_name"`
	EntityType       string  `json:"entity_type"`
	ActivityLevel    float64 `json:"activity_level"`
	PostsPerHour     float64 `json:"posts_per_hour"`
	CommentsPerHour  float64 `json:"comments_per_hour"`
	ActiveHours      []int   `json:"active_hours"`
	ResponseDelayMin int     `json:"response_delay_min"`
	ResponseDelayMax int     `json:"response_delay_max"`
	SentimentBias    float64 `json:"sentiment_bias"`
	Stance           string  `json:"stance"`
	InfluenceWeight  float64 `json:"influence_weight"`
}

type EventConfig struct {
	InitialPosts       []map[string]any `json:"initial_posts"`
	ScheduledEvents    []map[string]any `json:"scheduled_events"`
	HotTopics          []string         `json:"hot_topics"`
	NarrativeDirection string           `json:"narrative_direction"`
}

type PlatformConfig struct {
	Platform            string  `json:"platform"`
	RecencyWeight       float64 `json:"recency_weight"`
	PopularityWeight    float64 `json:"popularity_weight"`
	RelevanceWeight     float64 `json:"relevance_weight"`
	ViralThreshold      int     `json:"viral_threshold"`
	EchoChamberStrength float64 `json:"echo_chamber_strength"`
}

type SimulationConfig struct {
	SimulationID          string                `json:"simulation_id"`
	ProjectID             string                `json:"project_id"`
	GraphID               string                `json:"graph_id"`
	SimulationRequirement string                `json:"simulation_requirement"`
	TimeConfig            SimulationTimeConfig  `json:"time_config"`
	AgentConfigs          []AgentActivityConfig `json:"agent_configs"`
	EventConfig           EventConfig           `json:"event_config"`
	TwitterConfig         PlatformConfig        `json:"twitter_config"`
	RedditConfig          PlatformConfig        `json:"reddit_config"`
	LLMModel              string                `json:"llm_model"`
	LLMBaseURL            string                `json:"llm_base_url"`
	GeneratedAt           string                `json:"generated_at"`
	GenerationReasoning   string                `json:"generation_reasoning"`
}

type ContentGenerator interface {
	Execute(ctx context.Context, systemPrompt string, userPrompt string) (string, error)
}

var (
	ErrValidation = errors.New("ontology validation failed")
)
