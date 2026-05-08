# Wasm Greeter Example

This is the first repo-owned Wasm plugin example for the `go-mirofish` SDK.

Files:

- `manifest.json` — portable plugin manifest
- `greet-rust.wasm` — compiled guest module
- `rust/greet.rs` — source for the Rust guest
- `Cargo.toml` — build config for the Rust guest

## What it demonstrates

- manifest-based plugin loading
- Wasm guest invocation through the Go-native plugin runtime
- string input/output over guest memory
- host logging capability
- ABI version export

## Current guest contract

- entry: `greeting`
- allocate: `allocate`
- deallocate: `deallocate`
- capability: `log`

## Notes

- the current fixture uses the same Rust guest shape as Wazero’s allocation example, but it is now copied into this repository so the SDK does not depend on external module fixtures for its example path
