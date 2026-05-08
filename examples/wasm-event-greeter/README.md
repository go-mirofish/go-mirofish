# Wasm Event Greeter Example

This is the first repo-owned Wasm plugin example that emits host events.

Files:

- `manifest.json` — portable plugin manifest
- `event-greeter.wasm` — compiled guest module
- `src/lib.rs` — source for the Rust guest

## What it demonstrates

- ABI version export
- manifest-based plugin loading
- string input/output over guest memory
- host logging capability
- host event emission capability
- host time capability
