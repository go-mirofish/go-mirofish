package memory

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func newClient(fn roundTripFunc) *http.Client { return &http.Client{Transport: fn} }

func TestZepClient(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		method    string
		status    int
		body      string
		run       func(*testing.T, *ZepClient)
		expectErr error
	}{
		{
			name:   "search graph",
			method: http.MethodPost,
			status: http.StatusOK,
			body:   `{"facts":["a"],"nodes":[],"edges":[]}`,
			run: func(t *testing.T, client *ZepClient) {
				resp, err := client.SearchGraph(context.Background(), SearchRequest{Query: "q", GraphID: "g", Limit: 5})
				if err != nil || len(resp.Facts) != 1 {
					t.Fatalf("unexpected response: %#v %v", resp, err)
				}
			},
		},
		{
			name:      "unauthorized",
			method:    http.MethodGet,
			status:    http.StatusUnauthorized,
			body:      `{}`,
			expectErr: ErrMemoryUnauthorized,
			run: func(t *testing.T, client *ZepClient) {
				err := client.DeleteNode(context.Background(), "n")
				if err == nil {
					t.Fatalf("expected error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewZepClient("https://zep.test", "key", newClient(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: tt.status,
					Header:     http.Header{},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))
			tt.run(t, client)
		})
	}
}
