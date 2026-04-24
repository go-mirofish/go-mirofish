package report

import (
	"context"
	"time"
)

type Fetcher interface {
	Fetch(context.Context, Query) (FetchResult, error)
}

type ReportWriter interface {
	Write(context.Context, Report) ([]byte, error)
}

type Query struct {
	Key      string
	Source   string
	Filter   string
	TimeFrom time.Time
	TimeTo   time.Time
}

type FetchResult struct {
	Key       string         `json:"key"`
	Source    string         `json:"source"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

type ReportSpec struct {
	Title       string   `json:"title"`
	Format      string   `json:"format"`
	Queries     []Query  `json:"queries"`
	IncludeKeys []string `json:"include_keys,omitempty"`
}

type Report struct {
	Title   string        `json:"title"`
	Format  string        `json:"format"`
	Results []FetchResult `json:"results"`
}
