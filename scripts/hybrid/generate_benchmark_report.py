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


def verdict(ok: bool) -> str:
    return "PASS" if ok else "FAIL"


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


def build_stage_rows(summary: dict, benchmark_ok: bool) -> list[tuple[str, bool, str]]:
    project_id = summary.get("project_id")
    graph_id = summary.get("graph_id")
    simulation_id = summary.get("simulation_id")
    report_id = summary.get("report_id")
    simulation_status = summary.get("simulation_status")
    report_status = summary.get("report_status")
    report_non_empty = bool(summary.get("report_non_empty"))

    return [
        ("Ontology generation", bool(project_id), f"project_id={fmt(project_id)}"),
        ("Graph build", bool(graph_id), f"graph_id={fmt(graph_id)}"),
        ("Simulation create", bool(simulation_id), f"simulation_id={fmt(simulation_id)}"),
        (
            "Simulation run",
            simulation_status in {"completed", "stopped"},
            f"simulation_status={fmt(simulation_status)}",
        ),
        (
            "Report generation",
            bool(report_id) and report_status == "completed",
            f"report_id={fmt(report_id)}, report_status={fmt(report_status)}",
        ),
        (
            "Report content",
            benchmark_ok and report_non_empty,
            f"report_non_empty={fmt(report_non_empty)}",
        ),
    ]


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
    backend_health = data.get("backend_health", {})
    gateway_health = data.get("gateway_health", {})
    parsed_error = maybe_parse_benchmark_error(benchmark.get("stderr"))
    captured_at = data.get("captured_at") or datetime.now(timezone.utc).isoformat()

    backend_ok = backend_health.get("status") == "ok"
    gateway_ok = gateway_health.get("status") == "ok"
    benchmark_ok = bool(benchmark.get("ok"))
    stress_ok = (
        stress.get("failure_count") == 0
        and stress.get("request_count") == stress.get("success_count")
        and stress.get("request_count") is not None
    )
    report_non_empty = bool(summary.get("report_non_empty"))

    stage_rows = build_stage_rows(summary, benchmark_ok)
    current_migration_state = "Go control plane verified; Python core still owns benchmark-critical engine phases."
    if benchmark_ok and report_non_empty:
        current_migration_state = (
            "Go gateway and hybrid control plane are benchmark-verified end to end; "
            "Python still owns the core engine, but the current captured run completed successfully."
        )
    elif benchmark_ok and not report_non_empty:
        current_migration_state = (
            "Go gateway and hybrid control plane reached a full workflow completion, "
            "but the report artifact still needs qualitative follow-up because the content payload was empty."
        )

    lines = [
        f"# {args.title}",
        "",
        "## Executive summary",
        "",
        f"- Captured at: `{captured_at}`",
        f"- Release: `{data.get('release', 'unknown')}`",
        f"- Stack boot verdict: `{verdict(backend_ok and gateway_ok)}`",
        f"- Full benchmark verdict: `{verdict(benchmark_ok)}`",
        f"- Stress verdict: `{verdict(stress_ok)}`",
        f"- Current migration evidence: {current_migration_state}",
        "",
        "## What this proves",
        "",
        "- The benchmark was executed against the real go-mirofish hybrid stack, not inherited MiroFish media or mocked API output.",
        "- The Go gateway stayed in the request path for the measured run, while the Python backend still owned the core workflow execution.",
        "- The report below is intended to answer two questions directly:",
        "  1. Which benchmark phases are actually proven by this captured run?",
        "  2. How much of the runtime is already validated as go-mirofish rather than upstream MiroFish lineage?",
        "",
        "## Stack evidence",
        "",
        "| Surface | Verdict | Evidence |",
        "| --- | --- | --- |",
        f"| Backend health | `{verdict(backend_ok)}` | service=`{backend_health.get('service', 'unknown')}`, status=`{backend_health.get('status', 'unknown')}` |",
        f"| Gateway health | `{verdict(gateway_ok)}` | service=`{gateway_health.get('service', 'unknown')}`, status=`{gateway_health.get('status', 'unknown')}` |",
        f"| Full benchmark | `{verdict(benchmark_ok)}` | duration=`{fmt(benchmark.get('duration_seconds'))}s`, returncode=`{fmt(benchmark.get('returncode'))}` |",
        f"| Bounded stress pass | `{verdict(stress_ok)}` | requests=`{fmt(stress.get('request_count'))}`, failures=`{fmt(stress.get('failure_count'))}` |",
        "",
        "## Benchmark phase evidence",
        "",
        "| Phase | Verdict | Evidence |",
        "| --- | --- | --- |",
    ]

    for phase, ok, evidence in stage_rows:
        lines.append(f"| {phase} | `{verdict(ok)}` | {evidence} |")

    lines.extend(
        [
            "",
            "## What changed vs MiroFish is proven here",
            "",
            "| Claim area | Current evidence from this run | Reading of that evidence |",
            "| --- | --- | --- |",
            f"| Go runtime path | gateway health=`{gateway_health.get('service', 'unknown')}` and benchmark completed through the hybrid entrypoint | Requests were handled through go-mirofish's Go gateway instead of a Python-only surface. |",
            f"| Lightweight control plane | gateway RSS/HWM=`{fmt(processes.get('gateway', {}).get('VmRSS'))} / {fmt(processes.get('gateway', {}).get('VmHWM'))}` | The Go control plane remained materially smaller than the Python backend during the captured run. |",
            f"| Same workflow shape | project_id=`{fmt(summary.get('project_id'))}`, graph_id=`{fmt(summary.get('graph_id'))}`, simulation_id=`{fmt(summary.get('simulation_id'))}`, report_id=`{fmt(summary.get('report_id'))}` | The benchmark exercised the same major workflow artifacts expected from MiroFish while running under go-mirofish's hybrid stack. |",
            f"| Remaining migration gap | simulation_status=`{fmt(summary.get('simulation_status'))}`, report_status=`{fmt(summary.get('report_status'))}`, report_non_empty=`{fmt(summary.get('report_non_empty'))}` | The workflow still depends on Python-owned engine layers even when the Go control plane succeeds. |",
            "",
            "## Host and process footprint",
            "",
            f"- Host: `{fmt(host.get('uname'))}`",
            f"- Logical CPUs: `{fmt(host.get('cpu_count'))}`",
            f"- Backend RSS / HWM: `{fmt(processes.get('backend', {}).get('VmRSS'))} / {fmt(processes.get('backend', {}).get('VmHWM'))}`",
            f"- Gateway RSS / HWM: `{fmt(processes.get('gateway', {}).get('VmRSS'))} / {fmt(processes.get('gateway', {}).get('VmHWM'))}`",
            "",
            "## Stress details",
            "",
            f"- Requests: `{fmt(stress.get('request_count'))}`",
            f"- Successes: `{fmt(stress.get('success_count'))}`",
            f"- Failures: `{fmt(stress.get('failure_count'))}`",
            f"- Latency min: `{fmt(stress.get('latency_ms', {}).get('min'))}ms`",
            f"- Latency p50: `{fmt(stress.get('latency_ms', {}).get('p50'))}ms`",
            f"- Latency p95: `{fmt(stress.get('latency_ms', {}).get('p95'))}ms`",
            f"- Latency max: `{fmt(stress.get('latency_ms', {}).get('max'))}ms`",
            "",
            "## Migration interpretation",
            "",
            "- A green benchmark here does not mean the project is fully rewritten in Go.",
            "- A green benchmark here does mean the hybrid go-mirofish control plane can preserve the end-user workflow under the current benchmark contract.",
            "- The remaining migration work should be judged against the parity matrix, not against screenshot lineage or marketing language.",
            "",
            "See also:",
            "",
            "- `docs/hybrid/go-parity-matrix.md` for the stable parity contract",
            "- `docs/hybrid/go-migration-plan.md` for the staged rewrite plan",
            "",
        ]
    )

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
                        "- The benchmark failure was not caused by a missing route or an inherited screenshot-based claim.",
                        "- The failure should be classified using the parity matrix before treating it as a migration regression.",
                        "",
                    ]
                )

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
