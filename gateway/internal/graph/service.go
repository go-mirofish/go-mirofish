package graph

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-mirofish/go-mirofish/gateway/internal/memory"
	graphstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/graph"
)

type MemoryClient interface {
	CreateGraphID() string
	CreateGraph(ctx context.Context, spec memory.GraphSpec) error
	SubmitOntology(ctx context.Context, graphID string, ontology memory.OntologySpec) error
	IngestBatch(ctx context.Context, graphID string, episodes []memory.Episode) error
	GetGraphData(ctx context.Context, graphID string) (memory.GraphData, error)
	DeleteGraph(ctx context.Context, graphID string) error
}

type Service struct {
	store  *graphstore.Store
	memory MemoryClient
}

func NewService(store *graphstore.Store, memoryClient MemoryClient) *Service {
	return &Service{store: store, memory: memoryClient}
}

func (s *Service) StartBuild(ctx context.Context, req BuildRequest) (BuildResponse, error) {
	if strings.TrimSpace(req.ProjectID) == "" {
		return BuildResponse{}, fmt.Errorf("graph.StartBuild: project_id is required")
	}
	project, err := s.store.LoadProject(req.ProjectID)
	if err != nil {
		return BuildResponse{}, err
	}
	if !req.Force && valueOr(project["graph_id"], "") != "" {
		return BuildResponse{}, fmt.Errorf("graph.StartBuild: graph already exists")
	}
	taskID := fmt.Sprintf("%d", time.Now().UnixNano())
	if err := s.store.CreateTask(taskID, "Build graph: "+valueOr(project["name"], "graph"), map[string]any{"project_id": req.ProjectID}); err != nil {
		return BuildResponse{}, err
	}

	project["status"] = "graph_building"
	project["graph_build_task_id"] = taskID
	project["updated_at"] = time.Now().Format(time.RFC3339)
	if err := s.store.SaveProject(req.ProjectID, project); err != nil {
		return BuildResponse{}, err
	}

	go s.run(context.Background(), req.ProjectID, taskID, req)

	return BuildResponse{
		ProjectID: req.ProjectID,
		TaskID:    taskID,
		Message:   "Graph build started",
	}, nil
}

func (s *Service) GetGraphData(ctx context.Context, graphID string) (map[string]any, error) {
	if strings.TrimSpace(graphID) == "" {
		return nil, fmt.Errorf("graph.GetGraphData: graph_id is required")
	}
	data, err := s.memory.GetGraphData(ctx, graphID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"graph_id":   data.GraphID,
		"nodes":      data.Nodes,
		"edges":      data.Edges,
		"node_count": data.NodeCount,
		"edge_count": data.EdgeCount,
	}, nil
}

func (s *Service) DeleteGraph(ctx context.Context, graphID string) error {
	if strings.TrimSpace(graphID) == "" {
		return fmt.Errorf("graph.DeleteGraph: graph_id is required")
	}
	return s.memory.DeleteGraph(ctx, graphID)
}

func (s *Service) GetTask(taskID string) (graphstore.TaskState, error) {
	if strings.TrimSpace(taskID) == "" {
		return graphstore.TaskState{}, fmt.Errorf("graph.GetTask: task_id is required")
	}
	return s.store.LoadTask(taskID)
}

func (s *Service) ListTasks(taskType string) ([]graphstore.TaskState, error) {
	return s.store.ListTasks(strings.TrimSpace(taskType))
}

func (s *Service) GetProject(projectID string) (map[string]any, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("graph.GetProject: project_id is required")
	}
	return s.store.LoadProject(projectID)
}

func (s *Service) ListProjects(limit int) ([]map[string]any, error) {
	return s.store.ListProjects(limit)
}

func (s *Service) DeleteProject(projectID string) error {
	if strings.TrimSpace(projectID) == "" {
		return fmt.Errorf("graph.DeleteProject: project_id is required")
	}
	if _, err := s.store.LoadProject(projectID); err != nil {
		return err
	}
	return s.store.DeleteProject(projectID)
}

