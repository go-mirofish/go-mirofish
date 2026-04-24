package graphhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	intgraph "github.com/go-mirofish/go-mirofish/gateway/internal/graph"
	graphstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/graph"
)

type serviceStub struct {
	startBuild    func(context.Context, intgraph.BuildRequest) (intgraph.BuildResponse, error)
	getGraphData  func(context.Context, string) (map[string]any, error)
	deleteGraph   func(context.Context, string) error
	getTask       func(string) (graphstore.TaskState, error)
	listTasks     func(string) ([]graphstore.TaskState, error)
	getProject    func(string) (map[string]any, error)
	listProjects  func(int) ([]map[string]any, error)
	deleteProject func(string) error
	resetProject  func(string) (map[string]any, error)
}

func (s serviceStub) StartBuild(ctx context.Context, req intgraph.BuildRequest) (intgraph.BuildResponse, error) {
	return s.startBuild(ctx, req)
}

func (s serviceStub) GetGraphData(ctx context.Context, graphID string) (map[string]any, error) {
	return s.getGraphData(ctx, graphID)
}

func (s serviceStub) DeleteGraph(ctx context.Context, graphID string) error {
	return s.deleteGraph(ctx, graphID)
}

func (s serviceStub) GetTask(taskID string) (graphstore.TaskState, error) {
	return s.getTask(taskID)
}

func (s serviceStub) ListTasks(taskType string) ([]graphstore.TaskState, error) {
	return s.listTasks(taskType)
}

func (s serviceStub) GetProject(projectID string) (map[string]any, error) {
	return s.getProject(projectID)
}

func (s serviceStub) ListProjects(limit int) ([]map[string]any, error) {
	return s.listProjects(limit)
}

func (s serviceStub) DeleteProject(projectID string) error {
	return s.deleteProject(projectID)
}

func (s serviceStub) ResetProject(projectID string) (map[string]any, error) {
	return s.resetProject(projectID)
}

