package ontology

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ConfigResolver struct {
	gen ContentGenerator
}

func NewConfigResolver(gen ContentGenerator) *ConfigResolver { return &ConfigResolver{gen: gen} }

func (c *ConfigResolver) Resolve(ctx context.Context, simulationID, projectID, graphID, requirement, model, baseURL string, entities []Entity) (SimulationConfig, error) {
	_ = ctx
	agentConfigs := make([]AgentActivityConfig, 0, len(entities))
	initialPosts := make([]map[string]any, 0, len(entities))
	for idx, entity := range entities {
		entityType := "Entity"
		if len(entity.Labels) > 0 {
			entityType = entity.Labels[0]
		}
		agentConfigs = append(agentConfigs, AgentActivityConfig{
			AgentID:          idx,
			EntityUUID:       entity.UUID,
			EntityName:       entity.Name,
			EntityType:       entityType,
			ActivityLevel:    0.6,
			PostsPerHour:     2,
			CommentsPerHour:  3,
			ActiveHours:      []int{8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22},
			ResponseDelayMin: 5,
			ResponseDelayMax: 60,
			SentimentBias:    0,
			Stance:           "neutral",
			InfluenceWeight:  1,
		})
		if idx < 12 {
			initialPosts = append(initialPosts, map[string]any{
				"content":         shorten(entity.Summary, 240),
				"poster_type":     entityType,
				"poster_agent_id": idx,
			})
		}
	}

	cfg := SimulationConfig{
		SimulationID:          simulationID,
		ProjectID:             projectID,
		GraphID:               graphID,
		SimulationRequirement: requirement,
		TimeConfig: SimulationTimeConfig{
			TotalSimulationHours:      72,
			MinutesPerRound:           60,
			AgentsPerHourMin:          1,
			AgentsPerHourMax:          maxInt(5, len(entities)/2),
			PeakHours:                 []int{19, 20, 21, 22},
			PeakActivityMultiplier:    1.5,
			OffPeakHours:              []int{0, 1, 2, 3, 4, 5},
			OffPeakActivityMultiplier: 0.05,
			MorningHours:              []int{6, 7, 8},
			MorningActivityMultiplier: 0.4,
			WorkHours:                 []int{9, 10, 11, 12, 13, 14, 15, 16, 17, 18},
			WorkActivityMultiplier:    0.7,
		},
		AgentConfigs: agentConfigs,
		EventConfig: EventConfig{
			InitialPosts:       initialPosts,
			ScheduledEvents:    []map[string]any{},
			HotTopics:          []string{"public reaction", "policy response", "community impact"},
			NarrativeDirection: "Go deterministic config pipeline",
		},
		TwitterConfig: PlatformConfig{
			Platform:            "twitter",
			RecencyWeight:       0.4,
			PopularityWeight:    0.3,
			RelevanceWeight:     0.3,
			ViralThreshold:      10,
			EchoChamberStrength: 0.5,
		},
		RedditConfig: PlatformConfig{
			Platform:            "reddit",
			RecencyWeight:       0.4,
			PopularityWeight:    0.3,
			RelevanceWeight:     0.3,
			ViralThreshold:      10,
			EchoChamberStrength: 0.5,
		},
		LLMModel:            model,
		LLMBaseURL:          baseURL,
		GeneratedAt:         time.Now().Format(time.RFC3339),
		GenerationReasoning: "Go deterministic config pipeline",
	}

	if c.gen != nil {
		prompt := "Return JSON with optional keys hot_topics, narrative_direction, total_simulation_hours, minutes_per_round.\nScenario:\n" + requirement
		if content, err := c.gen.Execute(ctx, "Generate compact simulation config JSON.", prompt); err == nil {
			var payload map[string]any
			if json.Unmarshal([]byte(content), &payload) == nil {
				if hours := intValue(payload["total_simulation_hours"]); hours > 0 {
					cfg.TimeConfig.TotalSimulationHours = hours
				}
				if minutes := intValue(payload["minutes_per_round"]); minutes > 0 {
					cfg.TimeConfig.MinutesPerRound = minutes
				}
				if topics, ok := payload["hot_topics"].([]any); ok && len(topics) > 0 {
					cfg.EventConfig.HotTopics = nil
					for _, topic := range topics {
						cfg.EventConfig.HotTopics = append(cfg.EventConfig.HotTopics, strings.TrimSpace(fmt.Sprint(topic)))
					}
				}
				if narrative, ok := payload["narrative_direction"].(string); ok && narrative != "" {
					cfg.EventConfig.NarrativeDirection = narrative
				}
				cfg.GenerationReasoning = "Go LLM-backed config pipeline"
			}
		}
	}

	return cfg, nil
}

func intValue(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case float32:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	case json.Number:
		if parsed, err := v.Int64(); err == nil {
			return int(parsed)
		}
		if parsed, err := v.Float64(); err == nil {
			return int(parsed)
		}
	}
	return 0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
