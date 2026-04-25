package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var stackKeys = []string{
	"captured_at", "release", "host", "backend_health", "gateway_health",
	"processes", "benchmark", "stress",
}

func cmdMergeBundled(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("merge-bundled", flag.ExitOnError)
	liveArg := fs.String("live", "", "path to live benchmark JSON (default: auto-detect under benchmark/results/benchmarks/)")
	_ = fs.Parse(args)

	livePath := strings.TrimSpace(*liveArg)
	if livePath == "" {
		for _, name := range []string{
			"v0.1.0-live-benchmark.json",
			"live-benchmark.json",
		} {
			p := filepath.Join(repoRoot, "benchmark", "results", "benchmarks", name)
			if _, err := os.Stat(p); err == nil {
				livePath = p
				break
			}
		}
	}
	if livePath == "" {
		fmt.Fprintln(os.Stderr, "merge-bundled: no live benchmark JSON found; pass -live=path or run live-benchmark first")
		return 1
	}

	raw, err := os.ReadFile(livePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "merge-bundled:", err)
		return 1
	}
	var live map[string]json.RawMessage
	if err := json.Unmarshal(raw, &live); err != nil {
		fmt.Fprintln(os.Stderr, "merge-bundled:", err)
		return 1
	}
	stack := make(map[string]json.RawMessage)
	for _, k := range stackKeys {
		if v, ok := live[k]; ok {
			stack[k] = v
		}
	}

	bundled := filepath.Join(repoRoot, "docs", "bundled-benchmarks")
	entries, err := os.ReadDir(bundled)
	if err != nil {
		fmt.Fprintln(os.Stderr, "merge-bundled:", err)
		return 1
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, "__latest.json") {
			continue
		}
		if !strings.Contains(name, "__") {
			continue
		}
		path := filepath.Join(bundled, name)
		if err := mergeOneBundled(path, stack); err != nil {
			fmt.Fprintln(os.Stderr, "merge-bundled:", err)
			return 1
		}
		fmt.Println("updated", rel(repoRoot, path))
	}

	for _, p := range []string{
		filepath.Join(repoRoot, "benchmark", "results", "benchmarks", "examples-benchmark-suite.json"),
		filepath.Join(repoRoot, "benchmark", "results", "examples-benchmark-suite.json"),
	} {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if err := mergeExamplesSuite(p, stack); err != nil {
			fmt.Fprintln(os.Stderr, "merge-bundled:", err)
			return 1
		}
		fmt.Println("updated", rel(repoRoot, p))
	}
	return 0
}

func rel(root, p string) string {
	s, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return filepath.ToSlash(s)
}

func mergeOneBundled(path string, stack map[string]json.RawMessage) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	if data["example_key"] == nil && data["results"] != nil {
		return nil
	}
	for k, v := range stack {
		data[k] = v
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}

func mergeExamplesSuite(path string, stack map[string]json.RawMessage) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var root map[string]interface{}
	if err := json.Unmarshal(raw, &root); err != nil {
		return err
	}
	stackMap := stackToIface(stack)
	// {**stack, **sdata}: sdata (root) values win on duplicate keys, then re-apply stack... Python does:
	// out = {**stack, **sdata}  => sdata on top of stack
	merged := make(map[string]interface{}, len(stackMap)+len(root))
	for k, v := range stackMap {
		merged[k] = v
	}
	for k, v := range root {
		merged[k] = v
	}
	// each row: {**row, **stack} => stack on top
	rows, ok := merged["results"].([]interface{})
	if ok {
		for i, row := range rows {
			m, ok := row.(map[string]interface{})
			if !ok {
				continue
			}
			nr := make(map[string]interface{}, len(m)+len(stackMap))
			for k, v := range m {
				nr[k] = v
			}
			for k, v := range stackMap {
				nr[k] = v
			}
			rows[i] = nr
		}
		merged["results"] = rows
	}
	enc, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(enc, '\n'), 0o644)
}

func stackToIface(stack map[string]json.RawMessage) map[string]interface{} {
	out := make(map[string]interface{}, len(stack))
	for k, v := range stack {
		var x interface{}
		if err := json.Unmarshal(v, &x); err != nil {
			continue
		}
		out[k] = x
	}
	return out
}
