package worker

import (
	"strings"

	"github.com/go-mirofish/go-mirofish/gateway/internal/artifactcontract"
)

func availablePlatforms(platforms []Platform) (twitter, reddit bool) {
	for _, p := range platforms {
		switch p {
		case PlatformTwitter:
			twitter = true
		case PlatformReddit:
			reddit = true
		}
	}
	return twitter, reddit
}

func hasPlatform(platforms []Platform, want Platform) bool {
	for _, p := range platforms {
		if p == want {
			return true
		}
	}
	return false
}

func nativeEnvStatus(status, updatedAt string, twitterAvailable, redditAvailable bool) artifactcontract.EnvStatus {
	return artifactcontract.EnvStatus{
		WorkerProtocolVersion: ProtocolVersion,
		WorkerProtocolName:    ProtocolName,
		TransportRole:         ProtocolRoleWorkerState,
		Status:                status,
		UpdatedAt:             updatedAt,
		TwitterAvailable:      twitterAvailable,
		RedditAvailable:       redditAvailable,
	}
}

// runtimeStatusFromRunnerState maps run_state.json runner_status to the public state.json "status" field.
func runtimeStatusFromRunnerState(runner string) string {
	r := strings.ToLower(strings.TrimSpace(runner))
	if r == "" {
		return "unknown"
	}
	switch r {
	case string(artifactcontract.RunnerRunning),
		string(artifactcontract.RunnerStarting),
		string(artifactcontract.RunnerPaused),
		string(artifactcontract.RunnerStopping),
		string(artifactcontract.RunnerIdle):
		return "running"
	case "succeeded", "success":
		return "completed"
	case string(artifactcontract.RunnerCompleted):
		return "completed"
	case string(artifactcontract.RunnerStopped), "cancelled", "canceled":
		return "stopped"
	case string(artifactcontract.RunnerFailed), "error", "errored":
		return "failed"
	default:
		return r
	}
}
