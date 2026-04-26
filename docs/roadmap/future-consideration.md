# Future consideration: simulation depth beyond full Go ownership

`go-mirofish` is built on a **Go-owned** public path: the gateway, native workers, and the Vue app talk to an OpenAI-compatible LLM and Zep-backed graph memory; orchestration and durable rules live in **Go** so behaviour stays testable and deployable. **This page does not describe migration, parity, or how to port from another stack**—it only sketches **long-horizon R&D** to make social simulation more realistic, defensible, and deployable from edge (e.g. Raspberry Pi) to workstation.

**Scope:** research directions, possible algorithms, and system patterns. Nothing here is a committed roadmap item until scoped as an issue or RFC.

---

## 1. Echo chamber and diversity of opinion

**Problem:** Homogeneous agent populations collapse into a “happy consensus,” reducing usefulness for foresight and stress-testing.

**Directions:**

- **Cognitive bias anchoring:** Do not start every agent from a neutral prior. Assign **fixed priors** (e.g. weights over stance or values) that encode ideology, culture, or role, implemented as *data* in Go (config + seeded RNG), not as a single “neutral” system prompt.
- **Entropy injection:** The Go engine can **inject noise** into a tunable share of interactions (e.g. surface contrarian or minority-held content) so the feed does not become self-reinforcing. Parameters: rate, platform, topic.
- **Dialectics in reporting:** The **ReportAgent** can be steered to emphasize **tension and friction** (thesis + antithesis) before synthesis, not only “areas of agreement.”

---

## 2. Judgment-action gap (saying vs. doing)

**Problem:** LLM agents “say” dramatic things without **paying a cost** that constrains real behavior.

**Directions:**

- **Token- or account-weighted stakes:** Introduce a per-agent **resource** (e.g. reputation, “attention,” or an abstract currency) that is **spent** on high-impact actions (e.g. protest, large trade, mass broadcast).
- **Prospect-theory style risk:** Penalties for **losses** to reputation or position should weigh **heavier** than gains of similar size in the action-selection layer (implemented in **Go** rules, with the LLM only proposing *candidates*).
- **Outcome:** The orchestrator can **veto** or down-rank actions that the economic/state layer marks as infeasible, improving behavioral realism over pure text.

---

## 3. Hallucination cascades (Woozle / rumor spread)

**Problem:** Fictional “facts” echo through the social layer without a ground-truth check.

**Directions:**

- **Multi-agent fact-checking (“Oracle” / audit layer):** Dedicated **AuditAgents** (or a pipeline stage) with access to **grounded** sources: GraphRAG over uploaded docs, and optionally **live web RAG** (e.g. search APIs) when enabled. They do not need to be “in character” in the same channel; they **flag** or score claims before wide propagation.
- **Knowledge decay:** Implement **memory half-life** in Go: claims not reinforced by sources decay in confidence; low-confidence content can be treated as **rumor** in downstream reasoning and reporting.
- **Implementation note:** Verification middleware should be **first-class in Go** (intercept post → score → allow/shadow/flag) with the LLM as a **narration** layer, not the sole source of truth.

---

## 4. Quantitative gaps (technical accuracy)

**Problem:** LLMs are poor at exact arithmetic, markets, and accounting.

**Directions:**

- **Hybrid symbolic-neural design:** The LLM outputs **intentions**; a **Go-native math / market engine** applies rules: balances, slippage, fees, time steps. Numbers shown to users and to agents’ state should come from the engine, not free-generated digits.
- **Tool use:** Expose **calculators, simulators, or portfolio APIs** via structured tool calls so agents interact with a **rigid** numeric environment.
- **Philosophy:** *Narrative* from the model, **physics** from the engine (see also §Environmental hardening, below).

---

## 5. Temporal scaling and memory (long runs)

**Problem:** Storing every utterance does not scale; long horizons need **compression**.

**Directions:**

- **Recursive summarization:** On a schedule, roll recent interactions into **narrative summaries** stored as structured state (or graph nodes) instead of raw logs only.
- **Tiered memory (conceptual L1 / L2 / L3):**
  - **L1:** Hot, per-agent or per-session state in memory (tight cap).
  - **L2:** Mid-term rollups (e.g. daily summaries) in durable local storage.
  - **L3:** Long-horizon “core beliefs” and major events tied to GraphRAG / project graph.
- **“Janitor” job:** A Go background task every *N* ticks: summarize, merge into L3, trim volatile buffers, keeping context windows and disk growth bounded.

---

## 6. Sovereign agents and the “Governor”

**Principle:** Treat agents as **sovereign actors**: distinct state, memory rules, and constraints, not interchangeable text emitters.

**Core ideas:**

- **Central orchestrator (“Governor”):** One module owns world **physics**: time ticks, action costs, feed rules, and audit hooks.
- **Goroutine model (future):** Long-lived work per agent or per shard can use **goroutines** and **channels** for isolation and back-pressure; the exact model must match current gateway/worker layout; this is a *design target*, not the present architecture.
- **State machine per agent:** A struct (or small store row) can track **wealth, reputation, affiliation, bias scale,** and similar fields the simulation design requires.
- **Discrete time:** A **tick** (e.g. 1 tick = 1 hour simulated) synchronizes rule application, decay, and summarization across the swarm.

---

## 7. “Truth engine” (verification before broadcast)

