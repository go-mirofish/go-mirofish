package report

import (
	"context"
	"sort"
	"strings"

	"golang.org/x/sync/errgroup"
)

type SectionGenerator interface {
	GenerateSection(context.Context, SectionPlan, []string) (string, error)
}

type Assembler struct {
	generator SectionGenerator
}

func NewAssembler(generator SectionGenerator) *Assembler {
	return &Assembler{generator: generator}
}

func (a *Assembler) Assemble(ctx context.Context, outline Outline, facts []string) ([]GeneratedSection, string, error) {
	type result struct {
		Index   int
		Title   string
		Content string
	}
	results := make([]result, len(outline.Sections))
	group, ctx := errgroup.WithContext(ctx)
	for idx, section := range outline.Sections {
		idx, section := idx, section
		group.Go(func() error {
			content, err := a.generator.GenerateSection(ctx, section, facts)
			if err != nil {
				return err
			}
			results[idx] = result{Index: idx + 1, Title: section.Title, Content: content}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, "", err
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Index < results[j].Index })
	sections := make([]GeneratedSection, 0, len(results))
	var markdown strings.Builder
	markdown.WriteString("# ")
	markdown.WriteString(outline.Title)
	markdown.WriteString("\n\n> ")
	markdown.WriteString(outline.Summary)
	markdown.WriteString("\n\n")
	for _, item := range results {
		sections = append(sections, GeneratedSection{
			Index:   item.Index,
			Title:   item.Title,
			Content: item.Content,
		})
		markdown.WriteString("## ")
		markdown.WriteString(item.Title)
		markdown.WriteString("\n\n")
		markdown.WriteString(strings.TrimSpace(item.Content))
		markdown.WriteString("\n\n")
	}
	return sections, markdown.String(), nil
}
