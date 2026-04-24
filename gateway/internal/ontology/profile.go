package ontology

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ProfileGenerator struct {
	gen   ContentGenerator
	model string
}

func NewProfileGenerator(gen ContentGenerator, model string) *ProfileGenerator {
	return &ProfileGenerator{gen: gen, model: model}
}

func (p *ProfileGenerator) Generate(ctx context.Context, entities []Entity, platform string) ([]AgentProfile, error) {
	_ = ctx
	out := make([]AgentProfile, 0, len(entities))
	for idx, entity := range entities {
		entityType := "Entity"
		if len(entity.Labels) > 0 {
			entityType = entity.Labels[0]
		}
		bio := shorten(entity.Summary, 160)
		persona := entity.Summary
		topics := []string{"General", "Public Affairs"}
		if p.gen != nil {
			prompt := "Return JSON with keys bio, persona, interested_topics for this entity.\n" +
				"Name: " + entity.Name + "\nType: " + entityType + "\nSummary: " + entity.Summary
			if content, err := p.gen.Execute(ctx, "Generate a concise social profile JSON object.", prompt); err == nil {
				var payload map[string]any
				if json.Unmarshal([]byte(content), &payload) == nil {
					if v, ok := payload["bio"].(string); ok && strings.TrimSpace(v) != "" {
						bio = shorten(v, 160)
					}
					if v, ok := payload["persona"].(string); ok && strings.TrimSpace(v) != "" {
						persona = v
					}
					if rawTopics, ok := payload["interested_topics"].([]any); ok {
						var nextTopics []string
						for _, item := range rawTopics {
							if s := strings.TrimSpace(fmt.Sprint(item)); s != "" {
								nextTopics = append(nextTopics, s)
							}
						}
						if len(nextTopics) > 0 {
							topics = nextTopics
						}
					}
				}
			}
		}
		profile := AgentProfile{
			UserID:           idx,
			Username:         generateUsername(entity.Name, idx),
			Name:             entity.Name,
			Bio:              bio,
			Persona:          persona,
			Age:              24 + (idx % 25),
			Gender:           []string{"male", "female", "other"}[idx%3],
			MBTI:             []string{"INTJ", "ENTP", "ISFJ", "ESFP"}[idx%4],
			Country:          []string{"US", "UK", "Japan", "Germany", "India"}[idx%5],
			Profession:       entityType,
			InterestedTopics: topics,
			CreatedAt:        time.Now().Format("2006-01-02"),
		}
		if platform == "twitter" {
			profile.FriendCount = 50 + idx
			profile.FollowerCount = 100 + idx*10
			profile.StatusesCount = 100 + idx*3
		} else {
			profile.Karma = 500 + idx*17
		}
		out = append(out, profile)
	}
	return out, nil
}

func generateUsername(name string, idx int) string {
	sanitized := strings.ToLower(name)
	sanitized = strings.ReplaceAll(sanitized, " ", "_")
	var out strings.Builder
	for _, r := range sanitized {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			out.WriteRune(r)
		}
	}
	base := strings.Trim(out.String(), "_")
	if base == "" {
		base = "agent"
	}
	return fmt.Sprintf("%s_%03d", base, 100+idx)
}
