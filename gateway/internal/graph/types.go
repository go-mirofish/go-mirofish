package graph

type BuildRequest struct {
	ProjectID    string `json:"project_id"`
	GraphName    string `json:"graph_name,omitempty"`
	ChunkSize    int    `json:"chunk_size,omitempty"`
	ChunkOverlap int    `json:"chunk_overlap,omitempty"`
	Force        bool   `json:"force,omitempty"`
}

type BuildResponse struct {
	ProjectID string `json:"project_id"`
	TaskID    string `json:"task_id"`
	Message   string `json:"message"`
}
