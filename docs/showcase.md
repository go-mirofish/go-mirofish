# Showcase

The in-app **Showcase** page ([/docs/showcase](/docs/showcase)) is built from Vue i18n strings and screenshots in the repository. This file exists so the docs manifest can point at a stable repo path for “edit on GitHub” links.

**Product flow (five steps):** upload seed material → ontology & graph → environment / personas → multi-agent simulation → report and deep interaction.

**Live static demo:** [gomirofish.vercel.app](https://gomirofish.vercel.app)

**Run locally:** [Installation](./getting-started/installation.md) — `make up` then `npm run dev`, UI at port **5173**.

**Benchmark evidence:** [Bundled JSON](./bundled-benchmarks/README.md) and the generated [benchmark report](./report/benchmark-report.md).

**Screenshot assets** (also in the root [README](../README.md) gallery): [`static/image/Screenshot/Screenshot(1).png`](../static/image/Screenshot/Screenshot(1).png) through [`Screenshot(9).png`](../static/image/Screenshot/Screenshot(9).png) — the in-app Showcase loads the same files from `frontend/public/static/image/Screenshot/` at build time.
