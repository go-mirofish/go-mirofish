package provider

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

// RegistryConfig controls which providers are in the pool.
type RegistryConfig struct {
	// Primary LLM (maps to LLM_BASE_URL / LLM_API_KEY / LLM_MODEL_NAME).
	PrimaryURL   string
	PrimaryKey   string
	PrimaryModel string

	// Boost LLM (maps to LLM_BOOST_BASE_URL / LLM_BOOST_API_KEY / LLM_BOOST_MODEL_NAME).
	// Used as the highest-priority provider when set.
	BoostURL   string
	BoostKey   string
	BoostModel string

	// AutoDiscover probes well-known local inference server ports (Ollama, LM Studio,
	// llama.cpp) and adds any that respond to the pool.  Defaults to true.
	AutoDiscover bool

	// ExtraDiscoverURLs are additional base URLs probed during auto-discovery.
	// Parsed from LLM_DISCOVER_EXTRA_URLS (comma-separated).
	ExtraDiscoverURLs []string

	// Timeout is applied to inference calls for all providers in the pool.
	Timeout time.Duration
}

// rankedProvider wraps an Executor with routing metadata.
type rankedProvider struct {
	exec     Executor
	name     string
	kind     ProviderKind
	tier     ModelTier
	model    string
	baseURL  string
	priority int // lower = higher priority
}

// Registry manages a prioritised pool of LLM providers and exposes a single
// Executor that routes/falls-back across them.
type Registry struct {
	pool []rankedProvider
}

// NewRegistry builds the provider pool from cfg.  Auto-discovery runs
// synchronously with a short per-probe timeout so startup is not blocked.
func NewRegistry(cfg RegistryConfig) *Registry {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 300 * time.Second
	}

	var pool []rankedProvider

	// ── Priority 0: boost provider (highest quality / cloud) ─────────────────
	if cfg.BoostURL != "" && cfg.BoostKey != "" && cfg.BoostModel != "" {
		exec := NewExecutor(Config{
			BaseURL:      cfg.BoostURL,
			APIKey:       cfg.BoostKey,
			DefaultModel: cfg.BoostModel,
			ProviderName: "boost",
			Timeout:      timeout,
		}, nil)
		pool = append(pool, rankedProvider{
			exec:     exec,
			name:     "boost",
			kind:     guessKind(cfg.BoostURL),
			tier:     ClassifyModelTier(cfg.BoostModel),
			model:    cfg.BoostModel,
			baseURL:  cfg.BoostURL,
			priority: 0,
		})
	}

	// ── Priority 1: explicitly configured primary provider ────────────────────
	if cfg.PrimaryURL != "" && cfg.PrimaryModel != "" {
		key := cfg.PrimaryKey
		if key == "" {
			key = "local"
		}
		exec := NewExecutor(Config{
			BaseURL:      cfg.PrimaryURL,
			APIKey:       key,
			DefaultModel: cfg.PrimaryModel,
			ProviderName: "primary",
			Timeout:      timeout,
		}, nil)
		pool = append(pool, rankedProvider{
			exec:     exec,
			name:     "primary",
			kind:     guessKind(cfg.PrimaryURL),
			tier:     ClassifyModelTier(cfg.PrimaryModel),
			model:    cfg.PrimaryModel,
			baseURL:  cfg.PrimaryURL,
			priority: 1,
		})
	}

	// ── Priority 2+: auto-discovered local servers ────────────────────────────
	if cfg.AutoDiscover {
		probeCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		probeClient := &http.Client{Timeout: 3 * time.Second}

		discovered := Discover(probeCtx, probeClient, cfg.ExtraDiscoverURLs)
		for i, ep := range discovered {
			// Skip if already covered by the configured primary to avoid duplicates.
			if cfg.PrimaryURL != "" && sameBase(ep.BaseURL, cfg.PrimaryURL) {
				continue
			}
			if cfg.BoostURL != "" && sameBase(ep.BaseURL, cfg.BoostURL) {
				continue
			}
			best := pickBestModel(ep.Models)
			if best == nil {
				continue
			}
			exec := NewExecutor(Config{
				BaseURL:      ep.BaseURL,
				APIKey:       "local",
				DefaultModel: best.ID,
				ProviderName: string(ep.Kind),
				Timeout:      timeout,
			}, nil)
			name := string(ep.Kind) + "@" + ep.BaseURL
			pool = append(pool, rankedProvider{
				exec:     exec,
				name:     name,
				kind:     ep.Kind,
				tier:     best.Tier,
				model:    best.ID,
				baseURL:  ep.BaseURL,
				priority: 10 + i,
			})
			log.Printf("[provider] discovered %s model=%s tier=%s url=%s",
				ep.Kind, best.ID, TierName(best.Tier), ep.BaseURL)
		}
	}

	// Stable-sort by priority so boost < primary < discovered.
	sort.SliceStable(pool, func(i, j int) bool {
		return pool[i].priority < pool[j].priority
	})

	switch len(pool) {
	case 0:
		log.Printf("[provider] warning: no LLM providers configured or discovered; LLM features will be unavailable")
	default:
		log.Printf("[provider] pool: %d provider(s), leading with %q (model=%s tier=%s)",
			len(pool), pool[0].name, pool[0].model, TierName(pool[0].tier))
	}

	return &Registry{pool: pool}
}

