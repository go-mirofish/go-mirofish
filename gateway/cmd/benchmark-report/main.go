// Package main generates a markdown benchmark report from a benchmark JSON artifact.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	inputFile := flag.String("input", "", "benchmark JSON input file")
	outputFile := flag.String("output", "", "markdown output file")
	title := flag.String("title", "go-mirofish benchmark report", "report title")
	flag.Parse()

	if *inputFile == "" {
		fmt.Fprintln(os.Stderr, "-input is required")
		os.Exit(1)
	}

	raw, err := os.ReadFile(*inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read input: %v\n", err)
		os.Exit(1)
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		fmt.Fprintf(os.Stderr, "parse JSON: %v\n", err)
		os.Exit(1)
	}

	var sb strings.Builder
	sb.WriteString("# " + *title + "\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().UTC().Format(time.RFC3339)))

	if v, ok := data["captured_at"]; ok {
		sb.WriteString(fmt.Sprintf("**Captured:** %v\n\n", v))
	}
	if v, ok := data["release"]; ok {
		sb.WriteString(fmt.Sprintf("**Release:** %v\n\n", v))
	}
	if v, ok := data["base_url"]; ok {
		sb.WriteString(fmt.Sprintf("**Base URL:** %v\n\n", v))
	}

	if criteria, ok := data["release_criteria"].(map[string]any); ok {
		pass := criteria["pass"]
		sb.WriteString(fmt.Sprintf("## Release Criteria\n\n**Pass:** %v\n\n", pass))
		sb.WriteString(fmt.Sprintf("- Load p95 < %.0fms\n", asFloat(criteria["load_p95_under_ms"])))
		sb.WriteString(fmt.Sprintf("- Load error rate < %.2f%%\n", asFloat(criteria["load_error_rate_under"])*100))
		sb.WriteString(fmt.Sprintf("- Stress p95 < %.0fms\n\n", asFloat(criteria["stress_p95_under_ms"])))
	}

	if runs, ok := data["runs"].([]any); ok {
		sb.WriteString("## Benchmark Runs\n\n")
		sb.WriteString("| Profile | Concurrency | Requests | Throughput (rps) | Error Rate | p50 (ms) | p95 (ms) | p99 (ms) | Alloc MB |\n")
		sb.WriteString("|---------|-------------|----------|-----------------|------------|----------|----------|----------|----------|\n")
		for _, runAny := range runs {
			run, ok := runAny.(map[string]any)
			if !ok {
				continue
			}
			latency, _ := run["latency"].(map[string]any)
			mem, _ := run["memory"].(map[string]any)
			sb.WriteString(fmt.Sprintf("| %s | %d | %d | %.2f | %.4f | %.2f | %.2f | %.2f | %.2f |\n",
				run["profile"],
				int(asFloat(run["concurrency"])),
				int(asFloat(run["requests"])),
				asFloat(run["throughput_rps"]),
				asFloat(run["error_rate"]),
				asFloat(latency["p50_ms"]),
				asFloat(latency["p95_ms"]),
				asFloat(latency["p99_ms"]),
				asFloat(mem["alloc_mb"]),
			))
		}
		sb.WriteString("\n")
	}

	if bc, ok := data["baseline_compare"].(map[string]any); ok {
		if note, ok := bc["note"]; ok {
			sb.WriteString(fmt.Sprintf("## Baseline\n\n%v\n\n", note))
		}
	}

	if *outputFile != "" {
		if err := os.MkdirAll(dirOf(*outputFile), 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(*outputFile, []byte(sb.String()), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("benchmark report written to %s\n", *outputFile)
	} else {
		fmt.Print(sb.String())
	}
}

func asFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	}
	return 0
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
