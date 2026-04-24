package graphstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type TaskState struct {
	TaskID         string         `json:"task_id"`
	TaskType       string         `json:"task_type"`
	Status         string         `json:"status"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
	Progress       int            `json:"progress"`
	Message        string         `json:"message"`
	ProgressDetail map[string]any `json:"progress_detail"`
	Result         map[string]any `json:"result,omitempty"`
	Error          string         `json:"error,omitempty"`
	Metadata       map[string]any `json:"metadata"`
}

type Store struct {
	TasksDir    string
	ProjectsDir string
}

func New(tasksDir, projectsDir string) *Store {
	return &Store{TasksDir: tasksDir, ProjectsDir: projectsDir}
}

func (s *Store) taskPath(taskID string) string {
	return filepath.Join(s.TasksDir, taskID+".json")
}

func (s *Store) projectPath(projectID string) string {
	return filepath.Join(s.ProjectsDir, projectID, "project.json")
}

func (s *Store) CreateTask(taskID, taskType string, metadata map[string]any) error {
	if err := os.MkdirAll(s.TasksDir, 0o755); err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	task := TaskState{
		TaskID:         taskID,
		TaskType:       taskType,
		Status:         "pending",
		CreatedAt:      now,
		UpdatedAt:      now,
		Progress:       0,
		Message:        "",
		ProgressDetail: map[string]any{},
		Metadata:       metadata,
	}
	return s.writeTask(task)
}

func (s *Store) LoadTask(taskID string) (TaskState, error) {
	var task TaskState
	raw, err := os.ReadFile(s.taskPath(taskID))
	if err != nil {
		return task, err
	}
	err = json.Unmarshal(raw, &task)
	return task, err
}

func (s *Store) SaveTask(task TaskState) error {
	task.UpdatedAt = time.Now().Format(time.RFC3339)
	return s.writeTask(task)
}

func (s *Store) writeTask(task TaskState) error {
	raw, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.taskPath(task.TaskID), raw, 0o644)
}

func (s *Store) LoadProject(projectID string) (map[string]any, error) {
	raw, err := os.ReadFile(s.projectPath(projectID))
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *Store) SaveProject(projectID string, payload map[string]any) error {
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.projectPath(projectID), raw, 0o644)
}

func (s *Store) DeleteProject(projectID string) error {
	return os.RemoveAll(filepath.Dir(s.projectPath(projectID)))
}

func (s *Store) ListTasks(taskType string) ([]TaskState, error) {
	if err := os.MkdirAll(s.TasksDir, 0o755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.TasksDir)
	if err != nil {
		return nil, err
	}

	tasks := make([]TaskState, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		task, err := s.LoadTask(entry.Name()[:len(entry.Name())-len(filepath.Ext(entry.Name()))])
		if err != nil {
			continue
		}
		if taskType != "" && task.TaskType != taskType {
			continue
		}
		tasks = append(tasks, task)
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt > tasks[j].CreatedAt
	})

	return tasks, nil
}

func (s *Store) ListProjects(limit int) ([]map[string]any, error) {
	if err := os.MkdirAll(s.ProjectsDir, 0o755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.ProjectsDir)
	if err != nil {
		return nil, err
	}

	projects := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		project, err := s.LoadProject(entry.Name())
		if err != nil {
			continue
		}
		projects = append(projects, project)
	}

	sort.Slice(projects, func(i, j int) bool {
		ic, _ := projects[i]["created_at"].(string)
		jc, _ := projects[j]["created_at"].(string)
		return ic > jc
	})

	if limit > 0 && len(projects) > limit {
		projects = projects[:limit]
	}
	return projects, nil
}
