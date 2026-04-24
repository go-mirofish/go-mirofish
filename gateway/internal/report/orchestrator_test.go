package report

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type stubFetcher struct {
	results map[string]FetchResult
}

func (s stubFetcher) Fetch(ctx context.Context, q Query) (FetchResult, error) {
	return s.results[q.Key], nil
}

func TestOrchestratorRun(t *testing.T) {
	now := time.Now()
	orch := NewOrchestrator(
		stubFetcher{results: map[string]FetchResult{
			"b": {Key: "b", Source: "memory", Timestamp: now.Add(time.Minute)},
			"a": {Key: "a", Source: "memory", Timestamp: now},
		}},
		JSONWriter{},
	)

	body, err := orch.Run(context.Background(), ReportSpec{
		Title:  "test",
		Format: "json",
		Queries: []Query{
			{Key: "b", Source: "memory"},
			{Key: "a", Source: "memory"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var reportData Report
	if err := json.Unmarshal(body, &reportData); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if len(reportData.Results) != 2 || reportData.Results[0].Key != "a" {
		t.Fatalf("results were not deterministically ordered: %#v", reportData.Results)
	}
}
