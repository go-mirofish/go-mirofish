package graph

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-mirofish/go-mirofish/gateway/internal/memory"
	graphstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/graph"
)

type fakeMemoryClient struct {
	graphID        string
	createErr      error
	ontologyErr    error
	ingestErr      error
	deleteGraphErr error
	graphData      memory.GraphData
	ingestHook     func()
}

func (f *fakeMemoryClient) CreateGraphID() string { return f.graphID }

func (f *fakeMemoryClient) CreateGraph(ctx context.Context, spec memory.GraphSpec) error {
	_ = ctx
	_ = spec
	return f.createErr
}

func (f *fakeMemoryClient) SubmitOntology(ctx context.Context, graphID string, ontology memory.OntologySpec) error {
	_ = ctx
	_ = graphID
	_ = ontology
	return f.ontologyErr
}

func (f *fakeMemoryClient) IngestBatch(ctx context.Context, graphID string, episodes []memory.Episode) error {
	_ = ctx
	_ = graphID
	_ = episodes
	if f.ingestHook != nil {
		f.ingestHook()
	}
	return f.ingestErr
}

func (f *fakeMemoryClient) GetGraphData(ctx context.Context, graphID string) (memory.GraphData, error) {
	_ = ctx
	if f.graphData.GraphID == "" {
		f.graphData.GraphID = graphID
	}
	return f.graphData, nil
}

func (f *fakeMemoryClient) DeleteGraph(ctx context.Context, graphID string) error {
	_ = ctx
	_ = graphID
	return f.deleteGraphErr
}

func TestServiceStartBuildScenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		configure func(*testing.T, *graphstore.Store, string, string) *fakeMemoryClient
		wait      func(*testing.T, *graphstore.Store, string, string) (graphstore.TaskState, map[string]any)
		assert    func(*testing.T, graphstore.TaskState, map[string]any, string)
	}{
		{
			name: "build success",
			configure: func(t *testing.T, store *graphstore.Store, tasksDir, projectsDir string) *fakeMemoryClient {
				t.Helper()
				_ = store
				_ = tasksDir
				_ = projectsDir
				return &fakeMemoryClient{graphID: "graph-success"}
			},
			wait: func(t *testing.T, store *graphstore.Store, projectID, taskID string) (graphstore.TaskState, map[string]any) {
				t.Helper()
				task := waitForTaskStatus(t, store, taskID, StatusCompleted)
				project := waitForProjectStatus(t, store, projectID, "graph_completed")
				return task, project
			},
			assert: func(t *testing.T, task graphstore.TaskState, project map[string]any, projectID string) {
				t.Helper()
				if got := valueOr(project["graph_id"], ""); got != "graph-success" {
					t.Fatalf("project graph_id = %q, want %q", got, "graph-success")
				}
				if got := task.Result["project_id"]; got != projectID {
					t.Fatalf("task result project_id = %#v, want %q", got, projectID)
				}
				if got := task.Result["graph_id"]; got != "graph-success" {
					t.Fatalf("task result graph_id = %#v, want %q", got, "graph-success")
				}
				if got := task.Result["chunk_count"]; got != float64(4) {
					t.Fatalf("task result chunk_count = %#v, want 4", got)
				}
			},
		},
		{
			name: "memory zep failure",
			configure: func(t *testing.T, store *graphstore.Store, tasksDir, projectsDir string) *fakeMemoryClient {
				t.Helper()
				_ = store
				_ = tasksDir
				_ = projectsDir
				return &fakeMemoryClient{
					graphID:   "graph-zep-fail",
					createErr: errors.New("zep unavailable"),
				}
			},
			wait: func(t *testing.T, store *graphstore.Store, projectID, taskID string) (graphstore.TaskState, map[string]any) {
				t.Helper()
				task := waitForTaskStatus(t, store, taskID, StatusFailed)
				project := waitForProjectStatus(t, store, projectID, "graph_failed")
				return task, project
			},
			assert: func(t *testing.T, task graphstore.TaskState, project map[string]any, projectID string) {
				t.Helper()
				if task.Error != "zep unavailable" {
					t.Fatalf("task error = %q, want zep unavailable", task.Error)
				}
				if got := task.Progress; got != 5 {
					t.Fatalf("task progress = %d, want 5", got)
				}
				if got := task.Result["graph_id"]; got != "graph-zep-fail" {
					t.Fatalf("task result graph_id = %#v, want %q", got, "graph-zep-fail")
				}
				if got := valueOr(project["status"], ""); got != "graph_failed" {
					t.Fatalf("project status = %q, want graph_failed", got)
				}
				if got := valueOr(project["graph_id"], ""); got != "graph-zep-fail" {
					t.Fatalf("project graph_id = %q, want %q", got, "graph-zep-fail")
				}
			},
		},
		{
			name: "persistence failure",
			configure: func(t *testing.T, store *graphstore.Store, tasksDir, projectsDir string) *fakeMemoryClient {
				t.Helper()
				_ = tasksDir
				blocker := filepath.Join(filepath.Dir(projectsDir), "missing-projects")
				if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
					t.Fatalf("WriteFile blocker: %v", err)
				}
				return &fakeMemoryClient{
					graphID: "graph-persist-fail",
					ingestHook: func() {
						store.ProjectsDir = blocker
					},
				}
			},
			wait: func(t *testing.T, store *graphstore.Store, projectID, taskID string) (graphstore.TaskState, map[string]any) {
				t.Helper()
				task := waitForTaskStatus(t, store, taskID, StatusFailed)
				project := mustLoadProject(t, store, projectID)
				return task, project
			},
			assert: func(t *testing.T, task graphstore.TaskState, project map[string]any, projectID string) {
				t.Helper()
				if got := task.Progress; got != 90 {
					t.Fatalf("task progress = %d, want 90", got)
				}
				if got := valueOr(project["status"], ""); got != "graph_building" {
					t.Fatalf("project status = %q, want graph_building", got)
				}
				if got := valueOr(project["graph_build_task_id"], ""); got == "" {
					t.Fatalf("expected graph_build_task_id to remain persisted")
				}
				if got := valueOr(project["graph_id"], ""); got != "graph-persist-fail" {
					t.Fatalf("project graph_id = %q, want %q", got, "graph-persist-fail")
				}
				if got := task.Result["chunk_count"]; got != float64(4) {
					t.Fatalf("task result chunk_count = %#v, want 4", got)
				}
			},
		},
		{
			name: "partial failure persistence",
			configure: func(t *testing.T, store *graphstore.Store, tasksDir, projectsDir string) *fakeMemoryClient {
				t.Helper()
				_ = projectsDir
				missingTasksDir := filepath.Join(filepath.Dir(tasksDir), "missing-tasks")
				return &fakeMemoryClient{
					graphID: "graph-partial",
					ingestHook: func() {
						store.TasksDir = missingTasksDir
					},
				}
			},
			wait: func(t *testing.T, store *graphstore.Store, projectID, taskID string) (graphstore.TaskState, map[string]any) {
				t.Helper()
				project := waitForProjectStatus(t, store, projectID, "graph_completed")
				task := mustLoadTask(t, store, taskID)
				return task, project
			},
			assert: func(t *testing.T, task graphstore.TaskState, project map[string]any, projectID string) {
				t.Helper()
				if got := valueOr(project["graph_id"], ""); got != "graph-partial" {
					t.Fatalf("project graph_id = %q, want %q", got, "graph-partial")
				}
				if task.Status != StatusProcessing {
					t.Fatalf("task status = %q, want %q", task.Status, StatusProcessing)
				}
				if task.Progress != 50 {
					t.Fatalf("task progress = %d, want 50", task.Progress)
				}
				if got := task.ProgressDetail["stage"]; got != "ingesting_episodes" {
					t.Fatalf("task stage = %#v, want ingesting_episodes", got)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, store, projectID := newGraphServiceFixture(t, tt.configure)
			resp, err := service.StartBuild(context.Background(), BuildRequest{
				ProjectID:    projectID,
				ChunkSize:    16,
				ChunkOverlap: 4,
			})
			if err != nil {
				t.Fatalf("StartBuild: %v", err)
			}

			task, project := tt.wait(t, store, projectID, resp.TaskID)
			tt.assert(t, task, project, projectID)
		})
	}
}

