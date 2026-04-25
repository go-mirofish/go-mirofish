package worker

// NativeRoundScheduler defines deterministic round iteration for the Go-native engine.
// Rounds are 1..TotalRounds inclusive; each round visits every platform in order.
type NativeRoundScheduler struct {
	TotalRounds int
	Platforms   []Platform
	AgentIDs    []int
}

func (s NativeRoundScheduler) valid() bool {
	return s.TotalRounds > 0 && len(s.Platforms) > 0
}

// ForEachRound calls fn with round 1..TotalRounds. Stops if fn returns false.
func (s NativeRoundScheduler) ForEachRound(fn func(round int) bool) {
	if !s.valid() {
		return
	}
	for round := 1; round <= s.TotalRounds; round++ {
		if !fn(round) {
			return
		}
	}
}
