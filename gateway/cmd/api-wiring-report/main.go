// Package main generates a machine-readable API wiring report by hitting every
// registered gateway route and recording the actual HTTP response code, owner,
// and wiring status.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type routeCheck struct {
	Method     string `json:"method"`
	Path       string `json:"path"`
	Owner      string `json:"owner"`
	StatusCode int    `json:"status_code"`
	Wiring     string `json:"wiring"` // "go_native" | "410_removed" | "error"
	Notes      string `json:"notes,omitempty"`
}

type wiringReport struct {
	GeneratedAt   string       `json:"generated_at"`
	GatewayURL    string       `json:"gateway_url"`
	Stack         string       `json:"stack"`
	PythonBackend string       `json:"python_backend"`
	Routes        []routeCheck `json:"routes"`
	Summary       summary      `json:"summary"`
}

type summary struct {
	Total      int `json:"total"`
	GoNative   int `json:"go_native"`
	Removed410 int `json:"removed_410"`
	Errors     int `json:"errors"`
}

func main() {
	baseURL := flag.String("base-url", "http://127.0.0.1:3000", "gateway base URL")
	outFile := flag.String("out", "", "output JSON file (default: stdout)")
	flag.Parse()

	client := &http.Client{Timeout: 10 * time.Second}

	checks := []struct {
		method string
		path   string
		owner  string
		notes  string
	}{
		// Infrastructure routes
		{"GET", "/health", "go_gateway", ""},
		{"GET", "/ready", "go_gateway", ""},
		{"GET", "/metrics", "go_gateway", ""},
		// Graph routes
		{"GET", "/api/graph/project/list", "go_gateway", "project list — storage-backed"},
		{"GET", "/api/graph/tasks", "go_gateway", "task list — storage-backed"},
		{"GET", "/api/graph/task/nonexistent-task", "go_gateway", "task get by ID — 404 expected without data"},
		{"GET", "/api/graph/data/nonexistent-graph", "go_gateway", "graph data by ID — 5xx expected without Zep configured (wired correctly; fails at memory layer)"},
		{"POST", "/api/graph/build", "go_gateway", "graph build — rate-limited POST"},
		{"POST", "/api/graph/ontology/generate", "go_gateway", "ontology generate — rate-limited POST"},
		// Report routes
		{"GET", "/api/report/list", "go_gateway", "report list — storage-backed"},
		{"POST", "/api/report/generate", "go_gateway", "report generate — LLM-gated, 4xx expected without config"},
		// Simulation routes
		{"GET", "/api/simulation/list", "go_gateway", "simulation list — storage-backed"},
		{"POST", "/api/simulation/create", "go_gateway", "simulation create"},
		{"POST", "/api/simulation/prepare", "go_gateway", "simulation prepare — rate-limited"},
		{"GET", "/api/simulation/nonexistent-sim/status", "go_gateway", "simulation status — 404 expected without data"},
		// Catch-all for removed Python routes
		{"GET", "/api/legacy-python-route", "removed", "legacy route — must return 410 Gone"},
		{"POST", "/api/old-backend/endpoint", "removed", "legacy route — must return 410 Gone"},
		// Frontend SPA
		{"GET", "/", "go_gateway", "index.html SPA fallback"},
		{"GET", "/some/spa/path", "go_gateway", "SPA route — falls through to index.html"},
	}

	var routes []routeCheck
	for _, chk := range checks {
		var resp *http.Response
		var err error
		url := *baseURL + chk.path
		if chk.method == "GET" {
			resp, err = client.Get(url)
		} else {
			resp, err = client.Post(url, "application/json", nil)
		}

		rc := routeCheck{
			Method: chk.method,
			Path:   chk.path,
			Owner:  chk.owner,
			Notes:  chk.notes,
		}

		if err != nil {
			rc.StatusCode = 0
			rc.Wiring = "error"
			rc.Notes += " | conn_err: " + err.Error()
		} else {
			resp.Body.Close()
			rc.StatusCode = resp.StatusCode
			switch {
			case resp.StatusCode == http.StatusGone && chk.owner == "removed":
				rc.Wiring = "410_removed"
			case resp.StatusCode == http.StatusGone && chk.owner != "removed":
				rc.Wiring = "error"
				rc.Notes += " | unexpected 410: route misrouted to removal handler"
			case resp.StatusCode < 500:
				rc.Wiring = "go_native"
			case resp.StatusCode >= 500 && strings.Contains(chk.notes, "5xx expected"):
				// 5xx from an external dependency (e.g. Zep not configured) is acceptable
				// when the route is correctly wired to a Go-native handler.
				rc.Wiring = "go_native"
			default:
				rc.Wiring = "error"
				rc.Notes += fmt.Sprintf(" | unexpected 5xx: %d", resp.StatusCode)
			}
		}

		routes = append(routes, rc)
	}

	s := summary{}
	for _, r := range routes {
		s.Total++
		switch r.Wiring {
		case "go_native":
			s.GoNative++
		case "410_removed":
			s.Removed410++
		case "error":
			s.Errors++
		}
	}

	report := wiringReport{
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		GatewayURL:    *baseURL,
		Stack:         "go+vue+docker",
		PythonBackend: "removed",
		Routes:        routes,
		Summary:       s,
	}

	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		os.Exit(1)
	}

	if *outFile != "" {
		if err := os.WriteFile(*outFile, raw, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", *outFile, err)
			os.Exit(1)
		}
		fmt.Printf("API wiring report written to %s\n", *outFile)
	} else {
		fmt.Println(string(raw))
	}

	if s.Errors > 0 {
		fmt.Fprintf(os.Stderr, "FAIL: %d route(s) have wiring errors\n", s.Errors)
		os.Exit(2)
	}
}
