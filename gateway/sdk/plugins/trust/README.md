# Plugin Trust Policy

This package provides a minimal trust model for SDK plugins.

Current support:

- SHA-256 digest verification
- Ed25519 signature verification
- trusted signer allowlist
- optional unsigned-plugin allowance for development

The trust model is runtime-agnostic and can be used with Wasm or future Starlark plugins.
