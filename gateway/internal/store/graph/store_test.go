package graphstore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRoundTrips(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(*testing.T, *Store)
	}{
		{
			name: "task round trip",
			run: func(t *testing.T, store *Store) {
				t.Helper()

				if err := store.CreateTask("task-1", "Build graph", map[string]any{"project_id": "proj-1"}); err != nil {
					t.Fatalf("CreateTask: %v", err)
				}

				task, err := store.LoadTask("task-1")
				if err != nil {
					t.Fatalf("LoadTask: %v", err)
				}
				task.Status = "processing"
				task.Progress = 60
				task.Message = "Persisted"
				task.ProgressDetail = map[string]any{"stage": "saving"}

				if err := store.SaveTask(task); err != nil {
					t.Fatalf("SaveTask: %v", err)
				}

				got, err := store.LoadTask("task-1")
				if err != nil {
					t.Fatalf("LoadTask after save: %v", err)
				}
				if got.Status != "processing" || got.Progress != 60 {
					t.Fatalf("unexpected task: %#v", got)
				}
				if got.Metadata["project_id"] != "proj-1" {
					t.Fatalf("metadata project_id = %#v, want proj-1", got.Metadata["project_id"])
				}
			},
		},
		{
			name: "project round trip",
			run: func(t *testing.T, store *Store) {
				t.Helper()

				projectDir := filepath.Join(store.ProjectsDir, "proj-1")
				if err := os.MkdirAll(projectDir, 0o755); err != nil {
					t.Fatalf("MkdirAll: %v", err)
				}

				payload := map[string]any{"project_id": "proj-1", "status": "graph_building"}
				if err := store.SaveProject("proj-1", payload); err != nil {
					t.Fatalf("SaveProject: %v", err)
				}

				got, err := store.LoadProject("proj-1")
				if err != nil {
					t.Fatalf("LoadProject: %v", err)
				}
				if got["status"] != "graph_building" {
					t.Fatalf("status = %#v, want graph_building", got["status"])
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			store := New(filepath.Join(root, "tasks"), filepath.Join(root, "projects"))
			tt.run(t, store)
		})
	}
}

func TestStoreFailurePaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "create task mkdir failure",
			run: func(t *testing.T) {
				t.Helper()

				root := t.TempDir()
				blocker := filepath.Join(root, "blocker")
				if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
					t.Fatalf("WriteFile blocker: %v", err)
				}

				store := New(filepath.Join(blocker, "tasks"), filepath.Join(root, "projects"))
				if err := store.CreateTask("task-1", "Build graph", nil); err == nil {
					t.Fatalf("expected CreateTask error")
				}
			},
		},
		{
			name: "load task invalid json",
			run: func(t *testing.T) {
				t.Helper()

				root := t.TempDir()
				store := New(filepath.Join(root, "tasks"), filepath.Join(root, "projects"))
				if err := os.MkdirAll(store.TasksDir, 0o755); err != nil {
					t.Fatalf("MkdirAll tasks: %v", err)
				}
				if err := os.WriteFile(store.taskPath("task-1"), []byte("{"), 0o644); err != nil {
					t.Fatalf("WriteFile task: %v", err)
				}
				if _, err := store.LoadTask("task-1"); err == nil {
					t.Fatalf("expected LoadTask error")
				}
			},
		},
		{
			name: "save task missing required fields",
			run: func(t *testing.T) {
				t.Helper()

				root := t.TempDir()
				store := New(filepath.Join(root, "tasks"), filepath.Join(root, "projects"))
				if err := store.SaveTask(TaskState{TaskID: "task-1"}); err == nil {
					t.Fatalf("expected SaveTask error")
				}
			},
		},
		{
			name: "load project invalid json",
			run: func(t *testing.T) {
				t.Helper()

				root := t.TempDir()
				store := New(filepath.Join(root, "tasks"), filepath.Join(root, "projects"))
				projectDir := filepath.Join(store.ProjectsDir, "proj-1")
				if err := os.MkdirAll(projectDir, 0o755); err != nil {
					t.Fatalf("MkdirAll project: %v", err)
				}
				if err := os.WriteFile(store.projectPath("proj-1"), []byte("{"), 0o644); err != nil {
					t.Fatalf("WriteFile project: %v", err)
				}
				if _, err := store.LoadProject("proj-1"); err == nil {
					t.Fatalf("expected LoadProject error")
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t)
		})
	}
}

func TestStoreListAndDelete(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := New(filepath.Join(root, "tasks"), filepath.Join(root, "projects"))

	if err := store.CreateTask("task-1", "graph_build", map[string]any{"project_id": "proj-1"}); err != nil {
		t.Fatalf("CreateTask task-1: %v", err)
	}
	if err := store.CreateTask("task-2", "simulation_prepare", map[string]any{"simulation_id": "sim-1"}); err != nil {
		t.Fatalf("CreateTask task-2: %v", err)
	}

	projectDir := filepath.Join(store.ProjectsDir, "proj-1")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("MkdirAll project: %v", err)
	}
	if err := store.SaveProject("proj-1", map[string]any{
		"project_id": "proj-1",
		"created_at": "2026-04-24T00:00:00Z",
	}); err != nil {
		t.Fatalf("SaveProject proj-1: %v", err)
	}
	if _, err := os.Stat(store.projectPath("proj-1")); err != nil {
		t.Fatalf("expected project.json to exist: %v", err)
	}

	tasks, err := store.ListTasks("graph_build")
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].TaskID != "task-1" {
		t.Fatalf("unexpected tasks: %#v", tasks)
	}

	projects, err := store.ListProjects(10)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 1 || projects[0]["project_id"] != "proj-1" {
		t.Fatalf("unexpected projects: %#v", projects)
	}

	if err := store.DeleteProject("proj-1"); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	if _, err := store.LoadProject("proj-1"); !os.IsNotExist(err) {
		t.Fatalf("LoadProject after delete err = %v, want os.ErrNotExist", err)
	}
}