- **Cache of verified facts** from GraphRAG (e.g. Bloom filter, or a purpose-built in-process cache) for quick overlap checks.
- **Flow:** Agent proposes content → **interceptor** scores against source truth and audit policy → if hallucination risk is high, **down-rank, flag as rumor, or block** from global feed.
- **Implementation:** Prefer **Go** for the fast path; optional **sidecar** (e.g. Ollama, small local model) only where needed for cost/latency tradeoffs.

---

## 8. “Skin in the game” (action veto)

Illustrative shape (not current production code):

```go
type Agent struct {
    Balance    float64
    Conviction float64 // rises/falls with outcomes; gates costly actions
}
```

Before a costly action, the **Go** layer compares **Cost of action** to **Conviction** (and other resources). If insufficient, the engine **rejects** or revises the action regardless of the LLM’s raw text.

---

## 9. Suggested future technical stack (optional layers)

| Concern | Possible direction |
| --- | --- |
| High-volume fan-out / fan-in | [NATS](https://nats.io/) or Redis Streams; evaluate vs. current in-process design |
| Vector retrieval | e.g. Milvus, Qdrant, or continue Zep/cloud path, **Go clients** where possible |
| LLM calls | OpenAI-compatible APIs, Ollama, vLLM; [LangChainGo](https://github.com/tmc/langchaingo) or a thin in-house client; keep orchestration in **Go** |

**Philosophy:** **Environmental hardening:** the LLM provides flavor; the **Go** layer provides non-negotiable **rules** (e.g. “you cannot jump over the moon”: gravity in code).

---

## 10. Edge and Raspberry Pi (local “sovereign” simulation)

Supporting **Raspberry Pi** with a **Go-native** core is a strong fit for **local-first, low-footprint** runs versus heavier stacks. The upstream [MiroFish](https://github.com/666ghj/MiroFish) ecosystem often assumes **larger** RAM; a Go service can be tuned for **smaller** devices with explicit limits.

**Design themes:**

- **Thin-agent / tiered inference:** The Pi runs the **orchestrator** and game state; **heavy LLM** calls may target **Ollama on the Pi** (small models only) or a **remote** OpenAI-compatible endpoint on a stronger machine. Swarm size and model size must be **capped** on-device.
- **Storage:** Favor **embedded** or single-file options (e.g. SQLite, [Badger](https://github.com/dgraph-io/badger), [bbolt](https://github.com/etcd-io/bbolt)) or lightweight graph layers; avoid co-locating a full server-class DB on 4 to 8 GB RAM.
- **ARM build:** Cross-compile with `GOOS=linux GOARCH=arm64`; keep **static** or clearly documented dynamic deps.
- **Thermals and I/O:** Pi-class hardware benefits from **cooling**, **reliable storage** (minimize SD thrash; prefer SSD where possible), and **ZRAM** or conservative swap to reduce wear.
- **“Remote brain” vs “local brain”:** Pi 4 often acts as **controller only**; Pi 5 may run **tiny** local models; both align with the **resource-aware** scheduling idea below.

**Claim discipline (Pi / edge):** Do **not** use wording like *Pi verified*, *Raspberry Pi certified*, or similar unless you can show **evidence from a real on-device run** on the hardware and OS you are talking about—e.g. build logs, command transcripts, or screenshots of the app running the scenario on that device. *Hypothetical* or *laptop-only* performance does not count. If you are describing a target or experiment, say so clearly instead of implying production-grade validation.

---

## 11. Cross-platform and “single binary” experience

- **Build tags** (`//go:build`) for optional OS/arch paths (e.g. desktop vs. ARM edge).
- **Inference discovery:** On startup, detect GPU vs CPU, local Ollama vs API-only, and set **concurrency and model** defaults.
- **Long-term mobile:** A **UI-agnostic** Go library (`gomobile` or similar) is a *possible* path; most product code today is gateway + web UI.
- **Single-binary story:** `embed` for seeds, default assets, and small static UIs can reduce install friction (aligned with Go distribution strengths).

**Resource-aware scheduling (future):** A `SimulationManager` could throttle parallel agent steps when on **battery**, **thermal stress**, or **low-memory** targets, favoring stable runs over peak throughput on weak hardware.

---

## 12. How this document relates to the main roadmap

The **Roadmap** page in the app (also maintained as a markdown file in the repo) is where we track **near-term, concrete** work: hardening, proof, contracts, and documentation for the **stack as it ships today**, in a rough dependency order. **This page is different:** it is a **long-horizon, non-binding** sketch of research and architecture—echo chambers, economic realism, factuality, long-horizon memory, and edge—ideas we may pursue **after** the current path is more operationally mature. It does not replace the roadmap, and it is not a commitment until something is written up with scope, metrics, and risk as an issue or RFC.

**Next step for any item above:** break out into a dedicated RFC or issue with **scope, metrics, and risk** (especially for any change to the live simulation or report contracts).

### References and further reading (non-normative)

- [Upstream MiroFish (comparison / lineage)](https://github.com/666ghj/MiroFish)
- Prospect theory (behavioral risk): Kahneman and Tversky (summaries widely available)
- [Raspberry Pi + Go (community discussion, not project endorsement)](https://www.reddit.com/r/golang/comments/9cz78l/is_go_suited_for_running_on_iot_devices_or_are/)
- [IoT / camera on Pi (example thread)](https://www.reddit.com/r/golang/comments/1cb5vst/any_uptodate_go_frameworklibraries_for_raspberry/)
- [Cayley (graph, optional research)](https://github.com/cayleygraph/cayley)
- Peripherals / thermal awareness (example doc): e.g. vendor documentation for Pi cooling and 64-bit OS requirements; validate against your hardware.

---

*Last updated: long-horizon R&D themes.*
