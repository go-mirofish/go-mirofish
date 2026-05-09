# ADR 0001: Full-State Development Sequencing

- Status: Accepted
- Date: 2026-05-09
- Decision type: Target-state architecture and sprint sequencing
- Owners: go-mirofish maintainers

## Context

`go-mirofish` now has a Go-native public path, a live benchmark/proof surface, a Headless SDK, and a first plugin/runtime system.

The project now has three simultaneous ambitions:

1. become a **production-hard Go-native simulator**
2. become a **next-gen sovereign simulation engine**
3. remain viable for **local and edge deployment**, including constrained ARM64 profiles

The roadmap and future-consideration docs intentionally separate:

- **near-term shipping work** in [`docs/roadmap/roadmap.md`](../roadmap/roadmap.md)
- **long-horizon R&D** in [`docs/roadmap/future-consideration.md`](../roadmap/future-consideration.md)

The architectural risk is trying to implement all future-consideration themes as if they were one sprint backlog. That would couple:

- release hardening
- simulation engine redesign
- truth/audit systems
- memory systems
- edge optimization
- behavior policy tuning

into one unstable lane.

## Decision

`go-mirofish` will not implement "all future consideration now" and will not choose an arbitrary numeric half.

Instead, the project will develop toward the full target state through **stacked sprints** with this priority order:

1. **release confidence is the shipping gate**
2. **simulation depth is the core product differentiator**
3. **edge/local deployment is a constrained deployment profile**

This means:

- every sprint must preserve or improve operational trust
- sovereign-engine primitives can ship incrementally before full behavior complexity
- edge/local deployment must be implemented through bounded profiles, not through a parallel platform rewrite

## Current State

The authoritative current application state is:

- Go-native public control plane
- Go-native default simulation path
- Vue/Vite frontend
- reproducible benchmark/proof surfaces
- Headless SDK for Go and JavaScript
- Wasm and Starlark plugin/runtime support
- file-backed trust policy and signing helpers

This ADR assumes the current public path remains Go-owned and does not reintroduce a Python-first runtime model.

## Target Full-Developed State

The fully developed application state is defined as:

### Product state

- production-hard, observable, reproducible, and release-gated
- deeper simulation with explicit world rules, agent state, and action costs
- truth-aware reporting and claim handling
- long-horizon runs with compressed memory tiers
- local-first deployment from workstation to constrained ARM64 profile

### Engine state

- a **Governor** or equivalent orchestration core owns world rules
- agent actions are evaluated against non-negotiable Go rules
- the LLM provides proposals and narrative, not uncontrolled world physics
- plugin/runtime surfaces remain capability-gated and signed

### Deployment state

- workstation profile is the default full-capability mode
- constrained laptop and ARM64 modes are first-class, documented profiles
- Pi/edge claims remain evidence-based and device-verified before marketing language changes

## Architecture Boundaries

The system should evolve through these stable subsystem boundaries:

### 1. Gateway / control plane

Owns:

- public routes
- readiness
- health
- artifact validation
- release proof hooks
- operator-facing logs and failure classes

### 2. Governor core

Owns:

- discrete time ticks
- action validation
- action veto and costs
- world rules
- rule-enforced state transitions

### 3. Memory core

Owns:

- hot working state
- summarization
- durable compressed history
- compaction / janitor jobs

### 4. Truth core

Owns:

- claim records
- confidence and decay
- evidence references
- audit hooks and rumor labeling

### 5. Extension/runtime layer

Owns:

- Wasm plugins
- Starlark plugins
- capability-gated guest execution
- trust policy and signing

### 6. Deployment profile layer

Owns:

- workstation profile
- constrained local profile
- ARM64 edge profile
- concurrency caps
- model strategy defaults
- low-IO and low-memory behavior

## Non-Goals

This ADR explicitly rejects the following as a single combined sprint:

- full sovereign-agent society redesign
- complete oracle/audit ecosystem
- full behavioral economics system
- broad infrastructure expansion without measured need
- claiming Raspberry Pi verification without device evidence

## Sprint-by-Sprint Architecture Plan

The target state should be reached through the following sprint ADR sequence.

## Sprint 1: Confidence Spine

### Architectural decision

Prioritize production hardening and proof before deeper engine behavior.

### Scope

- readiness and health separation
- stronger logs and failure classification
- startup validation
- artifact/store hardening
- benchmark/stress/live-stack release gates

### Expected outcome

- the stack becomes easier to boot, debug, and trust
- release quality depends on proof, not optimism
- all future engine work lands on a stable operational base

