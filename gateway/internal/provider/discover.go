package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DiscoveredEndpoint describes a reachable local inference server.
type DiscoveredEndpoint struct {
	// BaseURL is the OpenAI-compatible /v1 root (e.g. "http://host.docker.internal:11434/v1").
	BaseURL string
	Kind    ProviderKind
	Models  []DiscoveredModel
}

// DiscoveredModel is a model available on a discovered endpoint.
type DiscoveredModel struct {
	ID   string
	Tier ModelTier
}

// defaultProbes lists (url, kind) pairs checked by Discover in priority order.
// host.docker.internal variants are tried first so Docker setups work without config.
var defaultProbes = []struct {
	url  string
	kind ProviderKind
}{
	// Ollama
	{"http://host.docker.internal:11434", KindOllama},
	{"http://127.0.0.1:11434", KindOllama},
	// LM Studio
	{"http://host.docker.internal:1234", KindLMStudio},
	{"http://127.0.0.1:1234", KindLMStudio},
	// llama.cpp server (default ports: 8080 and 8000)
	{"http://host.docker.internal:8080", KindLlamaCpp},
	{"http://127.0.0.1:8080", KindLlamaCpp},
	{"http://host.docker.internal:8000", KindLlamaCpp},
	{"http://127.0.0.1:8000", KindLlamaCpp},
}

// Discover probes well-known local inference server endpoints in parallel and
// returns every reachable one. Results are ordered: higher-tier models first.
//
// extraBaseURLs adds caller-supplied URLs to the probe list (e.g. from
// LLM_DISCOVER_EXTRA_URLS); duplicates are skipped automatically.
func Discover(ctx context.Context, client *http.Client, extraBaseURLs []string) []DiscoveredEndpoint {
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}

	type probe struct {
		url  string
		kind ProviderKind
	}
	probes := make([]probe, 0, len(defaultProbes)+len(extraBaseURLs))
	for _, p := range defaultProbes {
		probes = append(probes, probe{p.url, p.kind})
	}
	for _, u := range extraBaseURLs {
		u = strings.TrimRight(strings.TrimSpace(u), "/")
		if u != "" {
			probes = append(probes, probe{u, KindOpenAICompat})
		}
	}

	// Deduplicate
	seen := map[string]bool{}
	deduped := probes[:0]
	for _, p := range probes {
		key := strings.TrimRight(p.url, "/")
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, p)
	}
	probes = deduped

	type result struct {
		ep  DiscoveredEndpoint
		ok  bool
		idx int
	}

	results := make([]result, len(probes))
	var wg sync.WaitGroup
	for i, p := range probes {
		wg.Add(1)
		go func(idx int, base string, kind ProviderKind) {
			defer wg.Done()
			base = strings.TrimRight(base, "/")
			ep, ok := probeOpenAICompat(ctx, client, base, kind)
			if !ok && kind == KindOllama {
				ep, ok = probeOllamaNative(ctx, client, base)
			}
			results[idx] = result{ep: ep, ok: ok, idx: idx}
		}(i, p.url, p.kind)
	}
	wg.Wait()

	// Collect in original order (preserves priority)
	var out []DiscoveredEndpoint
	for _, r := range results {
		if r.ok {
			out = append(out, r.ep)
		}
	}
	return out
}

// probeOpenAICompat calls GET /v1/models. Returns true when the endpoint
// answers with a non-empty model list.
func probeOpenAICompat(ctx context.Context, client *http.Client, base string, kind ProviderKind) (DiscoveredEndpoint, bool) {
	url := base + "/v1/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return DiscoveredEndpoint{}, false
	}
	resp, err := client.Do(req)
	if err != nil {
		return DiscoveredEndpoint{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return DiscoveredEndpoint{}, false
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil || len(payload.Data) == 0 {
		return DiscoveredEndpoint{}, false
	}
	models := make([]DiscoveredModel, 0, len(payload.Data))
	for _, m := range payload.Data {
		models = append(models, DiscoveredModel{ID: m.ID, Tier: ClassifyModelTier(m.ID)})
	}
	return DiscoveredEndpoint{BaseURL: base + "/v1", Kind: kind, Models: models}, true
}

// probeOllamaNative calls GET /api/tags (Ollama-native endpoint available even
// when /v1 is disabled). Falls back to /v1 base URL for actual completions since
// Ollama has exposed /v1/chat/completions since v0.1.24.
func probeOllamaNative(ctx context.Context, client *http.Client, base string) (DiscoveredEndpoint, bool) {
	url := base + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return DiscoveredEndpoint{}, false
	}
	resp, err := client.Do(req)
	if err != nil {
		return DiscoveredEndpoint{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return DiscoveredEndpoint{}, false
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil || len(payload.Models) == 0 {
		return DiscoveredEndpoint{}, false
	}
	models := make([]DiscoveredModel, 0, len(payload.Models))
	for _, m := range payload.Models {
		models = append(models, DiscoveredModel{ID: m.Name, Tier: ClassifyModelTier(m.Name)})
	}
	return DiscoveredEndpoint{BaseURL: base + "/v1", Kind: KindOllama, Models: models}, true
}

// ClassifyModelTier infers a ModelTier from a model name string.
// It matches common size suffixes (7b, 13b, 70b, …) and recognises cloud
// model names (gpt, gemini, claude, grok).
func ClassifyModelTier(name string) ModelTier {
	lower := strings.ToLower(name)
	if containsAny(lower, "gpt-", "gemini", "claude", "grok", "mistral-large", "command-r") {
		return TierCloud
	}
	for _, s := range []string{"671b", "405b", "70b", "72b", "65b", "47b", "34b"} {
		if strings.Contains(lower, s) {
			return TierLarge
		}
	}
	for _, s := range []string{"22b", "20b", "14b", "13b", "12b", "11b", "10b"} {
		if strings.Contains(lower, s) {
			return TierMedium
		}
	}
	for _, s := range []string{"9b", "8b", "7b", "6b"} {
		if strings.Contains(lower, s) {
			return TierSmall
		}
	}
	for _, s := range []string{"3b", "2b", "1b", "0.5b", "500m", "mini"} {
		if strings.Contains(lower, s) {
			return TierTiny
		}
	}
	return TierSmall
}

// TierName returns a human-readable label for a ModelTier.
func TierName(t ModelTier) string {
	switch t {
	case TierTiny:
		return "tiny"
	case TierSmall:
		return "small"
	case TierMedium:
		return "medium"
	case TierLarge:
		return "large"
	case TierCloud:
		return "cloud"
	default:
		return "unknown"
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
