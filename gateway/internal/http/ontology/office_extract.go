package ontologyhttp

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var xmlTagRE = regexp.MustCompile(`<[^>]+>`)
var spaceCollapse = regexp.MustCompile(`\s+`)
var slideNumRE = regexp.MustCompile(`slide(\d+)\.xml`)

// extractDOCXText pulls text from word/document.xml (Office Open XML).
func extractDOCXText(data []byte) (string, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open docx zip: %w", err)
	}
	var doc io.ReadCloser
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			doc, err = f.Open()
			if err != nil {
				return "", err
			}
			break
		}
	}
	if doc == nil {
		return "", fmt.Errorf("word/document.xml not found in docx")
	}
	defer doc.Close()
	b, err := io.ReadAll(doc)
	if err != nil {
		return "", err
	}
	return textFromXMLish(string(b)), nil
}

// extractPPTXText concatenates text from ppt/slides/slideN.xml in order.
func extractPPTXText(data []byte) (string, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open pptx zip: %w", err)
	}
	prefix, suffix := "ppt/slides/slide", ".xml"
	var names []string
	for _, f := range r.File {
		n := f.Name
		if strings.HasPrefix(n, prefix) && strings.HasSuffix(n, suffix) {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no ppt/slides/slide*.xml in pptx")
	}
	sort.Slice(names, func(i, j int) bool { return slideNum(names[i]) < slideNum(names[j]) })
	var parts []string
	for _, name := range names {
		for _, f := range r.File {
			if f.Name != name {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			b, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				return "", err
			}
			t := textFromXMLish(string(b))
			if t != "" {
				parts = append(parts, t)
			}
			break
		}
	}
	out := strings.Join(parts, "\n\n")
	if strings.TrimSpace(out) == "" {
		return "", fmt.Errorf("no extractable text in pptx (slides may be image-only)")
	}
	return out, nil
}

func slideNum(name string) int {
	m := slideNumRE.FindStringSubmatch(name)
	if len(m) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

func textFromXMLish(s string) string {
	s = xmlTagRE.ReplaceAllString(s, " ")
	s = spaceCollapse.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