func newGraphServiceFixture(
	t *testing.T,
	configure func(*testing.T, *graphstore.Store, string, string) *fakeMemoryClient,
) (*Service, *graphstore.Store, string) {
	t.Helper()

	root := t.TempDir()
	tasksDir := filepath.Join(root, "tasks")
	projectsDir := filepath.Join(root, "projects")
	writeStore := graphstore.New(tasksDir, projectsDir)
	readStore := graphstore.New(tasksDir, projectsDir)
	projectID := "proj-1"
	projectDir := filepath.Join(projectsDir, projectID)

	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	project := map[string]any{
		"project_id": projectID,
		"name":       "Project",
		"status":     "ontology_generated",
		"ontology": map[string]any{
			"entity_types": []map[string]any{
				{
					"name":        "PERSON",
					"description": "Person",
					"attributes":  []map[string]any{{"name": "name", "type": "string", "description": "Name"}},
				},
			},
			"edge_types": []map[string]any{
				{
					"name":           "KNOWS",
					"description":    "Knows",
					"attributes":     []map[string]any{},
					"source_targets": []map[string]any{{"source": "PERSON", "target": "PERSON"}},
				},
			},
		},
	}
	raw, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "project.json"), raw, 0o644); err != nil {
		t.Fatalf("WriteFile project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "extracted_text.txt"), []byte("alpha beta gamma delta epsilon zeta eta theta"), 0o644); err != nil {
		t.Fatalf("WriteFile text: %v", err)
	}

	return NewService(writeStore, configure(t, writeStore, tasksDir, projectsDir)), readStore, projectID
}

