package ontologyhttp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	intontology "github.com/go-mirofish/go-mirofish/gateway/internal/ontology"
	intprovider "github.com/go-mirofish/go-mirofish/gateway/internal/provider"
	localfs "github.com/go-mirofish/go-mirofish/gateway/internal/store/localfs"
)

type ProxyFunc func(http.ResponseWriter, *http.Request)
type JSONFunc func(http.ResponseWriter, int, any)

type Handler struct {
	store     *localfs.Store
	proxy     ProxyFunc
	writeJSON JSONFunc
}

var newExecutor = func(cfg intprovider.Config) intprovider.Executor {
	return intprovider.NewExecutor(cfg, nil)
}

func NewHandler(store *localfs.Store, proxy ProxyFunc, writeJSON JSONFunc) *Handler {
	return &Handler{store: store, proxy: proxy, writeJSON: writeJSON}
}

type uploadFile struct {
	originalFilename string
	size             int64
	text             string
}

func (h *Handler) HandleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if shouldProxy(r) {
		h.proxy(w, r)
		return
	}
	projectName := strings.TrimSpace(r.FormValue("project_name"))
	if projectName == "" {
		projectName = "Unnamed Project"
	}
	simulationRequirement := strings.TrimSpace(r.FormValue("simulation_requirement"))
	if simulationRequirement == "" {
		h.writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": "Simulation requirement is required"})
		return
	}
	additionalContext := strings.TrimSpace(r.FormValue("additional_context"))
	files, extractedText, err := h.collectFiles(r)
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "error": err.Error()})
		return
	}
	projectID, projectPayload, err := h.createProjectRecord(projectName)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	var fileEntries []map[string]any
	totalText := strings.Builder{}
	for _, file := range files {
		fileEntries = append(fileEntries, map[string]any{"filename": file.originalFilename, "size": file.size})
		totalText.WriteString("\n\n=== ")
		totalText.WriteString(file.originalFilename)
		totalText.WriteString(" ===\n")
		totalText.WriteString(file.text)
	}
	projectPayload["simulation_requirement"] = simulationRequirement
	projectPayload["files"] = fileEntries
	projectPayload["total_text_length"] = len(totalText.String())
	if err := h.saveProjectText(projectID, totalText.String()); err != nil {
		h.writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	ontology, err := h.generateOntology(r.Context(), simulationRequirement, extractedText, additionalContext)
	if err != nil {
		h.writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	projectPayload["ontology"] = map[string]any{
		"entity_types": ontology["entity_types"],
		"edge_types":   ontology["edge_types"],
	}
	projectPayload["analysis_summary"] = ontology["analysis_summary"]
	projectPayload["status"] = "ontology_generated"
	projectPayload["updated_at"] = time.Now().Format(time.RFC3339)
	if err := h.store.WriteProject(projectID, projectPayload); err != nil {
		h.writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "error": err.Error()})
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"data": map[string]any{
			"project_id":        projectID,
			"project_name":      projectName,
			"ontology":          projectPayload["ontology"],
			"analysis_summary":  projectPayload["analysis_summary"],
			"files":             fileEntries,
			"total_text_length": projectPayload["total_text_length"],
		},
	})
}

func shouldProxy(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "multipart/form-data") {
		return true
	}
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		return true
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		return false
	}
	for _, file := range files {
		if strings.ToLower(filepath.Ext(file.Filename)) == ".pdf" {
			return true
		}
	}
	return false
}

func (h *Handler) collectFiles(r *http.Request) ([]uploadFile, string, error) {
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		return nil, "", fmt.Errorf("file upload is required")
	}
	tmpProjectDir := h.store.ProjectDir("tmp")
	filesDir := filepath.Join(tmpProjectDir, "files")
	_ = os.RemoveAll(tmpProjectDir)
	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		return nil, "", err
	}
	defer os.RemoveAll(tmpProjectDir)

	combined := strings.Builder{}
	var results []uploadFile
	for _, header := range files {
		ext := strings.ToLower(filepath.Ext(header.Filename))
		if ext != ".txt" && ext != ".md" && ext != ".markdown" {
			return nil, "", fmt.Errorf("unsupported file format: %s", ext)
		}
		src, err := header.Open()
		if err != nil {
			return nil, "", err
		}
		raw, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			return nil, "", err
		}
		text := preprocessText(string(raw))
		results = append(results, uploadFile{
			originalFilename: header.Filename,
			size:             int64(len(raw)),
			text:             text,
		})
		if combined.Len() > 0 {
			combined.WriteString("\n\n---\n\n")
		}
		combined.WriteString(text)
	}
	return results, combined.String(), nil
}

