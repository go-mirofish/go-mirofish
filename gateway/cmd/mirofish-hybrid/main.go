package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-mirofish/go-mirofish/gateway/internal/examples"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	wd, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	repoRoot := examples.ResolveRepoRoot(wd)

	switch os.Args[1] {
	case "help", "-h", "--help":
		usage()
		return
	case "merge-bundled":
		os.Exit(cmdMergeBundled(repoRoot, os.Args[2:]))
	case "live-benchmark":
		os.Exit(cmdLiveBenchmark(repoRoot, os.Args[2:]))
	case "stress-probe":
		os.Exit(cmdStressProbe(os.Args[2:]))
	case "api-smoke":
		os.Exit(cmdAPISmoke(repoRoot, os.Args[2:]))
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	exe := filepath.Base(os.Args[0])
	fmt.Printf(`%s — benchmark & bundled-json helpers (replaces scripts/hybrid).

Usage: %s <command> [options]

Commands:
  merge-bundled     Merge stack fields from a live benchmark JSON into docs/bundled-benchmarks/*__*__latest.json
  live-benchmark    Build gateway + frontend, run local gateway, then benchmark + markdown report
  stress-probe      Concurrent /health requests (latency min/p50/p95/max)
  api-smoke         Full API flow: ontology → graph → simulation → report (HTTP against gateway or Vite proxy)

`, exe, exe)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "ERROR:", err)
	os.Exit(1)
}
