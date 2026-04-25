package ontologyhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Documents net/http.ServeMux (Go 1.21) precedence: exact path must beat /api/ catch-all.
func TestServeMuxOntologyBeatsAPICatchAll(t *testing.T) {
	var which string
	mux := http.NewServeMux()
	mux.HandleFunc("/api/graph/ontology/generate", func(w http.ResponseWriter, r *http.Request) {
		which = "ontology"
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		which = "api_catchall"
	})
	req := httptest.NewRequest(http.MethodPost, "/api/graph/ontology/generate", nil)
	mux.ServeHTTP(httptest.NewRecorder(), req)
	if which != "ontology" {
		t.Fatalf("got %q, want ontology", which)
	}
}
