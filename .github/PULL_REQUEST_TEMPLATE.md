<!--
Commits: docs/contributing/commit-messages.md
GitHub only loads this file name for PR bodies, not PR_TEMPLATE.md.
-->

## PR summary

**Closes:** #<!-- issue number -->

**Commit style:** `type(scope): subject`. See [commit-messages.md](../docs/contributing/commit-messages.md).

---

## Layer 1: Intent parsing

**Primary goal**

<!-- What problem are we solving? What outcome do we want? -->

**User story**

<!-- As a [role], I want [capability], so that [benefit]. -->

**Impact**

<!-- Users, local-first goals, upstream parity, docs, maintenance. -->

**Metadata**

| Field | Value |
| --- | --- |
| Type | Feature / Bug / Chore / Docs / Epic |
| Area | gateway-go, frontend, contracts-parity, benchmark-ci, docker-infra, docs, upstream-sync, security, release |
| Migration phase | phase-1 … phase-6 or n/a |
| Related issues | <!-- #123 --> |

---

## Layer 2: Knowledge retrieval

**Skills / learning**

- [ ] Skills needed (Go, Vue, Docker, CI, …)
- [ ] Gaps closed in this PR (spike notes, links)

**Effort**

<!-- S / M / L, rough -->

**Resources reviewed**

- [ ] README / docs / `.env.example` / touched code paths
- [ ] Other: <!-- -->

**Architecture / diagram**

<!-- Link or “n/a”. -->

---

## Layer 3: Constraint analysis

**Dependencies**

- [ ] Blocked by / unblocks: <!-- or None -->
- [ ] External: <!-- LLM, Zep, … -->

**Technical constraints**

<!-- Parity, AGPL, protected Python paths, … -->

**Risks**

<!-- What could go wrong; what you did to mitigate. -->

---

## Layer 4: Solution generation

**Approach**

<!-- Design and tradeoffs. -->

**Design checklist**

- [ ] Additive-first / fork-friendly where possible
- [ ] Errors & UX considered
- [ ] Tests / verification performed
- [ ] No secrets in repo

**Acceptance criteria (from issue; check when done)**

- [ ]
- [ ]

---

## Layer 5: What this PR does

**Changes**

1.
2.
3.

**Files / areas**

<!-- High-level: gateway / backend / frontend / docs / ci -->

**Env / migrations**

<!-- `.env` or schema changes? Link issue decision if any. -->

---

## Layer 6: Validation & review

**How tested**

<!-- Commands, manual steps, screenshots. -->

**Quality**

- [ ] CI green
- [ ] Commits follow [Conventional Commits + scopes](../docs/contributing/commit-messages.md)
- [ ] `docs/` or README updated if user-facing behavior changed

**For reviewers**

<!-- Risk areas, focus files, follow-ups. -->
