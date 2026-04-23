# Raspberry Pi Validation

This page defines what `go-mirofish` can and cannot claim about Raspberry Pi support.

## Claim levels

### 1. ARM64-ready

This is the strongest claim you can make **without** a physical Raspberry Pi.

Evidence for this level:

- `linux/arm64` gateway build artifacts exist
- the hybrid runtime is designed for a local-first gateway + Python stack
- benchmark and contract scripts exist
- no x86-only gateway assumptions are required for normal startup

This repo already has evidence for that level through:

- [gateway/go.mod](https://github.com/go-mirofish/go-mirofish/blob/main/gateway/go.mod)
- [gateway/cmd/mirofish-gateway/main.go](https://github.com/go-mirofish/go-mirofish/blob/main/gateway/cmd/mirofish-gateway/main.go)
- [.github/workflows/go-build.yml](https://github.com/go-mirofish/go-mirofish/blob/main/.github/workflows/go-build.yml)
- [scripts/hybrid/run_benchmark_smoke.py](https://github.com/go-mirofish/go-mirofish/blob/main/scripts/hybrid/run_benchmark_smoke.py)

### 2. ARM64 runtime-verified

This means the hybrid stack has been run on an ARM64 Linux target, but not necessarily on a real Raspberry Pi.

Useful evidence:

- ARM64 CI run logs
- ARM64 container startup logs
- benchmark output from an ARM64 Linux machine

This is stronger than build-only evidence, but still not “verified on Raspberry Pi 4/5”.

### 3. Raspberry Pi verified

This claim requires real on-device execution on a Raspberry Pi 4 or 5.

Required evidence:

- successful startup on the target Pi
- benchmark result captured from the target Pi
- recorded CPU/RAM/startup observations from the actual device

## What you can say today

Safe wording:

- `ARM64-compatible`
- `Designed for Raspberry Pi 4/5 64-bit Linux`
- `ARM64-ready, pending on-device Raspberry Pi validation`

Avoid this wording until a real Pi run is captured:

- `Verified on Raspberry Pi 4`
- `Verified on Raspberry Pi 5`
- `Raspberry Pi supported` without qualification

## Validation matrix

| Claim | Build evidence | Runtime evidence | Real Pi hardware required |
| --- | --- | --- | --- |
| ARM64-ready | Yes | No | No |
| ARM64 runtime-verified | Yes | Yes | No |
| Raspberry Pi verified | Yes | Yes | Yes |

## Minimum Pi proof package

When a Pi tester is available, capture these:

1. Gateway binary architecture
   - `file gateway/bin/mirofish-gateway`

2. Startup proof
   - `./start.sh`
   - `curl http://127.0.0.1:3000/health`
   - `curl http://127.0.0.1:5001/health`

3. Benchmark proof
   - `python3 scripts/hybrid/run_benchmark_smoke.py --base-url http://127.0.0.1:3000`

4. Resource notes
   - startup time
   - peak RAM observed
   - model/provider used

5. Save the result into a committed benchmark proof artifact once a real Pi run exists.

## Suggested tester checklist

```bash
npm run build:gateway
./start.sh
curl http://127.0.0.1:3000/health
curl http://127.0.0.1:5001/health
python3 scripts/hybrid/run_benchmark_smoke.py --base-url http://127.0.0.1:3000
```

## Release rule

Until a real Raspberry Pi result is committed, treat the project as:

- ARM64-ready
- benchmark-capable
- pending on-device Pi validation
