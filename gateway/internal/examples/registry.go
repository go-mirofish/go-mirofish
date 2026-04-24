package examples

import (
	"fmt"
	"sort"
)

var registry = map[string]registeredExample{
	"product-launch-war-room": {
		def: Definition{
			Key:         "product-launch-war-room",
			Title:       "Product Launch PR War Room",
			Description: "Simulate a 72-hour outrage cycle across stakeholder archetypes and produce a risk report.",
			Tags:        []string{"privacy", "scale", "pr"},
			ConfigPath:  "examples/product-launch-war-room/config.json",
			Profiles:    []string{"small", "medium", "stress"},
		},
		run: runProductLaunchWarRoom,
	},
	"hyperlocal-urban-planning": {
		def: Definition{
			Key:         "hyperlocal-urban-planning",
			Title:       "Hyper-Local Urban Planning Rehearsal",
			Description: "Model coalition emergence for competing town development scenarios.",
			Tags:        []string{"edge", "planning", "civics"},
			ConfigPath:  "examples/hyperlocal-urban-planning/config.json",
			Profiles:    []string{"small", "medium", "stress"},
		},
		run: runUrbanPlanning,
	},
	"zero-day-incident-drill": {
		def: Definition{
			Key:         "zero-day-incident-drill",
			Title:       "Zero-Day Cyber Incident Drill",
			Description: "Model internal and external rumor spread during a breach response.",
			Tags:        []string{"security", "privacy", "communications"},
			ConfigPath:  "examples/zero-day-incident-drill/config.json",
			Profiles:    []string{"small", "medium", "stress"},
		},
		run: runZeroDayIncident,
	},
	"defi-sentiment-stress-test": {
		def: Definition{
			Key:         "defi-sentiment-stress-test",
			Title:       "De-Fi Market Sentiment Stress-Test",
			Description: "Forecast liquidation cascades across trader archetypes and panic thresholds.",
			Tags:        []string{"finance", "privacy", "scale"},
			ConfigPath:  "examples/defi-sentiment-stress-test/config.json",
			Profiles:    []string{"small", "medium", "stress"},
		},
		run: runDefiStress,
	},
	"lost-ending-literary-simulator": {
		def: Definition{
			Key:         "lost-ending-literary-simulator",
			Title:       "Lost Ending Literary Simulator",
			Description: "Simulate character-consistent endings from current chapter state.",
			Tags:        []string{"creative", "memory", "offline"},
			ConfigPath:  "examples/lost-ending-literary-simulator/config.json",
			Profiles:    []string{"small", "medium", "stress"},
		},
		run: runLiteraryEnding,
	},
}

func Definitions() []Definition {
	keys := make([]string, 0, len(registry))
	for key := range registry {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]Definition, 0, len(keys))
	for _, key := range keys {
		out = append(out, registry[key].def)
	}
	return out
}

func Run(key string, opts RunOptions) (RunResult, error) {
	entry, ok := registry[key]
	if !ok {
		return RunResult{}, fmt.Errorf("unknown example: %s", key)
	}
	return entry.run(opts)
}
