#!/usr/bin/env python3
"""
Run a bounded concurrent probe against go-mirofish health endpoints.
"""

from __future__ import annotations

import argparse
import concurrent.futures
import json
import statistics
import time
import urllib.error
import urllib.request


def hit(url: str) -> dict:
    started = time.perf_counter()
    try:
        with urllib.request.urlopen(url, timeout=10) as resp:
            body = resp.read()
        return {
            "ok": True,
            "url": url,
            "status": resp.status,
            "elapsed_ms": round((time.perf_counter() - started) * 1000, 2),
            "bytes": len(body),
        }
    except Exception as exc:  # pragma: no cover - exercised in live runs
        return {
            "ok": False,
            "url": url,
            "elapsed_ms": round((time.perf_counter() - started) * 1000, 2),
            "error": repr(exc),
        }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--gateway-url", default="http://127.0.0.1:3000/health")
    parser.add_argument("--backend-url", default="http://127.0.0.1:5001/health")
    parser.add_argument("--rounds", type=int, default=40)
    parser.add_argument("--workers", type=int, default=16)
    args = parser.parse_args()

    urls = [args.gateway_url, args.backend_url]
    samples: list[dict] = []

    with concurrent.futures.ThreadPoolExecutor(max_workers=args.workers) as executor:
        futures = [executor.submit(hit, url) for _ in range(args.rounds) for url in urls]
        for future in concurrent.futures.as_completed(futures):
            samples.append(future.result())

    oks = [sample for sample in samples if sample["ok"]]
    failures = [sample for sample in samples if not sample["ok"]]
    latencies = sorted(sample["elapsed_ms"] for sample in oks)

    report = {
        "request_count": len(samples),
        "success_count": len(oks),
        "failure_count": len(failures),
        "latency_ms": {
            "min": latencies[0] if latencies else None,
            "p50": round(statistics.median(latencies), 2) if latencies else None,
            "p95": round(latencies[max(0, int(len(latencies) * 0.95) - 1)], 2) if latencies else None,
            "max": latencies[-1] if latencies else None,
        },
        "failures": failures[:10],
    }

    print(json.dumps(report, indent=2))
    return 0 if not failures else 1


if __name__ == "__main__":
    raise SystemExit(main())
