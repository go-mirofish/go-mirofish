package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func newTestGateway(t *testing.T, handler func(*http.Request) (*http.Response, error)) *gateway {
	t.Helper()

	target, err := url.Parse("http://backend.test")
	if err != nil {
		t.Fatalf("parse dummy backend url: %v", err)
	}

	gw := newGateway(config{
		bindHost:        "127.0.0.1",
		port:            "3000",
		backendURL:      target,
		frontendDistDir: "frontend/dist",
	})
	gw.backendProxy.Transport = roundTripFunc(handler)
	return gw
}

func okBackendResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"success":true}`)),
	}
}

func TestSimulationRunAliasForwardsToStart(t *testing.T) {
	gw := newTestGateway(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/simulation/start" {
			t.Fatalf("expected /api/simulation/start, got %s", r.URL.Path)
		}

		return okBackendResponse(), nil
	})

	req := httptest.NewRequest(http.MethodPost, "/api/simulation/run", strings.NewReader(`{"simulation_id":"sim-1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	gw.handleSimulationRunAlias(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestSimulationStatusAliasForwardsToRunStatus(t *testing.T) {
	gw := newTestGateway(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/simulation/sim-9/run-status" {
			t.Fatalf("expected /api/simulation/sim-9/run-status, got %s", r.URL.Path)
		}

		return okBackendResponse(), nil
	})

	req := httptest.NewRequest(http.MethodGet, "/api/simulation/sim-9/status?verbose=1", nil)
	rec := httptest.NewRecorder()

	gw.handleSimulationStatusAlias(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestReportStatusAliasUsesProgressForReportID(t *testing.T) {
	gw := newTestGateway(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/report/report-42/progress" {
			t.Fatalf("expected /api/report/report-42/progress, got %s", r.URL.Path)
		}

		return okBackendResponse(), nil
	})

	req := httptest.NewRequest(http.MethodGet, "/api/report/generate/status?report_id=report-42", nil)
	rec := httptest.NewRecorder()

	gw.handleReportStatusAlias(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestReportStatusAliasBridgesQueryToPOSTBody(t *testing.T) {
	gw := newTestGateway(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/report/generate/status" {
			t.Fatalf("expected /api/report/generate/status, got %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}

		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		if payload["task_id"] != "task-7" {
			t.Fatalf("expected task_id task-7, got %#v", payload["task_id"])
		}
		if payload["simulation_id"] != "sim-7" {
			t.Fatalf("expected simulation_id sim-7, got %#v", payload["simulation_id"])
		}

		return okBackendResponse(), nil
	})

	req := httptest.NewRequest(http.MethodGet, "/api/report/generate/status?task_id=task-7&simulation_id=sim-7", nil)
	rec := httptest.NewRecorder()

	gw.handleReportStatusAlias(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
