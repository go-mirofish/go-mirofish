---
name: Issue (bugs, ideas, tasks, planning)
about: Report a bug, suggest a feature, track tasks, or host a focused discussion
title: ""
labels: []
assignees: []
---

## How we use GitHub Issues

Issues in **go-mirofish** are for **bugs**, **ideas**, **work tracking**, **planning**, and **focused discussion**. Implementation work is summarized in pull requests using the **six-layer** template in [`PULL_REQUEST_TEMPLATE.md`](../PULL_REQUEST_TEMPLATE.md) (GitHub accepts that name under `.github/`; there is no separate `PR_TEMPLATE.md`).

This file lives under [`.github/ISSUE_TEMPLATE/`](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/configuring-issue-templates-for-your-repository) so it appears in the **new issue** template chooser (current GitHub workflow).

### Common uses

| Use | Purpose |
| --- | --- |
| **Bug tracking** | Document errors with **steps to reproduce**, **expected** vs **actual** behavior, and environment (OS, Python/Node, Docker). |
| **Feature requests** | Propose **new behavior** or **enhancements**—problem you’re solving, who benefits, rough idea (not a full design spec unless you want). |
| **Task management** | Break epics into **small, actionable** items; use **task lists** (`- [ ]`) so progress is visible in the issue. |
| **Project planning** | Tie issues to **[GitHub Projects](https://docs.github.com/en/issues/planning-and-tracking-with-projects)** (board or roadmap) and **milestones** for releases or phases. |
| **Team discussion** | One thread per **design or implementation** decision so history stays searchable (link related PRs when decided). |

Official reference: [About issue and pull request templates](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/about-issue-and-pull-request-templates).

### Key GitHub features

- **Labels** — e.g. `bug`, `enhancement`, `good first issue`, plus area/phase labels your maintainers define.
- **Assignees** — who is **responsible** for driving the issue to closure.
- **Milestones** — group issues for a **release**, **phase**, or **deadline**.
- **Sub-issues** — **nest** smaller tasks under a parent for hierarchy ([about sub-issues](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/adding-sub-issues)).
- **Issue templates** — stored in **`.github/ISSUE_TEMPLATE/`** with valid `name:` and `about:` in YAML front matter ([configuring templates](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/configuring-issue-templates-for-your-repository)).

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

### Details

<!-- For bugs: steps to reproduce, expected behavior, actual behavior, logs/screenshots. -->
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

**Deep planning:** For large or cross-cutting work, maintainers may ask for the full **six-layer** breakdown—that structure lives in [`PULL_REQUEST_TEMPLATE.md`](../PULL_REQUEST_TEMPLATE.md) and in [docs/contributing/github-pr-6-layer.md](../../docs/contributing/github-pr-6-layer.md).
