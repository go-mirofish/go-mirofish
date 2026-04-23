#!/usr/bin/env python3
"""
Verify the additive hybrid migration contract freeze against the current repo.

This script is intentionally stdlib-only so it can run before backend deps are installed.
"""

from __future__ import annotations

import pathlib
import re
import sys


ROOT = pathlib.Path(__file__).resolve().parents[2]
GRAPH_API = ROOT / "backend" / "app" / "api" / "graph.py"
SIM_API = ROOT / "backend" / "app" / "api" / "simulation.py"
REPORT_API = ROOT / "backend" / "app" / "api" / "report.py"
FRONTEND_SIM = ROOT / "frontend" / "src" / "api" / "simulation.js"
FRONTEND_REPORT = ROOT / "frontend" / "src" / "api" / "report.js"
SEED = ROOT / "benchmark" / "seed.txt"
MATRIX = ROOT / "docs" / "hybrid" / "contract-matrix.md"


def load_text(path: pathlib.Path) -> str:
    return path.read_text(encoding="utf-8")


def require(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


def collect_route_decorators(text: str, blueprint: str) -> set[tuple[str, tuple[str, ...]]]:
    pattern = re.compile(
        rf"@{re.escape(blueprint)}\.route\('([^']+)'\s*,\s*methods=\[([^\]]+)\]\)",
        re.MULTILINE,
    )
    routes: set[tuple[str, tuple[str, ...]]] = set()
    for route, methods_blob in pattern.findall(text):
        methods = tuple(part.strip().strip("'\"") for part in methods_blob.split(","))
        routes.add((route, methods))
    return routes


def benchmark_checks() -> list[str]:
    text = load_text(SEED).strip()
    words = len(text.split())
    size = len(text.encode("utf-8"))

    require(400 <= words <= 600, f"benchmark/seed.txt word count {words} outside 400-600")
    require(size < 5_000, f"benchmark/seed.txt size {size} bytes is >= 5000")
    require("October 18, 2026" in text, "benchmark/seed.txt is missing the dated triggering event")

    return [
        f"benchmark seed words: {words}",
        f"benchmark seed bytes: {size}",
    ]


def route_checks() -> list[str]:
    graph_routes = collect_route_decorators(load_text(GRAPH_API), "graph_bp")
    sim_routes = collect_route_decorators(load_text(SIM_API), "simulation_bp")
    report_routes = collect_route_decorators(load_text(REPORT_API), "report_bp")

    require(("/ontology/generate", ("POST",)) in graph_routes, "missing graph ontology generate route")
    require(("/prepare", ("POST",)) in sim_routes, "missing simulation prepare route")
    require(("/prepare/status", ("POST",)) in sim_routes, "missing simulation prepare status route")
    require(("/start", ("POST",)) in sim_routes, "missing simulation start route")
    require(("/<simulation_id>/run-status", ("GET",)) in sim_routes, "missing simulation run-status route")
    require(("/generate", ("POST",)) in report_routes, "missing report generate route")
    require(("/generate/status", ("POST",)) in report_routes, "missing report generate/status POST route")
    require(("/<report_id>/progress", ("GET",)) in report_routes, "missing report progress route")
    require(("/<report_id>/download", ("GET",)) in report_routes, "missing report download route")

    return [
        "backend graph route: /api/graph/ontology/generate [POST]",
        "backend simulation routes: prepare, prepare/status, start, <id>/run-status",
        "backend report routes: generate, generate/status [POST], <report_id>/progress, <report_id>/download",
    ]


def frontend_checks() -> list[str]:
    frontend_sim = load_text(FRONTEND_SIM)
    frontend_report = load_text(FRONTEND_REPORT)

    require("/api/simulation/start" in frontend_sim, "frontend simulation API missing /api/simulation/start")
    require("/api/simulation/${simulationId}/run-status" in frontend_sim, "frontend simulation API missing run-status caller")
    require("/api/report/generate/status" in frontend_report, "frontend report API missing generate/status caller")
    require("report_id: reportId" in frontend_report, "frontend report API missing report_id query shape")

    return [
        "frontend simulation caller: /api/simulation/start",
        "frontend simulation caller: /api/simulation/${simulationId}/run-status",
        "frontend report caller: GET /api/report/generate/status?report_id=...",
    ]


def matrix_checks() -> list[str]:
    matrix = load_text(MATRIX)
    required_snippets = [
        "Alias `run -> start`",
        "Alias `status -> run-status`",
        "Method bridge plus `report_id -> /api/report/:report_id/progress`",
        "Gateway policy",
    ]
    for snippet in required_snippets:
        require(snippet in matrix, f"contract matrix missing snippet: {snippet}")
    return ["contract matrix contains the documented gateway alias policies"]


def main() -> int:
    checks: list[str] = []
    checks.extend(benchmark_checks())
    checks.extend(route_checks())
    checks.extend(frontend_checks())
    checks.extend(matrix_checks())

    for line in checks:
        print(f"PASS: {line}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except AssertionError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        raise SystemExit(1)
