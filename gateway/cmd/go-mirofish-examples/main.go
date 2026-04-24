package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-mirofish/go-mirofish/gateway/internal/examples"
)

func main() {
	var (
		listFlag   = flag.Bool("list", false, "list available examples")
		exampleKey = flag.String("example", "", "run one example by key")
		allFlag    = flag.Bool("all", false, "run all examples")
		smokeOnly  = flag.Bool("smoke-only", false, "run smoke validation only")
		benchOnly  = flag.Bool("bench-only", false, "run benchmark only")
		outputDir  = flag.String("output", "", "override output directory")
		profile    = flag.String("profile", "medium", "example profile: small, medium, stress")
		compare    = flag.String("compare", "", "compare two benchmark result files as base,current")
	)
	flag.Parse()

	repoRoot, err := os.Getwd()
	if err != nil {
		fail(err)
	}
	repoRoot = examples.ResolveRepoRoot(repoRoot)

	if *listFlag {
		emit(examples.Definitions())
		return
	}
	if *compare != "" {
		base, current, ok := splitPair(*compare)
		if !ok {
			fail(fmt.Errorf("--compare expects base,current"))
		}
		report, err := examples.CompareFiles(base, current)
		if err != nil {
			fail(err)
		}
		emit(report)
		return
	}

	if !*allFlag && *exampleKey == "" {
		fail(fmt.Errorf("provide --all or --example"))
	}

	opts := examples.RunOptions{
		RepoRoot:   repoRoot,
		OutputRoot: *outputDir,
		Profile:    *profile,
		SmokeOnly:  *smokeOnly,
	}

	if *allFlag {
		if *benchOnly {
			suite, err := examples.BenchmarkAll(opts)
			if err != nil {
				fail(err)
			}
			outPath := filepath.Join(repoRoot, "benchmark", "results", "examples-benchmark-suite.json")
			if err := examples.WriteBenchmarkSuite(outPath, suite); err != nil {
				fail(err)
			}
			emit(map[string]any{"result_path": outPath, "results": suite.Results})
			return
		}
		suite, err := examples.SmokeAll(opts)
		if err != nil {
			fail(err)
		}
		if *smokeOnly {
			emit(suite)
			return
		}
		bench, err := examples.BenchmarkAll(opts)
		if err != nil {
			fail(err)
		}
		outPath := filepath.Join(repoRoot, "benchmark", "results", "examples-benchmark-suite.json")
		if err := examples.WriteBenchmarkSuite(outPath, bench); err != nil {
			fail(err)
		}
		emit(map[string]any{
			"smoke":       suite.Results,
			"result_path": outPath,
			"benchmarks":  bench.Results,
		})
		return
	}

	if *benchOnly {
		result, err := examples.BenchmarkOne(*exampleKey, opts)
		if err != nil {
			fail(err)
		}
		outPath := filepath.Join(repoRoot, "benchmark", "results", *exampleKey, *profile, "latest.json")
		emit(map[string]any{"result_path": outPath, "result": result})
		return
	}

	smoke, err := examples.SmokeOne(*exampleKey, opts)
	if err != nil {
		fail(err)
	}
	if *smokeOnly {
		emit(smoke)
		return
	}
	bench, err := examples.BenchmarkOne(*exampleKey, opts)
	if err != nil {
		fail(err)
	}
	emit(map[string]any{
		"smoke":     smoke,
		"benchmark": bench,
	})
}

func splitPair(value string) (string, string, bool) {
	for i := 0; i < len(value); i++ {
		if value[i] == ',' {
			return value[:i], value[i+1:], true
		}
	}
	return "", "", false
}

func emit(payload any) {
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fail(err)
	}
	fmt.Println(string(raw))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