func TestHandleBuild(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		method     string
		body       string
		startBuild func(context.Context, intgraph.BuildRequest) (intgraph.BuildResponse, error)
		wantStatus int
		wantError  string
	}{
		{
			name:   "success",
			method: http.MethodPost,
			body:   `{"project_id":"proj-1","chunk_size":500}`,
			startBuild: func(ctx context.Context, req intgraph.BuildRequest) (intgraph.BuildResponse, error) {
				_ = ctx
				if req.ProjectID != "proj-1" {
					t.Fatalf("project_id = %q", req.ProjectID)
				}
				return intgraph.BuildResponse{ProjectID: "proj-1", TaskID: "task-1", Message: "started"}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "method not allowed",
			method:     http.MethodGet,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "bad json",
			method:     http.MethodPost,
			body:       `{`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid JSON body",
		},
		{
			name:   "service error",
			method: http.MethodPost,
			body:   `{"project_id":""}`,
			startBuild: func(ctx context.Context, req intgraph.BuildRequest) (intgraph.BuildResponse, error) {
				_ = ctx
				_ = req
				return intgraph.BuildResponse{}, errors.New("graph.StartBuild: project_id is required")
			},
			wantStatus: http.StatusBadRequest,
			wantError:  "graph.StartBuild: project_id is required",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := NewHandler(serviceStub{
				startBuild:    defaultStartBuild(tt.startBuild),
				getGraphData:  func(context.Context, string) (map[string]any, error) { return nil, nil },
				deleteGraph:   func(context.Context, string) error { return nil },
				getTask:       func(string) (graphstore.TaskState, error) { return graphstore.TaskState{}, nil },
				listTasks:     func(string) ([]graphstore.TaskState, error) { return nil, nil },
				getProject:    func(string) (map[string]any, error) { return nil, nil },
				listProjects:  func(int) ([]map[string]any, error) { return nil, nil },
				deleteProject: func(string) error { return nil },
				resetProject:  func(string) (map[string]any, error) { return nil, nil },
			})

			req := httptest.NewRequest(tt.method, "/api/graph/build", bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()
			handler.HandleBuild(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusMethodNotAllowed {
				if !strings.Contains(rec.Body.String(), "method not allowed") {
					t.Fatalf("body = %q, want method not allowed", rec.Body.String())
				}
				return
			}

			var payload map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if tt.wantError != "" && payload["error"] != tt.wantError {
				t.Fatalf("error = %#v, want %q", payload["error"], tt.wantError)
			}
		})
	}
}

func TestHandleGraphRoutes(t *testing.T) {
	t.Parallel()

	handler := NewHandler(serviceStub{
		startBuild: defaultStartBuild(nil),
		getGraphData: func(ctx context.Context, graphID string) (map[string]any, error) {
			_ = ctx
			if graphID != "graph-1" {
				t.Fatalf("graphID = %q", graphID)
			}
			return map[string]any{"graph_id": graphID, "node_count": 1, "edge_count": 0}, nil
		},
		deleteGraph: func(ctx context.Context, graphID string) error {
			_ = ctx
			if graphID == "missing" {
				return os.ErrNotExist
			}
			return nil
		},
		getTask: func(taskID string) (graphstore.TaskState, error) {
			if taskID == "missing" {
				return graphstore.TaskState{}, os.ErrNotExist
			}
			return graphstore.TaskState{TaskID: taskID, TaskType: "graph_build", Status: "processing"}, nil
		},
		listTasks: func(taskType string) ([]graphstore.TaskState, error) {
			if taskType != "graph_build" {
				t.Fatalf("taskType = %q", taskType)
			}
			return []graphstore.TaskState{{TaskID: "task-1", TaskType: taskType, Status: "pending"}}, nil
		},
		getProject: func(projectID string) (map[string]any, error) {
			if projectID == "missing" {
				return nil, os.ErrNotExist
			}
			return map[string]any{"project_id": projectID, "status": "graph_completed"}, nil
		},
		listProjects: func(limit int) ([]map[string]any, error) {
			if limit != 10 {
				t.Fatalf("limit = %d, want 10", limit)
			}
			return []map[string]any{{"project_id": "proj-1"}}, nil
		},
		deleteProject: func(projectID string) error {
			if projectID == "missing" {
				return os.ErrNotExist
			}
			return nil
		},
		resetProject: func(projectID string) (map[string]any, error) {
			if projectID == "missing" {
				return nil, os.ErrNotExist
			}
			return map[string]any{"project_id": projectID, "status": "ontology_generated", "graph_id": nil}, nil
		},
	})

	tests := []struct {
		name       string
		method     string
		target     string
		run        func(*Handler, *httptest.ResponseRecorder, *http.Request)
		wantStatus int
		wantError  string
	}{
		{
			name:       "graph data",
			method:     http.MethodGet,
			target:     "/api/graph/data/graph-1",
			run:        func(h *Handler, rec *httptest.ResponseRecorder, req *http.Request) { h.HandleGraphData(rec, req) },
			wantStatus: http.StatusOK,
		},
		{
			name:       "delete graph not found",
			method:     http.MethodDelete,
			target:     "/api/graph/delete/missing",
			run:        func(h *Handler, rec *httptest.ResponseRecorder, req *http.Request) { h.HandleDeleteGraph(rec, req) },
			wantStatus: http.StatusNotFound,
			wantError:  "file does not exist",
		},
		{
			name:       "task get",
			method:     http.MethodGet,
			target:     "/api/graph/task/task-1",
			run:        func(h *Handler, rec *httptest.ResponseRecorder, req *http.Request) { h.HandleTaskGet(rec, req) },
			wantStatus: http.StatusOK,
		},
		{
			name:       "task list",
			method:     http.MethodGet,
			target:     "/api/graph/tasks?task_type=graph_build",
			run:        func(h *Handler, rec *httptest.ResponseRecorder, req *http.Request) { h.HandleTaskList(rec, req) },
			wantStatus: http.StatusOK,
		},
		{
			name:       "project get",
			method:     http.MethodGet,
			target:     "/api/graph/project/proj-1",
			run:        func(h *Handler, rec *httptest.ResponseRecorder, req *http.Request) { h.HandleProjectRoute(rec, req) },
			wantStatus: http.StatusOK,
		},
		{
			name:       "project list",
			method:     http.MethodGet,
			target:     "/api/graph/project/list?limit=10",
			run:        func(h *Handler, rec *httptest.ResponseRecorder, req *http.Request) { h.HandleProjectList(rec, req) },
			wantStatus: http.StatusOK,
		},
		{
			name:       "project delete",
			method:     http.MethodDelete,
			target:     "/api/graph/project/proj-1",
			run:        func(h *Handler, rec *httptest.ResponseRecorder, req *http.Request) { h.HandleProjectRoute(rec, req) },
			wantStatus: http.StatusOK,
		},
		{
			name:       "project reset",
			method:     http.MethodPost,
			target:     "/api/graph/project/proj-1/reset",
			run:        func(h *Handler, rec *httptest.ResponseRecorder, req *http.Request) { h.HandleProjectRoute(rec, req) },
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(tt.method, tt.target, nil)
			rec := httptest.NewRecorder()
			tt.run(handler, rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
				var payload map[string]any
				if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
					t.Fatalf("Unmarshal: %v", err)
				}
				if tt.wantError != "" && payload["error"] != tt.wantError {
					t.Fatalf("error = %#v, want %q", payload["error"], tt.wantError)
				}
			}
		})
	}
}

func defaultStartBuild(fn func(context.Context, intgraph.BuildRequest) (intgraph.BuildResponse, error)) func(context.Context, intgraph.BuildRequest) (intgraph.BuildResponse, error) {
	if fn != nil {
		return fn
	}
	return func(context.Context, intgraph.BuildRequest) (intgraph.BuildResponse, error) {
		return intgraph.BuildResponse{}, nil
	}
}
