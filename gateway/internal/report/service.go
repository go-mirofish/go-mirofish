package report

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-mirofish/go-mirofish/gateway/internal/memory"
	"github.com/go-mirofish/go-mirofish/gateway/internal/provider"
	reportstore "github.com/go-mirofish/go-mirofish/gateway/internal/store/report"
)

type SimulationLookup interface {
	ReadSimulation(simulationID string) (map[string]any, error)
	ReadProject(projectID string) (map[string]any, error)
}

type GeneratedSection struct {
	Index   int
	Title   string
	Content string
}

type Service struct {
	store   reportstore.Store
	memory  memory.Client
	planner *Planner
}

func NewService(store reportstore.Store, memoryClient memory.Client, planner *Planner) *Service {
	return &Service{store: store, memory: memoryClient, planner: planner}
}

type ReportGenerateRequest struct {
	SimulationID    string `json:"simulation_id"`
	ForceRegenerate bool   `json:"force_regenerate,omitempty"`
}

type ReportGenerateResponse struct {
	SimulationID     string `json:"simulation_id"`
	ReportID         string `json:"report_id"`
	TaskID           string `json:"task_id"`
	Status           string `json:"status"`
	Message          string `json:"message"`
	AlreadyGenerated bool   `json:"already_generated"`
}

type ReportStatusResponse struct {
	ReportID         string `json:"report_id,omitempty"`
	SimulationID     string `json:"simulation_id,omitempty"`
	Status           string `json:"status"`
	Progress         int    `json:"progress"`
	Message          string `json:"message"`
	AlreadyCompleted bool   `json:"already_completed,omitempty"`
}

type sectionGenerator struct {
	executor provider.Executor
	model    string
}

func (s sectionGenerator) GenerateSection(ctx context.Context, plan SectionPlan, facts []string) (string, error) {
	if s.executor == nil {
		return "Evidence summary:\n\n- " + strings.Join(facts, "\n- "), nil
	}
	req := provider.ProviderRequest{
		Model: s.model,
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: "Write a concise report section from the provided simulation evidence."},
			{Role: provider.RoleUser, Content: "Section: " + plan.Title + "\nFacts:\n" + strings.Join(facts, "\n")},
		},
		Temperature: 0,
		MaxTokens:   4096,
	}
	resp, err := s.executor.Execute(ctx, req)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(resp.Content), nil
}

func (s *Service) Generate(ctx context.Context, req ReportGenerateRequest, lookup SimulationLookup, model string, exec provider.Executor) (ReportGenerateResponse, error) {
	if strings.TrimSpace(req.SimulationID) == "" {
		return ReportGenerateResponse{}, fmt.Errorf("report.Generate: %w", ErrInvalidReportRequest)
	}
	if !req.ForceRegenerate {
		if existing, found, err := s.store.FindBySimulation(req.SimulationID); err == nil && found && existing.Status == "completed" {
			return ReportGenerateResponse{
				SimulationID:     req.SimulationID,
				ReportID:         existing.ReportID,
				Status:           "completed",
				Message:          "Report already exists",
				AlreadyGenerated: true,
			}, nil
		}
	}

	sim, err := lookup.ReadSimulation(req.SimulationID)
	if err != nil {
		return ReportGenerateResponse{}, err
	}
	projectID, _ := sim["project_id"].(string)
	graphID, _ := sim["graph_id"].(string)
	project, err := lookup.ReadProject(projectID)
	if err != nil {
		return ReportGenerateResponse{}, err
	}
	requirement, _ := project["simulation_requirement"].(string)
	reportID := fmt.Sprintf("report_%d", time.Now().UnixNano())
	taskID := fmt.Sprintf("%d", time.Now().UnixNano())
	meta := reportstore.ReportMeta{
		ReportID:              reportID,
		SimulationID:          req.SimulationID,
		GraphID:               graphID,
		SimulationRequirement: requirement,
		Status:                "generating",
		CreatedAt:             time.Now().Format(time.RFC3339),
	}
	if err := s.store.CreateReport(reportID, meta); err != nil {
		return ReportGenerateResponse{}, err
	}
	if err := s.store.SaveProgress(reportID, NewProgress("generating", 0, "Starting report generation")); err != nil {
		return ReportGenerateResponse{}, err
	}

	go s.run(reportID, req.SimulationID, graphID, requirement, model, exec)

	return ReportGenerateResponse{
		SimulationID: req.SimulationID,
		ReportID:     reportID,
		TaskID:       taskID,
		Status:       "generating",
		Message:      "Report generation started",
	}, nil
}

