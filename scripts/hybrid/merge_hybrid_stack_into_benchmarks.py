#!/usr/bin/env python3
"""
Merge hybrid stack fields from v0.1.0-live-benchmark.json into per-example
docs/bundled-benchmarks/*__*__latest.json and enrich examples-benchmark-suite.json
(if present under benchmark/results/).

Run from repo root: python scripts/hybrid/merge_hybrid_stack_into_benchmarks.py
"""
from __future__ import annotations

import json
import pathlib


STACK_KEYS = (
    "captured_at",
    "release",
    "host",
    "backend_health",
    "gateway_health",
    "processes",
    "benchmark",
    "stress",
)


def main() -> int:
    root = pathlib.Path(__file__).resolve().parents[2]
    live_path = root / "benchmark" / "results" / "v0.1.0-live-benchmark.json"
    live = json.loads(live_path.read_text(encoding="utf-8"))
    stack = {k: live[k] for k in STACK_KEYS if k in live}

    bundled = root / "docs" / "bundled-benchmarks"
    for path in sorted(bundled.glob("*__*__latest.json")):
        data = json.loads(path.read_text(encoding="utf-8"))
        if data.get("example_key") is None and "results" in data:
            continue
        merged = {**data, **stack}
        path.write_text(json.dumps(merged, indent=2) + "\n", encoding="utf-8")
        print("updated", path.relative_to(root))

    suite = root / "benchmark" / "results" / "examples-benchmark-suite.json"
    if suite.is_file():
        sdata = json.loads(suite.read_text(encoding="utf-8"))
        out = {**stack, **sdata}
        results = out.get("results")
        if isinstance(results, list):
            out["results"] = [{**row, **stack} for row in results if isinstance(row, dict)]
        suite.write_text(json.dumps(out, indent=2) + "\n", encoding="utf-8")
        print("updated", suite.relative_to(root))

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
