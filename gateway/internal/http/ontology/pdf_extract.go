package ontologyhttp

import (
	"bytes"
	"fmt"
	"strings"

	"rsc.io/pdf"
)

// ExtractPDFForOntology extracts text from a PDF; replace in tests to avoid loading real files.
var ExtractPDFForOntology = extractPDFTextDefault

func extractPDFTextDefault(data []byte) (string, error) {
	ra := bytes.NewReader(data)
	r, err := pdf.NewReader(ra, int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open pdf: %w", err)
	}
	var b strings.Builder
	n := r.NumPage()
	for i := 1; i <= n; i++ {
		p := r.Page(i)
		content := p.Content()
		for _, t := range content.Text {
			b.WriteString(t.S)
		}
		if i < n {
			b.WriteString("\n\n")
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		return "", fmt.Errorf("no extractable text in PDF (image-only slides, scan, or unsupported encoding); try a .txt/.md export, a text-based PDF, or .docx/.pptx")
	}
	return out, nil
}
