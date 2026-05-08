# Wasm Plugin Runtime

Import path:

```go
github.com/go-mirofish/go-mirofish/gateway/sdk/plugins/wasm
```

This package is the first step toward a strictly Go-native, multi-language plugin SDK.

## Design

- runtime: `wazero`
- host language: Go
- guest language: anything that can compile to WebAssembly
- transport: guest memory + exported functions
- host callbacks: logging and event emission

## Default contract

The default contract expects guest exports:

- `mirofish_abi_version() -> i32`
- `allocate(size) -> ptr`
- `deallocate(ptr, size)`
- `run(ptr, size) -> packed(ptr,size)`

Portable manifests can map onto this contract with:

- plugin name
- plugin version
- runtime kind
- API version
- entry export
- allocator exports
- optional start functions
- explicit capabilities

The default host module is:

- module: `mirofish`
- log import: `log`
- event import: `emit_event`

## Why Wasm first

- pure Go runtime
- strong sandboxing
- language-agnostic guest model
- good long-term fit for edge and local-first deployment

## Current scope

This is the minimal runtime scaffold:

- compile guest modules
- parse and validate plugin manifests
- validate ABI
- invoke byte-oriented guest functions
- capture guest logs and emitted events
- load plugins from manifest + module directories

## Current capabilities

- `log`
- `emit_event`
- `time_now`
- `random_bytes`

These are capability-gated in the host runtime so future plugins can declare the minimum surface they need.

## First-party example

Use the repo-owned example:

- `examples/wasm-greeter/manifest.json`
- `examples/wasm-greeter/greet-rust.wasm`
- `examples/wasm-greeter/rust/greet.rs`

Starlark or other embedded interpreters can sit beside this later, but Wasm is the primary multi-language path.
