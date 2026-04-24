package memory

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestChunkAndEpisodeHelpers(t *testing.T) {
	chunks := ChunkText("abcdefghijklmnopqrstuvwxyz", 10, 2)
	if len(chunks) < 3 {
		t.Fatalf("expected multiple chunks, got %#v", chunks)
	}
	episodes := EncodeEpisodes(chunks, "graph-1")
	if len(episodes) != len(chunks) {
		t.Fatalf("expected same number of episodes")
	}
}

func TestZepWriteSideCalls(t *testing.T) {
	client := NewZepClient("https://zep.test", "key", newClient(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	}))
	if err := client.CreateGraph(context.Background(), GraphSpec{GraphID: "g"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := client.SubmitOntology(context.Background(), "g", OntologySpec{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := client.IngestEpisode(context.Background(), Episode{GraphID: "g", Data: "x", Type: "text"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := client.IngestBatch(context.Background(), "g", []Episode{{GraphID: "g", Data: "x", Type: "text"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
