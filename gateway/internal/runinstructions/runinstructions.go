// Package runinstructions builds public run_instructions payloads owned by the Go control plane
// (gateway API), without embedding legacy Python CLI strings in the default response.
package runinstructions

import (
	"fmt"
	"os"
	"strings"
)

// Build returns a Go-gateway–native run instruction map. All simulation execution in production
// flows through the gateway HTTP API and the Go-native worker; paths are provided for inspection only.
func Build(simulationID, simulationDir, configFile, scriptsDir string) map[string]any {
	gw := strings.TrimSpace(os.Getenv("GATEWAY_PUBLIC_BASE_URL"))
	if gw == "" {
		gw = "http://127.0.0.1:3000"
	}
	gw = strings.TrimRight(gw, "/")
	instr := fmt.Sprintf(
		"Control plane: Mirofish gateway (Go), worker runtime: native.\n"+
			"1) Start the API: make up (Docker). For UI, run: npm run dev (Vite on :5173).\n"+
			"2) Start this simulation: POST %s/api/simulation/start with JSON body "+
			`{"simulation_id":"%s","platform":"parallel"}`+" (use twitter, reddit, or parallel).\n"+
			"3) Poll status: GET %s/api/simulation/%s/status\n"+
			"4) Optional: set GATEWAY_PUBLIC_BASE_URL if your gateway is not on %s.\n",
		gw, simulationID, gw, simulationID, gw,
	)
	return map[string]any{
		"simulation_id":  simulationID,
		"simulation_dir":   simulationDir,
		"config_file":      configFile,
		"scripts_dir":      scriptsDir,
		"control_plane":    "go_gateway",
		"worker_runtime":   "native",
		"gateway_base_url": gw,
		"api": map[string]any{
			"start": map[string]any{
				"method": "POST",
				"path":   "/api/simulation/start",
				"body": map[string]any{
					"simulation_id": simulationID,
					"platform":      "parallel",
				},
			},
			"status": map[string]any{
				"method": "GET",
				"path":   fmt.Sprintf("/api/simulation/%s/status", simulationID),
			},
		},
		"instructions": instr,
	}
}