func (h *Handler) createProjectRecord(name string) (string, map[string]any, error) {
	projectID := "proj_" + fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	now := time.Now().Format(time.RFC3339)
	payload := map[string]any{
		"project_id":             projectID,
		"name":                   name,
		"status":                 "created",
		"created_at":             now,
		"updated_at":             now,
		"files":                  []any{},
		"total_text_length":      0,
		"ontology":               nil,
		"analysis_summary":       nil,
		"graph_id":               nil,
		"graph_build_task_id":    nil,
		"simulation_requirement": nil,
		"chunk_size":             500,
		"chunk_overlap":          50,
		"error":                  nil,
	}
	return projectID, payload, h.store.WriteProject(projectID, payload)
}

func (h *Handler) saveProjectText(projectID, text string) error {
	textPath := filepath.Join(h.store.ProjectDir(projectID), "extracted_text.txt")
	if err := os.MkdirAll(filepath.Dir(textPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(textPath, []byte(text), 0o644)
}

func (h *Handler) generateOntology(ctx context.Context, simulationRequirement, sourceText, additionalContext string) (map[string]any, error) {
	baseURL := strings.TrimRight(os.Getenv("LLM_BASE_URL"), "/")
	apiKey := strings.TrimSpace(os.Getenv("LLM_API_KEY"))
	model := strings.TrimSpace(os.Getenv("LLM_MODEL_NAME"))
	if baseURL == "" || apiKey == "" || model == "" {
		return nil, fmt.Errorf("LLM_BASE_URL, LLM_API_KEY, and LLM_MODEL_NAME are required for Go ontology generation")
	}
	exec := newExecutor(intprovider.Config{
		BaseURL:      baseURL,
		APIKey:       apiKey,
		DefaultModel: model,
		ProviderName: "openai-compatible",
	})
	builder := intontology.NewBuilder(providerAdapter{exec: exec, model: model})
	ontology, err := builder.Build(ctx, intontology.BuildInput{
		SimulationRequirement: simulationRequirement,
		SourceText:            sourceText,
		AdditionalContext:     additionalContext,
	})
	if err != nil {
		return nil, err
	}
	entityTypes := make([]map[string]any, 0, len(ontology.EntityTypes))
	for _, entity := range ontology.EntityTypes {
		entityTypes = append(entityTypes, map[string]any{
			"name":        entity.Name,
			"description": entity.Description,
			"attributes":  entity.Attributes,
			"examples":    entity.Examples,
		})
	}
	edgeTypes := make([]map[string]any, 0, len(ontology.EdgeTypes))
	for _, edge := range ontology.EdgeTypes {
		sourceTargets := make([]map[string]any, 0, len(edge.SourceTargets))
		for _, target := range edge.SourceTargets {
			sourceTargets = append(sourceTargets, map[string]any{
				"source": target.Source,
				"target": target.Target,
			})
		}
		edgeTypes = append(edgeTypes, map[string]any{
			"name":           edge.Name,
			"description":    edge.Description,
			"attributes":     edge.Attributes,
			"source_targets": sourceTargets,
		})
	}
	return map[string]any{
		"entity_types":     entityTypes,
		"edge_types":       edgeTypes,
		"analysis_summary": ontology.AnalysisSummary,
	}, nil
}

type providerAdapter struct {
	exec  intprovider.Executor
	model string
}

func (p providerAdapter) Execute(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	resp, err := p.exec.Execute(ctx, intprovider.ProviderRequest{
		Model: p.model,
		Messages: []intprovider.Message{
			{Role: intprovider.RoleSystem, Content: systemPrompt},
			{Role: intprovider.RoleUser, Content: userPrompt},
		},
		Temperature:    0,
		MaxTokens:      8192,
		ResponseFormat: &intprovider.ResponseFormat{Type: intprovider.ResponseFormatJSONObject},
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func preprocessText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
