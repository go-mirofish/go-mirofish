---
name: Issue (bugs, ideas, tasks, planning)
about: Report a bug, suggest a feature, track tasks, or host a focused discussion
title: ""
labels: []
assignees: []
---

## How we use GitHub Issues

Issues in **go-mirofish** are for **bugs**, **ideas**, **work tracking**, **planning**, and **focused discussion**. Implementation work is summarized in pull requests using the **six-layer template** in [`.github/PULL_REQUEST_TEMPLATE.md`](./PULL_REQUEST_TEMPLATE.md) (GitHub requires that exact filename; there is no separate `PR_TEMPLATE.md`).

### Common uses

| Use | Purpose |
| --- | --- |
| **Bug tracking** | Document errors with **steps to reproduce**, **expected** vs **actual** behavior, and environment (OS, Python/Node, Docker). |
| **Feature requests** | Propose **new behavior** or **enhancements**—problem you’re solving, who benefits, rough idea (not a full design spec unless you want). |
| **Task management** | Break epics into **small, actionable** items; use **task lists** (`- [ ]`) so progress is visible in the issue. |
| **Project planning** | Tie issues to **[GitHub Projects](https://docs.github.com/en/issues/planning-and-tracking-with-projects)** (board or roadmap) and **milestones** for releases or phases. |
| **Team discussion** | One thread per **design or implementation** decision so history stays searchable (link related PRs when decided). |

Official reference: [GitHub Docs — Issues](https://docs.github.com/en/issues).

### Key GitHub features

- **Labels** — e.g. `bug`, `enhancement`, `good first issue`, plus area/phase labels your maintainers define.
- **Assignees** — who is **responsible** for driving the issue to closure.
- **Milestones** — group issues for a **release**, **phase**, or **deadline**.
- **Sub-issues** — **nest** smaller tasks under a parent for hierarchy ([about sub-issues](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/adding-sub-issues)).
- **Issue templates** — this file is the **default**; it nudges everyone toward consistent detail ([about templates](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests)).

---

## Your issue

**What is this?** (check all that apply)

- [ ] Bug report
- [ ] Feature / enhancement
- [ ] Task / chore
- [ ] Discussion / question
- [ ] Planning note (epic / phase)

**Summary**

<!-- One short paragraph: what this issue is about. -->

### Details<!-- For bugs: steps to reproduce, expected behavior, actual behavior, logs/screenshots. -->
<!-- For features: problem today, proposed direction, non-goals if any. -->
<!-- For tasks: bullet list or task list. -->

### Task list (optional)

- [ ]
- [ ]

### Links

<!-- Related issues, PRs, discussions, docs paths. Closes # when a PR fixes this. -->

### Environment (bugs only)

<!-- OS, branch/commit, Docker vs source, relevant `.env` keys (never paste secrets). -->

---

**Deep planning:** For large or cross-cutting work, maintainers may ask for the full **six-layer** breakdown—that structure lives in [`.github/PULL_REQUEST_TEMPLATE.md`](./PULL_REQUEST_TEMPLATE.md) and in [docs/contributing/github-issues-6-layer.md](../docs/contributing/github-issues-6-layer.md).