func (s *Service) run(reportID, simulationID, graphID, requirement, model string, exec provider.Executor) {
	ctx := context.Background()
	factsResp, err := s.memory.SearchGraph(ctx, memory.SearchRequest{
		Query:   requirement,
		GraphID: graphID,
		Limit:   20,
		Scope:   "edges",
	})
	if err != nil {
		s.fail(reportID, err)
		return
	}
	if err := s.store.SaveProgress(reportID, NewProgress("planning", 20, "Planning report outline")); err != nil {
		s.fail(reportID, err)
		return
	}
	outline, err := s.planner.Plan(ctx, requirement, factsResp.Facts)
	if err != nil {
		s.fail(reportID, err)
		return
	}
	assembler := NewAssembler(sectionGenerator{executor: exec, model: model})
	if err := s.store.SaveProgress(reportID, NewProgress("generating", 50, "Generating report sections")); err != nil {
		s.fail(reportID, err)
		return
	}
	sections, markdown, err := assembler.Assemble(ctx, outline, factsResp.Facts)
	if err != nil {
		s.fail(reportID, err)
		return
	}
	for _, section := range sections {
		if err := s.store.SaveSection(reportID, section.Index, section.Title, section.Content); err != nil {
			s.fail(reportID, err)
			return
		}
	}
	if err := s.store.SaveMarkdown(reportID, markdown); err != nil {
		s.fail(reportID, err)
		return
	}
	meta, err := s.store.LoadMeta(reportID)
	if err != nil {
		s.fail(reportID, err)
		return
	}
	meta.Status = "completed"
	meta.CompletedAt = time.Now().Format(time.RFC3339)
	meta.MarkdownContent = markdown
	meta.Outline = map[string]any{
		"title":    outline.Title,
		"summary":  outline.Summary,
		"sections": outline.Sections,
	}
	if err := s.store.SaveMeta(reportID, meta); err != nil {
		s.fail(reportID, err)
		return
	}
	_ = s.store.SaveProgress(reportID, NewProgress("completed", 100, "Report generated"))
}

func (s *Service) StatusByReportID(reportID string) (ReportStatusResponse, error) {
	if reportID == "" {
		return ReportStatusResponse{}, fmt.Errorf("report.StatusByReportID: %w", ErrInvalidReportRequest)
	}
	progress, err := s.store.LoadProgress(reportID)
	if err == nil {
		return ReportStatusResponse{
			ReportID:     reportID,
			SimulationID: progress.SimulationID,
			Status:       progress.Status,
			Progress:     progress.Progress,
			Message:      progress.Message,
		}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return ReportStatusResponse{}, err
	}
	meta, err := s.store.LoadMeta(reportID)
	if err != nil {
		return ReportStatusResponse{}, err
	}
	return ReportStatusResponse{
		ReportID:         reportID,
		SimulationID:     meta.SimulationID,
		Status:           meta.Status,
		Progress:         ternaryInt(meta.Status == "completed", 100, 0),
		Message:          ternaryString(meta.Status == "completed", "Report generated", ""),
		AlreadyCompleted: meta.Status == "completed",
	}, nil
}

func (s *Service) StatusBySimulationID(simulationID string) (ReportStatusResponse, error) {
	meta, found, err := s.store.FindBySimulation(simulationID)
	if err != nil {
		return ReportStatusResponse{}, err
	}
	if !found {
		return ReportStatusResponse{}, reportstore.ErrProgressNotFound
	}
	return s.StatusByReportID(meta.ReportID)
}

func (s *Service) Get(reportID string) (map[string]any, error) {
	meta, err := s.store.LoadMeta(reportID)
	if err != nil {
		return nil, err
	}
	return reportMetaToMap(meta), nil
}

func (s *Service) GetBySimulation(simulationID string) (map[string]any, bool, error) {
	meta, found, err := s.store.FindBySimulation(simulationID)
	if err != nil || !found {
		return nil, found, err
	}
	return reportMetaToMap(meta), true, nil
}

func (s *Service) List(simulationID string, limit int) ([]map[string]any, error) {
	reports, err := s.store.ListReports(simulationID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(reports))
	for _, report := range reports {
		out = append(out, reportMetaToMap(report))
	}
	return out, nil
}

func (s *Service) Progress(reportID string) (map[string]any, error) {
	progress, err := s.store.LoadProgress(reportID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"report_id":     reportID,
		"simulation_id": progress.SimulationID,
		"status":        progress.Status,
		"progress":      progress.Progress,
		"message":       progress.Message,
		"updated_at":    progress.UpdatedAt,
	}, nil
}

func (s *Service) Sections(reportID string) (map[string]any, error) {
	sections, err := s.store.LoadSections(reportID)
	if err != nil {
		return nil, err
	}
	meta, err := s.store.LoadMeta(reportID)
	isComplete := err == nil && meta.Status == "completed"
	return map[string]any{"report_id": reportID, "sections": sections, "total_sections": len(sections), "is_complete": isComplete}, nil
}

