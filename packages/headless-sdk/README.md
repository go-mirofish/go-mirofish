# go-mirofish-sdk

Headless JavaScript SDK for the `go-mirofish` gateway APIs.

## Install

```bash
npm install go-mirofish-sdk
```

## Quick start

```js
import createHeadlessSDK from 'go-mirofish-sdk'

const sdk = createHeadlessSDK({
  baseURL: 'http://127.0.0.1:3000',
})

const health = await sdk.system.getHealth()
console.log(health)
```

## What it provides

- graph APIs
- simulation APIs
- report APIs
- health / ready / metrics / providers helpers
- retry handling for non-4xx request failures

## Main export

- `createHeadlessSDK(options)`
- `createTransport(options)`
- `requestWithRetry(requestFn, maxRetries, delayMs)`

## Integration goal

This package is intended to be the JavaScript “just import and plug in” client for a running `go-mirofish` gateway.

It does **not** bundle the gateway binary.

It does **not** replace the Go module SDK at:

```go
github.com/go-mirofish/go-mirofish/gateway/sdk/headless
```

Instead, it gives JavaScript and Node consumers a stable client package for the same HTTP surface.
