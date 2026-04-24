package report

import (
	"context"
	"sort"

	"golang.org/x/sync/errgroup"
)

type Orchestrator struct {
	fetcher Fetcher
	writer  ReportWriter
}

func NewOrchestrator(fetcher Fetcher, writer ReportWriter) *Orchestrator {
	return &Orchestrator{fetcher: fetcher, writer: writer}
}

func (o *Orchestrator) Run(ctx context.Context, spec ReportSpec) ([]byte, error) {
	results := make([]FetchResult, len(spec.Queries))
	group, ctx := errgroup.WithContext(ctx)

	for idx, query := range spec.Queries {
		idx, query := idx, query
		group.Go(func() error {
			result, err := o.fetcher.Fetch(ctx, query)
			if err != nil {
				return err
			}
			results[idx] = result
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Key == results[j].Key {
			return results[i].Timestamp.Before(results[j].Timestamp)
		}
		return results[i].Key < results[j].Key
	})

	return o.writer.Write(ctx, Report{
		Title:   spec.Title,
		Format:  spec.Format,
		Results: results,
	})
}
