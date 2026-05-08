# Starlark Plugin Runtime

Import path:

```go
github.com/go-mirofish/go-mirofish/gateway/sdk/plugins/starlark
```

This is the second plugin path after Wasm.

## Why Starlark

- pure Go runtime
- deterministic, policy-friendly scripting
- Python-like syntax
- good fit for agent rules and safe configuration logic

## Current scope

- manifest-backed script loading
- capability-gated builtins
- byte/string input -> string output
- log / emit_event / time_now / random helper support
- bounded execution steps
- context-driven cancellation
