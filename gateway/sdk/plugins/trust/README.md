# Plugin Trust Policy

This package provides a minimal trust model for SDK plugins.

Current support:

- SHA-256 digest verification
- Ed25519 signature verification
- trusted signer allowlist
- optional unsigned-plugin allowance for development
- JSON-backed trust policy loading for repeatable deployment config
- Ed25519 signing helpers for manifest + module production

The trust model is runtime-agnostic and can be used with Wasm or future Starlark plugins.

## JSON policy format

```json
{
  "require_digest": true,
  "require_signed": true,
  "allow_unsigned": false,
  "trusted_signers": {
    "core": "BASE64_ED25519_PUBLIC_KEY"
  }
}
```

Use `LoadPolicyFile(...)` when you want a first-party, file-backed trust configuration instead of constructing `Policy` in code.

## Signing helpers

Use these helpers when you want to produce trusted plugin manifests with the same payload format the verifier expects:

- `LoadPrivateKeyFile(path)`
- `ParsePrivateKey(raw)`
- `SignManifestAndModule(manifest, module, signerID, privateKey)`
