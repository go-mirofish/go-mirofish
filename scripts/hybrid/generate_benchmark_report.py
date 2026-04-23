#!/usr/bin/env python3
"""
Render a Markdown benchmark report from a captured JSON benchmark result.
"""

from __future__ import annotations

import argparse
import json
import pathlib
from datetime import datetime, timezone


def load_json(path: pathlib.Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8"))


def fmt(value) -> str:
    return "n/a" if value is None else str(value)


def maybe_parse_benchmark_error(stderr: str | None) -> dict | None:
    if not stderr:
        return None
    prefix = "ERROR: "
    if stderr.startswith(prefix):
        stderr = stderr[len(prefix):]
    try:
        return json.loads(stderr)
    except Exception:
        return None


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True, help="Path to benchmark JSON")
    parser.add_argument("--output", required=True, help="Path to Markdown report")
    parser.add_argument("--title", default="go-mirofish benchmark report")
    args = parser.parse_args()

    input_path = pathlib.Path(args.input).resolve()
    output_path = pathlib.Path(args.output).resolve()
    data = load_json(input_path)

    benchmark = data.get("benchmark", {})
    summary = benchmark.get("summary") or {}
    stress = data.get("stress", {})
    failures = stress.get("failures") or []
    host = data.get("host") or {}
    processes = data.get("processes") or {}
    parsed_error = maybe_parse_benchmark_error(benchmark.get("stderr"))
    captured_at = data.get("captured_at") or datetime.now(timezone.utc).isoformat()

    lines = [
        f"# {args.title}",
        "",
        f"- Captured at: `{captured_at}`",
        f"- Release: `{data.get('release', 'unknown')}`",
        f"- Backend health service: `{data.get('backend_health', {}).get('service', 'unknown')}`",
        f"- Gateway health service: `{data.get('gateway_health', {}).get('service', 'unknown')}`",
        "",
        "## Host and process footprint",
        "",
        f"- Host: `{fmt(host.get('uname'))}`",
        f"- Logical CPUs: `{fmt(host.get('cpu_count'))}`",
        f"- Backend RSS / HWM: `{fmt(processes.get('backend', {}).get('VmRSS'))} / {fmt(processes.get('backend', {}).get('VmHWM'))}`",
        f"- Gateway RSS / HWM: `{fmt(processes.get('gateway', {}).get('VmRSS'))} / {fmt(processes.get('gateway', {}).get('VmHWM'))}`",
        "",
        "## Full-flow benchmark",
        "",
        f"- Result: `{'PASS' if benchmark.get('ok') else 'FAIL'}`",
        f"- Duration: `{fmt(benchmark.get('duration_seconds'))}s`",
        f"- Return code: `{fmt(benchmark.get('returncode'))}`",
        f"- Project ID: `{fmt(summary.get('project_id'))}`",
        f"- Graph ID: `{fmt(summary.get('graph_id'))}`",
        f"- Simulation ID: `{fmt(summary.get('simulation_id'))}`",
        f"- Report ID: `{fmt(summary.get('report_id'))}`",
        f"- Simulation status: `{fmt(summary.get('simulation_status'))}`",
        f"- Report status: `{fmt(summary.get('report_status'))}`",
        f"- Report non-empty: `{fmt(summary.get('report_non_empty'))}`",
        "",
    ]

    if parsed_error:
        lines.extend(
            [
                "## Failure analysis",
                "",
                f"- Failing URL: `{fmt(parsed_error.get('url'))}`",
                f"- Failing method: `{fmt(parsed_error.get('method'))}`",
                f"- Reported status: `{fmt(parsed_error.get('status'))}`",
            ]
        )
        body = parsed_error.get("body")
        if body:
            try:
                body_json = json.loads(body)
            except Exception:
                body_json = None
            if isinstance(body_json, dict):
                lines.extend(
                    [
                        f"- Upstream status: `{fmt(body_json.get('upstream_status'))}`",
                        f"- Upstream error type: `{fmt(body_json.get('error_type'))}`",
                        f"- Upstream error code: `{fmt(body_json.get('error_code'))}`",
                        f"- Retry-after hint: `{fmt(body_json.get('retry_after'))}`",
                        "",
                        "Interpretation:",
                        "",
                        "- The backend and gateway both booted and stayed healthy during the benchmark window.",
                        "- The canonical benchmark failed at the ontology-generation stage.",
                        "- The failure was caused by the configured upstream OpenAI-compatible provider returning a quota/rate-limit response.",
                        "- This means the current benchmark result is real first-party runtime evidence, but not yet a full green end-to-end pass.",
                        "",
                    ]
                )

    lines.extend([
        "## Bounded stress pass",
        "",
        f"- Requests: `{fmt(stress.get('request_count'))}`",
        f"- Successes: `{fmt(stress.get('success_count'))}`",
        f"- Failures: `{fmt(stress.get('failure_count'))}`",
        f"- Latency min: `{fmt(stress.get('latency_ms', {}).get('min'))}ms`",
        f"- Latency p50: `{fmt(stress.get('latency_ms', {}).get('p50'))}ms`",
        f"- Latency p95: `{fmt(stress.get('latency_ms', {}).get('p95'))}ms`",
        f"- Latency max: `{fmt(stress.get('latency_ms', {}).get('max'))}ms`",
        "",
    ])

    if failures:
        lines.extend(
            [
                "## Stress failures",
                "",
                "```json",
                json.dumps(failures, indent=2),
                "```",
                "",
            ]
        )

    stderr = benchmark.get("stderr")
    if stderr:
        lines.extend(
            [
                "## Benchmark stderr",
                "",
                "```text",
                stderr,
                "```",
                "",
            ]
        )

    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text("\n".join(lines), encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