func waitForTaskStatus(t *testing.T, store *graphstore.Store, taskID, wantStatus string) graphstore.TaskState {
	t.Helper()

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		task, err := store.LoadTask(taskID)
		if err == nil && task.Status == wantStatus {
			return task
		}
		time.Sleep(25 * time.Millisecond)
	}

	task, err := store.LoadTask(taskID)
	if err != nil {
		t.Fatalf("LoadTask: %v", err)
	}
	t.Fatalf("task status = %q, want %q", task.Status, wantStatus)
	return graphstore.TaskState{}
}

func waitForProjectStatus(t *testing.T, store *graphstore.Store, projectID, wantStatus string) map[string]any {
	t.Helper()

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		project, err := store.LoadProject(projectID)
		if err == nil && valueOr(project["status"], "") == wantStatus {
			return project
		}
		time.Sleep(25 * time.Millisecond)
	}

	project, err := store.LoadProject(projectID)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	t.Fatalf("project status = %q, want %q", valueOr(project["status"], ""), wantStatus)
	return nil
}

func mustLoadProject(t *testing.T, store *graphstore.Store, projectID string) map[string]any {
	t.Helper()

	project, err := store.LoadProject(projectID)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	return project
}

func mustLoadTask(t *testing.T, store *graphstore.Store, taskID string) graphstore.TaskState {
	t.Helper()

	task, err := store.LoadTask(taskID)
	if err != nil {
		t.Fatalf("LoadTask: %v", err)
	}
	return task
}

func TestServiceControlPlaneHelpers(t *testing.T) {
	t.Parallel()

	service, store, projectID := newGraphServiceFixture(t, func(t *testing.T, store *graphstore.Store, tasksDir, projectsDir string) *fakeMemoryClient {
		t.Helper()
		_ = store
		_ = tasksDir
		_ = projectsDir
		return &fakeMemoryClient{
			graphID: "graph-1",
			graphData: memory.GraphData{
				GraphID: "graph-1",
				Nodes: []memory.GraphNode{
					{UUID: "node-1", Name: "Alice", Labels: []string{"Entity", "Person"}, Attributes: map[string]any{"role": "founder"}},
				},
				Edges: []memory.GraphEdge{
					{UUID: "edge-1", Name: "KNOWS", Fact: "Alice knows Bob", SourceNodeUUID: "node-1", TargetNodeUUID: "node-2"},
				},
				NodeCount: 1,
				EdgeCount: 1,
			},
		}
	})

	if err := store.CreateTask("task-1", "graph_build", map[string]any{"project_id": projectID}); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	data, err := service.GetGraphData(context.Background(), "graph-1")
	if err != nil {
		t.Fatalf("GetGraphData: %v", err)
	}
	if data["graph_id"] != "graph-1" {
		t.Fatalf("graph_id = %#v, want graph-1", data["graph_id"])
	}

	project, err := service.GetProject(projectID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if project["project_id"] != projectID {
		t.Fatalf("project_id = %#v, want %q", project["project_id"], projectID)
	}

	projects, err := service.ListProjects(10)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("len(projects) = %d, want 1", len(projects))
	}

	task, err := service.GetTask("task-1")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if task.TaskID != "task-1" {
		t.Fatalf("task_id = %q, want task-1", task.TaskID)
	}

	tasks, err := service.ListTasks("graph_build")
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("len(tasks) = %d, want 1", len(tasks))
	}

	resetProject, err := service.ResetProject(projectID)
	if err != nil {
		t.Fatalf("ResetProject: %v", err)
	}
	if got := valueOr(resetProject["status"], ""); got != "ontology_generated" {
		t.Fatalf("reset status = %q, want ontology_generated", got)
	}
	if resetProject["graph_id"] != nil {
		t.Fatalf("reset graph_id = %#v, want nil", resetProject["graph_id"])
	}

	if err := service.DeleteGraph(context.Background(), "graph-1"); err != nil {
		t.Fatalf("DeleteGraph: %v", err)
	}

	if err := service.DeleteProject(projectID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	if _, err := store.LoadProject(projectID); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("LoadProject after delete err = %v, want os.ErrNotExist", err)
	}
}

func TestFilterEntitiesFromGraphData(t *testing.T) {
	t.Parallel()

	result := FilterEntitiesFromGraphData(map[string]any{
		"nodes": []memory.GraphNode{
			{UUID: "node-1", Name: "Alice", Labels: []string{"Entity", "Person"}, Summary: "Founder", Attributes: map[string]any{"role": "founder"}},
			{UUID: "node-2", Name: "Bob", Labels: []string{"Entity", "Company"}, Summary: "Company", Attributes: map[string]any{}},
		},
		"edges": []memory.GraphEdge{
			{UUID: "edge-1", Name: "WORKS_AT", Fact: "Alice works at Bob", SourceNodeUUID: "node-1", TargetNodeUUID: "node-2"},
		},
	}, []string{"Person"}, true)

	entities, _ := result["entities"].([]map[string]any)
	if len(entities) != 1 {
		t.Fatalf("len(entities) = %d, want 1", len(entities))
	}
	if entities[0]["name"] != "Alice" {
		t.Fatalf("name = %#v, want Alice", entities[0]["name"])
	}
	if result["filtered_count"] != 1 {
		t.Fatalf("filtered_count = %#v, want 1", result["filtered_count"])
	}
}
