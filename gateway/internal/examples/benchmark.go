package examples

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func BenchmarkOne(key string, opts RunOptions) (BenchmarkResult, error) {
	entry, ok := registry[key]
	if !ok {
		return BenchmarkResult{}, fmt.Errorf("unknown example: %s", key)
	}

	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	start := time.Now()
	result, err := entry.run(opts)
	if err != nil {
		return BenchmarkResult{}, err
	}
	startupLatencyMS := time.Since(start).Seconds() * 1000

	replay, err := entry.run(RunOptions{
		RepoRoot:   opts.RepoRoot,
		OutputRoot: filepath.Join(os.TempDir(), "go-mirofish-replay", key, defaultProfile(opts.Profile)),
		Profile:    opts.Profile,
	})
	if err != nil {
		return BenchmarkResult{}, fmt.Errorf("replay %s: %w", key, err)
	}
	hashesA, err := sortedArtifactHashes(result.Artifacts)
	if err != nil {
		return BenchmarkResult{}, err
	}
	hashesB, err := sortedArtifactHashes(replay.Artifacts)
	if err != nil {
		return BenchmarkResult{}, err
	}
	deterministic := true
	for artifactKey, hash := range hashesA {
		if hashesB[artifactKey] != hash {
			deterministic = false
			break
		}
	}

	runtime.ReadMemStats(&after)
	totalRuntimeMS := time.Since(start).Seconds() * 1000

	profile := defaultProfile(opts.Profile)
	thresholds := lookupThresholds(entry.def, profile, opts.RepoRoot)
	eval := evaluate(thresholds, startupLatencyMS, totalRuntimeMS, len(result.Artifacts) > 0, deterministic)

	bench := BenchmarkResult{
		ExampleKey:          result.ExampleKey,
		Title:               result.Title,
		Profile:             profile,
		ConfigName:          filepath.Base(entry.def.ConfigPath),
		AgentCount:          result.AgentCount,
		InteractionCount:    result.InteractionCount,
		TaskCount:           result.TaskCount,
		StartupLatencyMS:    startupLatencyMS,
		TotalRuntimeMS:      totalRuntimeMS,
		ArtifactCount:       len(result.Artifacts),
		ArtifactSuccess:     len(result.Artifacts) > 0,
		DeterministicReplay: deterministic,
		MemoryAllocBytes:    after.TotalAlloc - before.TotalAlloc,
		LocalOnly:           result.LocalOnly,
		Environment:         environmentMetadata(),
		Thresholds:          thresholds,
		Evaluation:          eval,
		Artifacts:           filepathToSlashMapFromRoot(opts.RepoRoot, result.Artifacts),
		ArtifactHashes:      hashesA,
		CompletedAt:         nowRFC3339(),
	}

	path := benchmarkResultPath(opts, key, profile)
	if err := writeResultJSON(path, bench); err != nil {
		return BenchmarkResult{}, err
	}
	if err := copyLatest(path, benchmarkLatestPath(opts, key, profile)); err != nil {
		return BenchmarkResult{}, err
	}
	return bench, nil
}

func BenchmarkAll(opts RunOptions) (BenchmarkSuite, error) {
	defs := Definitions()
	results := make([]BenchmarkResult, 0, len(defs))
	for _, def := range defs {
		result, err := BenchmarkOne(def.Key, opts)
		if err != nil {
			return BenchmarkSuite{}, err
		}
		results = append(results, result)
	}
	return BenchmarkSuite{GeneratedAt: nowRFC3339(), Results: results}, nil
}

func SmokeOne(key string, opts RunOptions) (SmokeResult, error) {
	entry, ok := registry[key]
	if !ok {
		return SmokeResult{}, fmt.Errorf("unknown example: %s", key)
	}
	result, err := entry.run(opts)
	if err != nil {
		return SmokeResult{
			ExampleKey:      key,
			Title:           entry.def.Title,
			Profile:         defaultProfile(opts.Profile),
			Success:         false,
			ArtifactSuccess: false,
			OutputDir:       sanitizePath(opts.RepoRoot, resultOutputRoot(opts, key)),
			FailureReason:   err.Error(),
			CompletedAt:     nowRFC3339(),
		}, nil
	}
	smoke := SmokeResult{
		ExampleKey:       result.ExampleKey,
		Title:            result.Title,
		Profile:          defaultProfile(opts.Profile),
		Success:          true,
		ArtifactSuccess:  len(result.Artifacts) > 0,
		AgentCount:       result.AgentCount,
		InteractionCount: result.InteractionCount,
		TaskCount:        result.TaskCount,
		Artifacts:        filepathToSlashMapFromRoot(opts.RepoRoot, result.Artifacts),
		OutputDir:        sanitizePath(opts.RepoRoot, result.OutputDir),
		CompletedAt:      nowRFC3339(),
	}
	return smoke, nil
}

func sanitizePath(root, value string) string {
	if root != "" {
		if rel, err := filepath.Rel(root, value); err == nil {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(strings.TrimPrefix(value, root))
}

func SmokeAll(opts RunOptions) (SmokeSuite, error) {
	defs := Definitions()
	results := make([]SmokeResult, 0, len(defs))
	for _, def := range defs {
		result, err := SmokeOne(def.Key, opts)
		if err != nil {
			return SmokeSuite{}, err
		}
		results = append(results, result)
	}
	suite := SmokeSuite{GeneratedAt: nowRFC3339(), Results: results}
	path := smokeResultPath(opts)
	if err := writeResultJSON(path, suite); err != nil {
		return SmokeSuite{}, err
	}
	if err := copyLatest(path, smokeLatestPath(opts)); err != nil {
		return SmokeSuite{}, err
	}
	return suite, nil
}

func WriteBenchmarkSuite(path string, suite BenchmarkSuite) error {
	return writeResultJSON(path, suite)
}

func CompareFiles(basePath, currentPath string) (CompareReport, error) {
	var base BenchmarkResult
	if err := readJSONFile(basePath, &base); err != nil {
		return CompareReport{}, err
	}
	var current BenchmarkResult
	if err := readJSONFile(currentPath, &current); err != nil {
		return CompareReport{}, err
	}
	return CompareReport{
		Base:    basePath,
		Current: currentPath,
		Results: []CompareResult{
			{
				ExampleKey:         current.ExampleKey,
				Profile:            current.Profile,
				RuntimeDeltaMS:     current.TotalRuntimeMS - base.TotalRuntimeMS,
				StartupDeltaMS:     current.StartupLatencyMS - base.StartupLatencyMS,
				MemoryDeltaBytes:   int64(current.MemoryAllocBytes) - int64(base.MemoryAllocBytes),
				ArtifactCountDelta: current.ArtifactCount - base.ArtifactCount,
				StatusChanged:      current.Evaluation.Status != base.Evaluation.Status,
			},
		},
	}, nil
}
