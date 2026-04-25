package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultRequirement = "Simulate how this fictional scenario evolves across public discussion, " +
	"identify the key actors and likely sentiment shifts, and generate a concise analysis report."

func cmdAPISmoke(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("api-smoke", flag.ExitOnError)
	baseURL := fs.String("base-url", "http://127.0.0.1:3000", "Gateway base (or Vite with /api proxy)")
	ontologyBase := fs.String("ontology-base-url", "", "Override for ontology only (default: --base-url)")
	pdf := fs.String("pdf", "", "Optional path to a document; default benchmark/seed.txt")
	reqStr := fs.String("requirement", "", "Simulation requirement text")
	maxRounds := fs.Int("max-rounds", 3, "Simulation max rounds")
	outPath := fs.String("output", "", "Write full JSON record to this file")
	timeoutSec := fs.Int("timeout-seconds", 900, "Total poll budget")
	reqTimeout := fs.Int("request-timeout", 120, "Per-request timeout (seconds)")
	_ = fs.Parse(args)

	seedPath := filepath.Join(repoRoot, "benchmark", "seed.txt")
	if strings.TrimSpace(*pdf) != "" {
		p := *pdf
		if !filepath.IsAbs(p) {
			p = filepath.Join(repoRoot, p)
		}
		seedPath = filepath.Clean(p)
	}
	if _, err := os.Stat(seedPath); err != nil {
		fmt.Fprintln(os.Stderr, "api-smoke:", err)
		return 1
	}

	base := strings.TrimRight(*baseURL, "/")
	ontoBase := base
	if strings.TrimSpace(*ontologyBase) != "" {
		ontoBase = strings.TrimRight(*ontologyBase, "/")
	}
	requirement := strings.TrimSpace(*reqStr)
	if requirement == "" {
		requirement = defaultRequirement
	}

	c := &http.Client{Timeout: time.Duration(*reqTimeout) * time.Second}
	t0 := time.Now()
	timings := map[string]float64{}

	body, contentType, err := buildMultipart(seedPath, "Hybrid benchmark smoke", requirement)
	if err != nil {
		fmt.Fprintln(os.Stderr, "api-smoke:", err)
		return 1
	}
	ontology, err := postJSON(c, ontoBase+"/api/graph/ontology/generate", body, contentType)
	if err != nil {
		fmt.Fprintln(os.Stderr, "api-smoke: ontology:", err)
		return 1
	}
	timings["ontology_request_s"] = time.Since(t0).Seconds()

	data := jsonMap(ontology, "data")
	projectID, _ := data["project_id"].(string)
	if projectID == "" {
		fmt.Fprintln(os.Stderr, "api-smoke: missing project_id")
		return 1
	}
	if tid, _ := data["task_id"].(string); tid != "" {
		if err := pollTask(c, base, ontoBase, tid, "ontology task", *timeoutSec, *reqTimeout); err != nil {
			fmt.Fprintln(os.Stderr, "api-smoke:", err)
			return 1
		}
	} else if data["ontology"] == nil {
		fmt.Fprintln(os.Stderr, "api-smoke: ontology success without task_id or ontology payload")
		return 1
	}

	t1 := time.Now()
	gb, err := postJSON(c, base+"/api/graph/build", jsonBody(map[string]any{"project_id": projectID}), "application/json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "api-smoke: graph build:", err)
		return 1
	}
	gd := jsonMap(gb, "data")
	gtask, _ := gd["task_id"].(string)
	gtaskPayload, err := pollGraphTask(c, base, gtask, *timeoutSec, *reqTimeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "api-smoke:", err)
		return 1
	}
	timings["graph_build_total_s"] = time.Since(t1).Seconds()
	gd2 := jsonMap(gtaskPayload, "data")
	graphID, _ := gd2["graph_id"].(string)
	if graphID == "" {
		pi, _ := getJSON(c, base+"/api/graph/project/"+url.PathEscape(projectID))
		if m := jsonMap(pi, "data"); m != nil {
			graphID, _ = m["graph_id"].(string)
		}
	}
	if graphID == "" {
		fmt.Fprintln(os.Stderr, "api-smoke: graph_id missing after build")
		return 1
	}

	t2 := time.Now()
	simCreate, err := postJSON(c, base+"/api/simulation/create", jsonBody(map[string]any{
		"project_id": projectID,
		"graph_id":   graphID,
	}), "application/json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "api-smoke: simulation create:", err)
		return 1
	}
	simID, _ := jsonMap(simCreate, "data")["simulation_id"].(string)
	prepare, err := postJSON(c, base+"/api/simulation/prepare", jsonBody(map[string]any{
		"simulation_id":        simID,
		"graph_id":             graphID,
		"use_llm_for_profiles": false,
	}), "application/json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "api-smoke: prepare:", err)
		return 1
	}
	prepTask, _ := jsonMap(prepare, "data")["task_id"].(string)
	if err := pollPrepare(c, base, prepTask, simID, *timeoutSec, *reqTimeout); err != nil {
		fmt.Fprintln(os.Stderr, "api-smoke:", err)
		return 1
	}
	timings["simulation_create_prepare_s"] = time.Since(t2).Seconds()

	t3 := time.Now()
	_, err = postJSON(c, base+"/api/simulation/run", jsonBody(map[string]any{
		"simulation_id": simID,
		"max_rounds":    *maxRounds,
	}), "application/json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "api-smoke: run:", err)
		return 1
	}
	simSt, err := pollUntil(
		c,
		func() (map[string]any, string, error) {
			p, e := getJSON(c, base+"/api/simulation/"+url.PathEscape(simID)+"/status")
			if e != nil {
				return nil, "", e
			}
			d := jsonMap(p, "data")
			st, _ := d["runner_status"].(string)
			if st == "" {
				st, _ = d["status"].(string)
			}
			return p, st, nil
		},
		map[string]struct{}{"completed": {}, "stopped": {}},
		time.Duration(*timeoutSec)*time.Second,
		5*time.Second,
		"simulation run",
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "api-smoke:", err)
		return 1
	}
	timings["simulation_run_s"] = time.Since(t3).Seconds()
	simData := jsonMap(simSt, "data")

	repGen, err := postJSON(c, base+"/api/report/generate", jsonBody(map[string]any{
		"simulation_id": simID,
	}), "application/json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "api-smoke: report generate:", err)
		return 1
	}
	rd := jsonMap(repGen, "data")
	var reportID string
	if x, _ := rd["report_id"].(string); x != "" {
		reportID = x
	}
	reportTask, _ := rd["task_id"].(string)
	already, _ := rd["already_generated"].(bool)

	t4 := time.Now()
	var reportStatus map[string]any
	if already {
		reportStatus, err = getJSON(c, base+"/api/report/generate/status?report_id="+url.QueryEscape(reportID))
	} else if reportTask != "" {
		u := fmt.Sprintf("%s/api/report/generate/status?task_id=%s&simulation_id=%s",
			base, url.QueryEscape(reportTask), url.QueryEscape(simID))
		reportStatus, err = pollReport(c, u, *timeoutSec, *reqTimeout)
	} else if reportID != "" {
		u := base + "/api/report/generate/status?report_id=" + url.QueryEscape(reportID)
		reportStatus, err = pollReport(c, u, *timeoutSec, *reqTimeout)
	} else {
		fmt.Fprintln(os.Stderr, "api-smoke: report generation: no report_id or task_id")
		return 1
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "api-smoke:", err)
		return 1
	}
	timings["report_generation_s"] = time.Since(t4).Seconds()
	rsd := jsonMap(reportStatus, "data")
	if reportID == "" {
		if x, _ := rsd["report_id"].(string); x != "" {
			reportID = x
		}
	}
	if m := rsd["result"]; m != nil {
		if mm, ok := m.(map[string]any); ok {
			if x, _ := mm["report_id"].(string); x != "" {
				reportID = x
			}
		}
	}
	if reportID == "" {
		fmt.Fprintln(os.Stderr, "api-smoke: no report_id after generation")
		return 1
	}
	report, err := getJSON(c, base+"/api/report/"+url.PathEscape(reportID))
	if err != nil {
		fmt.Fprintln(os.Stderr, "api-smoke: fetch report:", err)
		return 1
	}
	timings["wall_clock_total_s"] = time.Since(t0).Seconds()

	repD := jsonMap(report, "data")
	nonEmpty := firstStr(repD["markdown_content"]) != "" || firstStr(repD["content"]) != ""

	simStatus, _ := simData["runner_status"].(string)
	if simStatus == "" {
		simStatus, _ = simData["status"].(string)
	}
	repS, _ := rsd["status"].(string)

	kind := "live_hybrid_smoke"
	if *pdf != "" {
		kind = "live_hybrid_pdf_benchmark"
	}
	out := map[string]any{
		"kind":            kind,
		"base_url":        base,
		"ontology_base_url": ontoBase,
		"input": map[string]any{
			"path":        seedPath,
			"requirement": requirement,
			"max_rounds":  *maxRounds,
		},
		"ids": map[string]any{
			"project_id":     projectID,
			"graph_id":       graphID,
			"simulation_id":  simID,
			"report_id":      reportID,
		},
		"timings_s": timings,
		"outcomes": map[string]any{
			"simulation_status":  simStatus,
			"report_status":     repS,
			"report_non_empty":   nonEmpty,
		},
	}
	if *outPath != "" {
		b, _ := json.MarshalIndent(out, "", "  ")
		_ = os.MkdirAll(filepath.Dir(*outPath), 0o755)
		_ = os.WriteFile(*outPath, append(b, '\n'), 0o644)
		fmt.Fprintf(os.Stderr, "Wrote: %s\n", *outPath)
	}
	summary := map[string]any{
		"ids":        out["ids"],
		"timings_s":  timings,
		"outcomes":   out["outcomes"],
	}
	enc, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Println(string(enc))
	return 0
}

func firstStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func jsonMap(v map[string]any, key string) map[string]any {
	if v == nil {
		return nil
	}
	if raw, ok := v[key].(map[string]any); ok {
		return raw
	}
	return nil
}

func buildMultipart(seedPath, projectName, requirement string) ([]byte, string, error) {
	b := &bytes.Buffer{}
	w := multipart.NewWriter(b)
	_ = w.WriteField("project_name", projectName)
	_ = w.WriteField("simulation_requirement", requirement)
	f, err := os.Open(seedPath)
	if err != nil {
		_ = w.Close()
		return nil, "", err
	}
	part, err := w.CreateFormFile("files", filepath.Base(seedPath))
	if err != nil {
		_ = f.Close()
		_ = w.Close()
		return nil, "", err
	}
	if _, err = io.Copy(part, f); err != nil {
		_ = f.Close()
		_ = w.Close()
		return nil, "", err
	}
	_ = f.Close()
	ct := w.FormDataContentType()
	if err = w.Close(); err != nil {
		return nil, "", err
	}
	return b.Bytes(), ct, nil
}

func jsonBody(m map[string]any) []byte {
	b, _ := json.Marshal(m)
	return b
}

func postJSON(c *http.Client, u string, body []byte, contentType string) (map[string]any, error) {
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func getJSON(c *http.Client, u string) (map[string]any, error) {
	resp, err := c.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func pollTask(c *http.Client, _, ontoBase, taskID, label string, timeoutSec, _ int) error {
	_, err := pollUntil(
		c,
		func() (map[string]any, string, error) {
			p, e := getJSON(c, ontoBase+"/api/graph/task/"+url.PathEscape(taskID))
			if e != nil {
				return nil, "", e
			}
			d := jsonMap(p, "data")
			st, _ := d["status"].(string)
			return p, st, nil
		},
		map[string]struct{}{"completed": {}},
		time.Duration(timeoutSec)*time.Second,
		2*time.Second,
		label,
	)
	return err
}

func pollGraphTask(c *http.Client, base, taskID string, timeoutSec, reqTo int) (map[string]any, error) {
	return pollUntil(
		c,
		func() (map[string]any, string, error) {
			p, e := getJSON(c, base+"/api/graph/task/"+url.PathEscape(taskID))
			if e != nil {
				return nil, "", e
			}
			d := jsonMap(p, "data")
			st, _ := d["status"].(string)
			return p, st, nil
		},
		map[string]struct{}{"completed": {}},
		time.Duration(timeoutSec)*time.Second,
		2*time.Second,
		"graph build task",
	)
}

func pollPrepare(c *http.Client, base, taskID, simID string, timeoutSec, reqTo int) error {
	_, err := pollUntil(
		c,
		func() (map[string]any, string, error) {
			body := jsonBody(map[string]any{"task_id": taskID, "simulation_id": simID})
			p, e := postJSON(c, base+"/api/simulation/prepare/status", body, "application/json")
			if e != nil {
				return nil, "", e
			}
			d := jsonMap(p, "data")
			st, _ := d["status"].(string)
			if st == "" {
				st, _ = d["runner_status"].(string)
			}
			return p, st, nil
		},
		map[string]struct{}{"ready": {}, "completed": {}},
		time.Duration(timeoutSec)*time.Second,
		3*time.Second,
		"simulation prepare",
	)
	return err
}

func pollReport(c *http.Client, u string, timeoutSec, reqTo int) (map[string]any, error) {
	c2 := *c
	if reqTo > 0 {
		c2.Timeout = time.Duration(reqTo) * time.Second
	}
	return pollUntil(
		&c2,
		func() (map[string]any, string, error) {
			p, e := getJSON(&c2, u)
			if e != nil {
				return nil, "", e
			}
			d := jsonMap(p, "data")
			st, _ := d["status"].(string)
			return p, st, nil
		},
		map[string]struct{}{"completed": {}},
		time.Duration(timeoutSec)*time.Second,
		5*time.Second,
		"report generation",
	)
}

func pollUntil(
	c *http.Client,
	fetch func() (map[string]any, string, error),
	accepted map[string]struct{},
	total time.Duration,
	interval time.Duration,
	label string,
) (map[string]any, error) {
	deadline := time.Now().Add(total)
	var last map[string]any
	for time.Now().Before(deadline) {
		p, st, err := fetch()
		last = p
		if err != nil {
			time.Sleep(interval)
			continue
		}
		if _, ok := accepted[st]; ok {
			return p, nil
		}
		if st == "failed" || st == "error" {
			return p, fmt.Errorf("%s failed: %v", label, p)
		}
		time.Sleep(interval)
	}
	return last, fmt.Errorf("%s: timeout, last status incomplete", label)
}