func (s *Service) ResetProject(projectID string) (map[string]any, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("graph.ResetProject: project_id is required")
	}
	project, err := s.store.LoadProject(projectID)
	if err != nil {
		return nil, err
	}

	if ontology, ok := project["ontology"]; ok && ontology != nil {
		project["status"] = "ontology_generated"
	} else {
		project["status"] = "created"
	}
	project["graph_id"] = nil
	project["graph_build_task_id"] = nil
	project["error"] = nil
	project["updated_at"] = time.Now().Format(time.RFC3339)

	if err := s.store.SaveProject(projectID, project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *Service) run(ctx context.Context, projectID, taskID string, req BuildRequest) {
	project, err := s.store.LoadProject(projectID)
	if err != nil {
		s.fail(projectID, taskID, 0, "loading_project", err.Error(), map[string]any{})
		return
	}
	s.update(taskID, 5, "creating_graph", "Creating graph")
	graphID := s.memory.CreateGraphID()
	if err := s.memory.CreateGraph(ctx, memory.GraphSpec{
		GraphID:     graphID,
		Name:        firstString(project["name"], "go-mirofish graph"),
		Description: "go-mirofish social simulation graph",
	}); err != nil {
		s.fail(projectID, taskID, 5, "creating_graph", err.Error(), map[string]any{"graph_id": graphID})
		return
	}
	project["graph_id"] = graphID
	project["updated_at"] = time.Now().Format(time.RFC3339)
	_ = s.store.SaveProject(projectID, project)

	s.update(taskID, 15, "submitting_ontology", "Submitting ontology")
	ontology, _ := project["ontology"].(map[string]any)
	spec, err := memory.ToOntologySpec(ontology)
	if err != nil {
		s.fail(projectID, taskID, 15, "submitting_ontology", err.Error(), map[string]any{"graph_id": graphID})
		return
	}
	if err := s.memory.SubmitOntology(ctx, graphID, spec); err != nil {
		s.fail(projectID, taskID, 15, "submitting_ontology", err.Error(), map[string]any{"graph_id": graphID})
		return
	}

	s.update(taskID, 30, "chunking_text", "Chunking text")
	text := readExtractedText(projectID, s.store)
	chunkSize := req.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 500
	}
	chunkOverlap := req.ChunkOverlap
	if chunkOverlap < 0 {
		chunkOverlap = 50
	}
	episodes := BuildEpisodes(text, chunkSize, chunkOverlap, graphID)
	if len(episodes) == 0 {
		s.fail(projectID, taskID, 30, "chunking_text", "no extracted text available for graph build", map[string]any{"graph_id": graphID})
		return
	}

	s.update(taskID, 50, "ingesting_episodes", "Ingesting episodes")
	if err := s.memory.IngestBatch(ctx, graphID, episodes); err != nil {
		s.fail(projectID, taskID, 50, "ingesting_episodes", err.Error(), map[string]any{"graph_id": graphID, "chunk_count": len(episodes)})
		return
	}

	s.update(taskID, 90, "finalizing", "Finalizing graph build")
	project["graph_id"] = graphID
	project["status"] = "graph_completed"
	project["error"] = nil
	project["updated_at"] = time.Now().Format(time.RFC3339)
	if err := s.store.SaveProject(projectID, project); err != nil {
		s.fail(projectID, taskID, 90, "finalizing", err.Error(), map[string]any{"graph_id": graphID, "chunk_count": len(episodes)})
		return
	}
	task, err := s.store.LoadTask(taskID)
	if err != nil {
		return
	}
	task.Status = StatusCompleted
	task.Progress = 100
	task.Message = "Graph build complete"
	task.ProgressDetail = map[string]any{"stage": "completed", "graph_id": graphID, "chunk_count": len(episodes)}
	task.Result = map[string]any{
		"project_id":  projectID,
		"graph_id":    graphID,
		"chunk_count": len(episodes),
	}
	_ = s.store.SaveTask(task)
}

func (s *Service) update(taskID string, progress int, stage string, message string) {
	task, err := s.store.LoadTask(taskID)
	if err != nil {
		return
	}
	task.Status = StatusProcessing
	task.Progress = progress
	task.Message = message
	task.ProgressDetail = map[string]any{"stage": stage}
	_ = s.store.SaveTask(task)
}

func (s *Service) fail(projectID, taskID string, progress int, stage string, message string, partial map[string]any) {
	project, err := s.store.LoadProject(projectID)
	if err == nil {
		project["status"] = "graph_failed"
		project["error"] = message
		project["updated_at"] = time.Now().Format(time.RFC3339)
		for key, value := range partial {
			project[key] = value
		}
		_ = s.store.SaveProject(projectID, project)
	}
	task, err := s.store.LoadTask(taskID)
	if err != nil {
		return
	}
	task.Status = StatusFailed
	task.Progress = progress
	task.Message = "Build failed: " + message
	task.Error = message
	task.ProgressDetail = map[string]any{"stage": stage}
	if len(partial) > 0 {
		task.Result = partial
	}
	_ = s.store.SaveTask(task)
}

func valueOr(value any, fallback string) string {
	if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
		return s
	}
	return fallback
}

func firstString(value any, fallback string) string { return valueOr(value, fallback) }

func readExtractedText(projectID string, store *graphstore.Store) string {
	path := filepath.Join(store.ProjectsDir, projectID, "extracted_text.txt")
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(raw)
}

