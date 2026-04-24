package examples

import (
	"path/filepath"
	"testing"
)

func TestRunAllExamplesAndBenchmarks(t *testing.T) {
	repoRoot := ResolveRepoRoot(filepath.Clean(filepath.Join("..", "..")))
	for _, def := range Definitions() {
		t.Run(def.Key, func(t *testing.T) {
			result, err := Run(def.Key, RunOptions{
				RepoRoot:   repoRoot,
				OutputRoot: filepath.Join(t.TempDir(), def.Key, "artifacts"),
				Profile:    "small",
			})
			if err != nil {
				t.Fatalf("Run(%s): %v", def.Key, err)
			}
			if len(result.Artifacts) == 0 {
				t.Fatalf("expected artifacts for %s", def.Key)
			}
			if result.Profile != "small" {
				t.Fatalf("expected small profile, got %s", result.Profile)
			}
			for name, path := range result.Artifacts {
				if path == "" {
					t.Fatalf("artifact %s path empty", name)
				}
			}
			smoke, err := SmokeOne(def.Key, RunOptions{
				RepoRoot:   repoRoot,
				OutputRoot: filepath.Join(t.TempDir(), def.Key, "smoke"),
				Profile:    "small",
				SmokeOnly:  true,
			})
			if err != nil {
				t.Fatalf("SmokeOne(%s): %v", def.Key, err)
			}
			if !smoke.Success || !smoke.ArtifactSuccess {
				t.Fatalf("expected smoke success for %s, got %#v", def.Key, smoke)
			}
			bench, err := BenchmarkOne(def.Key, RunOptions{
				RepoRoot:   repoRoot,
				OutputRoot: filepath.Join(t.TempDir(), def.Key, "benchmark"),
				Profile:    "small",
			})
			if err != nil {
				t.Fatalf("BenchmarkOne(%s): %v", def.Key, err)
			}
			if !bench.ArtifactSuccess {
				t.Fatalf("expected artifact success for %s", def.Key)
			}
			if bench.Profile != "small" {
				t.Fatalf("expected benchmark profile small, got %s", bench.Profile)
			}
			if bench.Evaluation.Status == "" {
				t.Fatalf("expected benchmark evaluation status")
			}
		})
	}
}
