# Plugin SDK

This directory holds the next-generation plugin SDK surfaces for `go-mirofish`.

Current direction:

- **Wasm-first** plugin runtime via `wazero`
- pure Go host runtime
- versioned plugin ABI
- host callbacks for logging and event emission

The first implementation lives under `plugins/wasm`.

Portable pieces now available:

- plugin manifest model
- capability set
- directory discovery
- registry / registration model
- runtime-aware default module resolution:
  - `wasm` -> `plugin.wasm`
  - `starlark` -> `plugin.star`
- file-backed trust policy loading
- signing helpers for manifest + module release artifacts
