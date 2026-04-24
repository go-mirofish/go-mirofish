package examples

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

func ResolveRepoRoot(start string) string {
	candidates := []string{
		start,
		filepath.Clean(filepath.Join(start, "..")),
		filepath.Clean(filepath.Join(start, "..", "..")),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "examples", "index.json")); err == nil {
			return candidate
		}
	}
	return start
}

func defaultProfile(profile string) string {
	if strings.TrimSpace(profile) == "" {
		return "medium"
	}
	return profile
}

func resultOutputRoot(opts RunOptions, exampleKey string) string {
	if opts.OutputRoot != "" {
		return opts.OutputRoot
	}
	return filepath.Join(opts.RepoRoot, "examples", exampleKey, "artifacts", defaultProfile(opts.Profile))
}

func benchmarkResultPath(opts RunOptions, exampleKey, profile string) string {
	ts := time.Now().UTC().Format("20060102T150405Z")
	return filepath.Join(opts.RepoRoot, "benchmark", "results", exampleKey, profile, ts+".json")
}

func benchmarkLatestPath(opts RunOptions, exampleKey, profile string) string {
	return filepath.Join(opts.RepoRoot, "benchmark", "results", exampleKey, profile, "latest.json")
}

func smokeResultPath(opts RunOptions) string {
	ts := time.Now().UTC().Format("20060102T150405Z")
	return filepath.Join(opts.RepoRoot, "benchmark", "results", "smoke", ts+".json")
}

func smokeLatestPath(opts RunOptions) string {
	return filepath.Join(opts.RepoRoot, "benchmark", "results", "smoke", "latest.json")
}

func environmentMetadata() BenchmarkEnvironment {
	return BenchmarkEnvironment{
		Timestamp: nowRFC3339(),
		GitCommit: gitCommit(),
		Hostname:  "local-host",
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
		CPUCount:  runtime.NumCPU(),
	}
}

func gitCommit() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func evaluate(th BenchmarkThresholds, startupMS, runtimeMS float64, artifactSuccess, deterministic bool) BenchmarkEvaluation {
	status := "pass"
	var warnings []string
	var failures []string
	if !artifactSuccess {
		failures = append(failures, "artifact generation failed")
	}
	if !deterministic {
		failures = append(failures, "deterministic replay failed")
	}
	if th.StartupFailMS > 0 && startupMS > th.StartupFailMS {
		failures = append(failures, fmt.Sprintf("startup %.2fms exceeds fail threshold %.2fms", startupMS, th.StartupFailMS))
	} else if th.StartupWarnMS > 0 && startupMS > th.StartupWarnMS {
		warnings = append(warnings, fmt.Sprintf("startup %.2fms exceeds warn threshold %.2fms", startupMS, th.StartupWarnMS))
	}
	if th.RuntimeFailMS > 0 && runtimeMS > th.RuntimeFailMS {
		failures = append(failures, fmt.Sprintf("runtime %.2fms exceeds fail threshold %.2fms", runtimeMS, th.RuntimeFailMS))
	} else if th.RuntimeWarnMS > 0 && runtimeMS > th.RuntimeWarnMS {
		warnings = append(warnings, fmt.Sprintf("runtime %.2fms exceeds warn threshold %.2fms", runtimeMS, th.RuntimeWarnMS))
	}
	if len(failures) > 0 {
		status = "fail"
	} else if len(warnings) > 0 {
		status = "warn"
	}
	return BenchmarkEvaluation{Status: status, Warnings: warnings, Failures: failures}
}

func writeResultJSON(path string, payload any) error {
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func copyLatest(path, latest string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := ensureDir(filepath.Dir(latest)); err != nil {
		return err
	}
	return os.WriteFile(latest, raw, 0o644)
}

func readJSONFile[T any](path string, into *T) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, into)
}

func readTextFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func writeJSONArtifact(path string, payload any) error {
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func writeTextArtifact(path, body string) error {
	if err := ensureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

func sentenceSplit(input string) []string {
	fields := strings.FieldsFunc(input, func(r rune) bool {
		return r == '.' || r == '!' || r == '?'
	})
	var out []string
	for _, field := range fields {
		cleaned := strings.TrimSpace(field)
		if cleaned != "" {
			out = append(out, cleaned)
		}
	}
	return out
}

func hashFile(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func mustPath(repoRoot string, parts ...string) string {
	return filepath.Join(append([]string{repoRoot}, parts...)...)
}

func ratio(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

func filepathToSlashMap(values map[string]string) map[string]string {
	return filepathToSlashMapFromRoot("", values)
}

func filepathToSlashMapFromRoot(root string, values map[string]string) map[string]string {
	out := make(map[string]string, len(values))
	for k, v := range values {
		if root != "" {
			if rel, err := filepath.Rel(root, v); err == nil {
				out[k] = filepath.ToSlash(rel)
				continue
			}
		}
		out[k] = filepath.ToSlash(v)
	}
	return out
}

func sortedArtifactHashes(artifacts map[string]string) (map[string]string, error) {
	hashes := make(map[string]string, len(artifacts))
	for _, key := range sortedKeys(artifacts) {
		hash, err := hashFile(artifacts[key])
		if err != nil {
			return nil, fmt.Errorf("hash artifact %s: %w", key, err)
		}
		hashes[key] = hash
	}
	return hashes, nil
}

func lookupThresholds(def Definition, profile, repoRoot string) BenchmarkThresholds {
	var payload struct {
		Profiles map[string]ScenarioProfile `json:"profiles"`
	}
	if err := readJSONFile(filepath.Join(repoRoot, def.ConfigPath), &payload); err != nil {
		return BenchmarkThresholds{}
	}
	if selected, ok := payload.Profiles[profile]; ok {
		return selected.Thresholds
	}
	return BenchmarkThresholds{}
}