### Exit criteria

- release commands are reproducible
- failure modes are visible and classifiable
- no silent partial writes on critical artifact paths

## Sprint 2: Sovereign-Core Primitives

### Architectural decision

Add engine primitives before policy complexity.

### Scope

- Governor interface or equivalent orchestration package
- discrete tick scheduler
- `AgentState`
- `ActionProposal`
- action cost / veto path
- explicit state-transition hooks

### Expected outcome

- the simulator gains a real engine spine
- the LLM stops being the sole source of behavior authority
- deeper simulation can grow without reworking the control plane

### Exit criteria

- simulation steps can be expressed as ticked state transitions
- action acceptance is rule-mediated in Go
- engine primitives are testable without policy tuning

## Sprint 3: Truth and Claim Layer

### Architectural decision

Truth control must be structural, not only prompt-based.

### Scope

- `ClaimRecord`
- confidence labels
- decay policy
- evidence references
- optional audit hook points
- separation of grounded vs speculative vs contested output

### Expected outcome

- reports become more defensible
- rumor cascades can be modeled and contained
- future audit/oracle systems have a stable substrate

### Exit criteria

- claims can be classified independently from narration
- confidence can decay over time
- report output can surface truth state explicitly

## Sprint 4: Memory Tiers and Long-Horizon Runs

### Architectural decision

Long simulations require managed compression, not unbounded raw history.

### Scope

- hot memory tier
- summary tier
- durable history tier
- summarization / compaction hooks
- janitor/background maintenance path

### Expected outcome

- longer useful runs without context collapse
- lower storage growth pressure
- better continuity between simulation phases

### Exit criteria

- volatile history can be compacted safely
- summaries are attached to durable state
- long-run memory cost is bounded by policy

## Sprint 5: Edge and Local Deployment Profiles

### Architectural decision

Edge support is a profile problem, not a separate application.

### Scope

- deployment profile model
- workstation, constrained-local, and ARM64-edge defaults
- concurrency caps
- low-IO / low-memory policies
- build and packaging proof for ARM64

### Expected outcome

- the same engine can run across workstation and edge profiles
- local-first positioning becomes technically coherent
- Pi/ARM64 work stops being aspirational-only

### Exit criteria

- ARM64 build path stays green
- profile-based runtime behavior is configurable
- documentation clearly separates "ready" from "device verified"

## Sprint 6: Behavior Depth and Audit Expansion

### Architectural decision

Only after the engine spine, truth layer, and memory tiers are stable should richer social dynamics land.

### Scope

- bias priors
- entropy injection
- judgment-action gap tuning
- limited audit/oracle role expansion
- dialectical or friction-first reporting modes

### Expected outcome

- simulation depth becomes a visible differentiator
- reports become more useful for foresight and stress-testing
- agent behavior reflects cost, disagreement, and uncertainty more realistically

### Exit criteria

- behavior policies are tunable data/rule systems
- reporting can surface conflict and tension intentionally
- no major regression in release confidence or runtime stability

## Expected State After the Sequence

If the sprint sequence completes successfully, the application should reach this state:

- a release-grade Go-native simulator
- a sovereign-core engine with explicit world physics
- truth-aware and memory-aware long-running simulation
- signed, capability-gated extension surfaces
- bounded edge/local deployment profiles

## Tradeoffs

### Accepted tradeoffs

- slower visible feature breadth in exchange for engine coherence
- phased behavior realism instead of a one-sprint "smart swarm" push
- profile-based edge support instead of broad hardware promises

### Rejected tradeoffs

- implementing all future-consideration items immediately
- treating release confidence and engine depth as separate products
- optimizing for Raspberry Pi first at the expense of core engine quality

## Operational Rules

- no sprint may weaken release confidence to gain simulation novelty
- no edge claim may outrun evidence
- no behavior-policy layer may bypass Go-owned engine rules
- no plugin/runtime expansion may bypass trust or capability gates

## Success Metrics

The sequence is successful when:

- release proof is deterministic
- deeper simulation no longer depends on prompt-only behavior
- truth labeling is explicit
- long-run memory is bounded
- workstation and constrained local profiles are both first-class

## Follow-on ADRs

This ADR should be followed by narrower ADRs when each sprint begins, for example:

- Governor state model
- truth classification contracts
- memory compaction rules
- deployment profile contracts
- audit/oracle expansion rules

This file defines the **ordering and target state**, not the final internal design of every subsystem.
