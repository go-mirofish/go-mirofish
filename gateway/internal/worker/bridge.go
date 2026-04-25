package worker

import "encoding/json"

// SimulationStateReader is implemented by stores that can read simulation and run state.
type SimulationStateReader interface {
	ReadSimulation(simulationID string) (map[string]any, error)
	ReadRunState(simulationID string) (map[string]any, error)
}

// intValue converts common JSON numeric types to int.
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

func ternary(cond bool, ifTrue, ifFalse string) string {
	if cond {
		return ifTrue
	}
	return ifFalse
}