func FilterEntitiesFromGraphData(graphData map[string]any, entityTypes []string, enrich bool) map[string]any {
	allowed := map[string]bool{}
	for _, entityType := range entityTypes {
		if entityType != "" {
			allowed[entityType] = true
		}
	}

	nodes, _ := graphData["nodes"].([]map[string]any)
	if nodes == nil {
		switch rawNodes := graphData["nodes"].(type) {
		case []memory.GraphNode:
			nodes = make([]map[string]any, 0, len(rawNodes))
			for _, node := range rawNodes {
				nodes = append(nodes, map[string]any{
					"uuid":       node.UUID,
					"name":       node.Name,
					"labels":     node.Labels,
					"summary":    node.Summary,
					"attributes": node.Attributes,
					"created_at": node.CreatedAt,
				})
			}
		case []any:
			nodes = make([]map[string]any, 0, len(rawNodes))
			for _, raw := range rawNodes {
				switch node := raw.(type) {
				case map[string]any:
					nodes = append(nodes, node)
				case memory.GraphNode:
					nodes = append(nodes, map[string]any{
						"uuid":       node.UUID,
						"name":       node.Name,
						"labels":     node.Labels,
						"summary":    node.Summary,
						"attributes": node.Attributes,
						"created_at": node.CreatedAt,
					})
				}
			}
		}
	}
	edges, _ := graphData["edges"].([]map[string]any)
	if edges == nil {
		switch rawEdges := graphData["edges"].(type) {
		case []memory.GraphEdge:
			edges = make([]map[string]any, 0, len(rawEdges))
			for _, edge := range rawEdges {
				edges = append(edges, map[string]any{
					"uuid":             edge.UUID,
					"name":             edge.Name,
					"fact":             edge.Fact,
					"fact_type":        edge.FactType,
					"source_node_uuid": edge.SourceNodeUUID,
					"target_node_uuid": edge.TargetNodeUUID,
					"source_node_name": edge.SourceNodeName,
					"target_node_name": edge.TargetNodeName,
					"attributes":       edge.Attributes,
					"created_at":       edge.CreatedAt,
					"valid_at":         edge.ValidAt,
					"invalid_at":       edge.InvalidAt,
					"expired_at":       edge.ExpiredAt,
					"episodes":         edge.Episodes,
				})
			}
		case []any:
			edges = make([]map[string]any, 0, len(rawEdges))
			for _, raw := range rawEdges {
				switch edge := raw.(type) {
				case map[string]any:
					edges = append(edges, edge)
				case memory.GraphEdge:
					edges = append(edges, map[string]any{
						"uuid":             edge.UUID,
						"name":             edge.Name,
						"fact":             edge.Fact,
						"fact_type":        edge.FactType,
						"source_node_uuid": edge.SourceNodeUUID,
						"target_node_uuid": edge.TargetNodeUUID,
						"source_node_name": edge.SourceNodeName,
						"target_node_name": edge.TargetNodeName,
						"attributes":       edge.Attributes,
						"created_at":       edge.CreatedAt,
						"valid_at":         edge.ValidAt,
						"invalid_at":       edge.InvalidAt,
						"expired_at":       edge.ExpiredAt,
						"episodes":         edge.Episodes,
					})
				}
			}
		}
	}

	filtered := make([]map[string]any, 0)
	entityTypesFound := map[string]bool{}
	for _, node := range nodes {
		labels := toStringSlice(node["labels"])
		customLabels := make([]string, 0, len(labels))
		for _, label := range labels {
			if label != "Entity" && label != "Node" {
				customLabels = append(customLabels, label)
			}
		}
		if len(customLabels) == 0 {
			continue
		}
		entityType := customLabels[0]
		if len(allowed) > 0 && !allowed[entityType] {
			continue
		}
		entityTypesFound[entityType] = true

		entity := map[string]any{
			"uuid":       node["uuid"],
			"name":       node["name"],
			"labels":     labels,
			"summary":    node["summary"],
			"attributes": node["attributes"],
		}

		if enrich {
			relatedEdges := make([]map[string]any, 0)
			relatedNodeUUIDs := map[string]bool{}
			uuid := fmt.Sprint(node["uuid"])
			for _, edge := range edges {
				source := fmt.Sprint(edge["source_node_uuid"])
				target := fmt.Sprint(edge["target_node_uuid"])
				switch {
				case source == uuid:
					relatedEdges = append(relatedEdges, map[string]any{
						"direction":        "outgoing",
						"edge_name":        edge["name"],
						"fact":             edge["fact"],
						"target_node_uuid": target,
					})
					relatedNodeUUIDs[target] = true
				case target == uuid:
					relatedEdges = append(relatedEdges, map[string]any{
						"direction":        "incoming",
						"edge_name":        edge["name"],
						"fact":             edge["fact"],
						"source_node_uuid": source,
					})
					relatedNodeUUIDs[source] = true
				}
			}
			relatedNodes := make([]map[string]any, 0)
			for _, candidate := range nodes {
				if relatedNodeUUIDs[fmt.Sprint(candidate["uuid"])] {
					relatedNodes = append(relatedNodes, candidate)
				}
			}
			entity["related_edges"] = relatedEdges
			entity["related_nodes"] = relatedNodes
		}
		filtered = append(filtered, entity)
	}

	entityTypeList := make([]string, 0, len(entityTypesFound))
	for entityType := range entityTypesFound {
		entityTypeList = append(entityTypeList, entityType)
	}
	sort.Strings(entityTypeList)

	return map[string]any{
		"entities":       filtered,
		"entity_types":   entityTypeList,
		"total_count":    len(nodes),
		"filtered_count": len(filtered),
	}
}

func toStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, fmt.Sprint(item))
		}
		return out
	default:
		return nil
	}
}
