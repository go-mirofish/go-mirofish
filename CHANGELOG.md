# Changelog

All notable changes to `go-mirofish` will be documented in this file.

This project uses a curated, release-oriented changelog:
- human-written release notes for important product changes
- conventional-commit history as supporting input
- version tags as the source of release boundaries

## v0.1.0 - 2026-04-24

Initial public `go-mirofish` release.

### Added

- Added a Go gateway layer with route aliases, static asset serving, health endpoint support, and gateway tests.
- Added hybrid runtime entrypoints with `start.sh`, `start.bat`, organized Dockerfiles, and a single canonical `docker-compose.yml`.
- Added benchmark and verification tooling, including a seed fixture, contract matrix, contract verifier, benchmark smoke runner, and release/showcase documentation.
- Added local OpenAI-compatible provider support for setups like `llama.cpp` and Ollama without requiring a real cloud API key for localhost deployments.
- Added contributor and release tooling including Husky, Commitlint, Changesets, Renovate, release workflows, and the release changelog generator.

### Changed

- Reworked the repo around the `go-mirofish` product identity across docs, package metadata, runtime names, gateway binaries, and health/service labels.
- Centralized the container/runtime layout so Docker orchestration and gateway packaging live in one organized surface.
- Improved the source-development workflow for Windows and Git Bash by replacing shell-fragile commands with cross-platform Node wrappers.
- Shifted the checked-in application surface to English-first source strings and collapsed the locale registry to English for the current public release.

### Fixed

- Fixed gateway compatibility for aliased API routes used by the frontend and hybrid runtime.
- Fixed local startup script resolution for Windows venv layouts and renamed gateway binaries.
- Fixed multiple backend/runtime text, parser, and script issues uncovered during the English-only cleanup and hybrid migration work.
- Hardened LLM client retry behavior for transient OpenAI-compatible provider failures.

### Documentation

- Rewrote installation, provider, Ollama, hybrid runtime, showcase, and Raspberry Pi validation docs for the `go-mirofish` release line.
- Removed misleading upstream screenshot/demo framing from the active release surface and replaced it with first-party proof expectations.

### Notes

- Raspberry Pi support is currently positioned as `ARM64-ready` and pending on-device validation.
- First-party screenshots and demo recordings are still expected to be captured from a browser-capable runtime session and added separately.