func (s *Service) Section(reportID string, sectionIndex int) (map[string]any, error) {
	sections, err := s.store.LoadSections(reportID)
	if err != nil {
		return nil, err
	}
	for _, section := range sections {
		if idx, _ := section["section_index"].(int); idx == sectionIndex {
			return section, nil
		}
	}
	return nil, os.ErrNotExist
}

func (s *Service) Download(reportID string) ([]byte, error) {
	markdown, err := s.store.LoadMarkdown(reportID)
	if err != nil {
		return nil, err
	}
	return []byte(markdown), nil
}

func (s *Service) Delete(reportID string) error {
	return s.store.DeleteReport(reportID)
}

func (s *Service) AgentLog(reportID string, fromLine int) (map[string]any, error) {
	return s.store.GetAgentLog(reportID, fromLine)
}

func (s *Service) AgentLogStream(reportID string) ([]map[string]any, error) {
	return s.store.GetAgentLogStream(reportID)
}

func (s *Service) ConsoleLog(reportID string, fromLine int) (map[string]any, error) {
	return s.store.GetConsoleLog(reportID, fromLine)
}

func (s *Service) ConsoleLogStream(reportID string) ([]string, error) {
	return s.store.GetConsoleLogStream(reportID)
}

func (s *Service) SearchGraph(ctx context.Context, graphID, query string, limit int) (memory.SearchResponse, error) {
	return s.memory.SearchGraph(ctx, memory.SearchRequest{GraphID: graphID, Query: query, Limit: limit, Scope: "edges"})
}

func (s *Service) GraphStatistics(ctx context.Context, graphID string) (map[string]any, error) {
	facts, err := s.memory.GetFacts(ctx, graphID, 200)
	if err != nil {
		return nil, err
	}
	graphData, err := s.memory.GetGraphData(ctx, graphID)
	if err != nil {
		graphData = memory.GraphData{GraphID: graphID}
	}
	nodeCount := graphData.NodeCount
	if nodeCount == 0 {
		nodeCount = len(graphData.Nodes)
	}
	edgeCount := graphData.EdgeCount
	if edgeCount == 0 {
		edgeCount = len(graphData.Edges)
	}
	return map[string]any{
		"graph_id":    graphID,
		"facts_count": len(facts),
		"node_count":  nodeCount,
		"edge_count":  edgeCount,
		"total_nodes": nodeCount,
		"total_edges": edgeCount,
	}, nil
}

func (s *Service) Chat(ctx context.Context, simulationID, message string, history []map[string]any, lookup SimulationLookup, model string, exec provider.Executor) (map[string]any, error) {
	report, found, err := s.GetBySimulation(simulationID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, os.ErrNotExist
	}
	reportID := report["report_id"].(string)
	markdown, err := s.store.LoadMarkdown(reportID)
	if err != nil {
		return nil, err
	}
	if exec == nil {
		return map[string]any{"response": markdown, "answer": markdown, "tool_calls": []any{}, "sources": []any{}}, nil
	}
	resp, err := exec.Execute(ctx, provider.ProviderRequest{
		Model: model,
		Messages: []provider.Message{
			{Role: provider.RoleSystem, Content: "Answer questions using the supplied report context."},
			{Role: provider.RoleUser, Content: "Report:\n" + markdown + "\n\nUser message:\n" + message},
		},
		Temperature: 0,
		MaxTokens:   2048,
	})
	if err != nil {
		return nil, err
	}
	_ = lookup
	_ = history
	answer := strings.TrimSpace(resp.Content)
	return map[string]any{"response": answer, "answer": answer, "tool_calls": []any{}, "sources": []any{}}, nil
}

func (s *Service) fail(reportID string, err error) {
	meta, loadErr := s.store.LoadMeta(reportID)
	if loadErr == nil {
		meta.Status = "failed"
		meta.Error = err.Error()
		_ = s.store.SaveMeta(reportID, meta)
	}
	_ = s.store.SaveProgress(reportID, NewProgress("failed", 0, err.Error()))
}

func ternaryInt(ok bool, yes, no int) int {
	if ok {
		return yes
	}
	return no
}

func ternaryString(ok bool, yes, no string) string {
	if ok {
		return yes
	}
	return no
}

func reportMetaToMap(meta reportstore.ReportMeta) map[string]any {
	return map[string]any{
		"report_id":              meta.ReportID,
		"simulation_id":          meta.SimulationID,
		"graph_id":               meta.GraphID,
		"simulation_requirement": meta.SimulationRequirement,
		"status":                 meta.Status,
		"outline":                meta.Outline,
		"markdown_content":       meta.MarkdownContent,
		"created_at":             meta.CreatedAt,
		"completed_at":           meta.CompletedAt,
		"error":                  meta.Error,
	}
}