// Executor returns a single Executor that routes across all pool providers
// with transparent fallback.  Returns nil when the pool is empty.
func (r *Registry) Executor() Executor {
	if len(r.pool) == 0 {
		return nil
	}
	return &RegistryExecutor{pool: r.pool}
}

// PoolInfo returns a snapshot of the current provider pool for introspection.
func (r *Registry) PoolInfo() []ProviderInfo {
	out := make([]ProviderInfo, len(r.pool))
	for i, p := range r.pool {
		out[i] = ProviderInfo{
			Name:    p.name,
			Kind:    string(p.kind),
			Model:   p.model,
			Tier:    TierName(p.tier),
			BaseURL: p.baseURL,
		}
	}
	return out
}

// ProviderInfo is a serialisable summary of one pool entry.
type ProviderInfo struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Model   string `json:"model"`
	Tier    string `json:"tier"`
	BaseURL string `json:"base_url"`
}

// ─── RegistryExecutor ─────────────────────────────────────────────────────────

// RegistryExecutor routes requests across the pool in priority order.
// It falls back to the next provider only on unavailability errors; it does NOT
// fall back on timeouts (the model connected but is generating slowly) or client
// errors (bad request format, logic issue).
type RegistryExecutor struct {
	pool []rankedProvider
}

func (re *RegistryExecutor) Execute(ctx context.Context, req ProviderRequest) (ProviderResponse, error) {
	var lastErr error
	for _, p := range re.pool {
		resp, err := p.exec.Execute(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !isAvailabilityError(err) {
			// Connection-level problem — don't try next provider.
			break
		}
		log.Printf("[provider] fallback from %q (%v) → trying next", p.name, err)
	}
	if lastErr == nil {
		lastErr = &Error{
			Op:      "Execute",
			Kind:    ErrUnavailable,
			Provider: "registry",
			Message: "no providers available",
		}
	}
	return ProviderResponse{}, lastErr
}

// isAvailabilityError returns true for errors that indicate the provider is
// unreachable so we can try the next one in the pool.
func isAvailabilityError(err error) bool {
	var provErr *Error
	if !errors.As(err, &provErr) {
		return true // unknown — try next to be safe
	}
	return provErr.Kind == ErrUnavailable || provErr.Kind == ErrCircuitOpen
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func guessKind(baseURL string) ProviderKind {
	lower := strings.ToLower(baseURL)
	if strings.Contains(lower, "11434") || strings.Contains(lower, "ollama") {
		return KindOllama
	}
	if strings.Contains(lower, "1234") {
		return KindLMStudio
	}
	if strings.Contains(lower, "8080") || strings.Contains(lower, "8000") {
		return KindLlamaCpp
	}
	if containsAny(lower, "openai.com", "generativelanguage", "x.ai", "anthropic", "groq.com", "dashscope") {
		return KindCloud
	}
	return KindOpenAICompat
}

// sameBase returns true when two base URLs point to the same server
// (strips trailing slashes and the /v1 suffix before comparing).
func sameBase(a, b string) bool {
	norm := func(s string) string {
		s = strings.TrimRight(strings.TrimSpace(s), "/")
		s = strings.TrimSuffix(s, "/v1")
		s = strings.TrimRight(s, "/")
		return strings.ToLower(s)
	}
	return norm(a) == norm(b)
}

// pickBestModel returns the model with the highest tier in the list, or nil.
func pickBestModel(models []DiscoveredModel) *DiscoveredModel {
	if len(models) == 0 {
		return nil
	}
	best := &models[0]
	for i := 1; i < len(models); i++ {
		if models[i].Tier > best.Tier {
			best = &models[i]
		}
	}
	return best
}
