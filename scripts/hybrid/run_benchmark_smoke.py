#!/usr/bin/env python3
"""
Run the canonical benchmark flow against the hybrid API surface.

This script intentionally uses only the Python standard library so it can run
without the backend dependency graph.
"""

from __future__ import annotations

import argparse
import json
import mimetypes
import pathlib
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid


ROOT = pathlib.Path(__file__).resolve().parents[2]
SEED_PATH = ROOT / "benchmark" / "seed.txt"
DEFAULT_REQUIREMENT = (
    "Simulate how this fictional scenario evolves across public discussion, "
    "identify the key actors and likely sentiment shifts, and generate a concise analysis report."
)


def request_json(url: str, *, method: str = "GET", body: bytes | None = None, headers: dict[str, str] | None = None) -> dict:
    req = urllib.request.Request(url, data=body, method=method, headers=headers or {})
    with urllib.request.urlopen(req, timeout=30) as resp:
        payload = resp.read().decode("utf-8")
    return json.loads(payload)


def encode_multipart(fields: dict[str, str], files: dict[str, pathlib.Path]) -> tuple[bytes, str]:
    boundary = f"mirofish-{uuid.uuid4().hex}"
    chunks: list[bytes] = []

    for key, value in fields.items():
        chunks.extend(
            [
                f"--{boundary}\r\n".encode(),
                f'Content-Disposition: form-data; name="{key}"\r\n\r\n'.encode(),
                value.encode("utf-8"),
                b"\r\n",
            ]
        )

    for key, path in files.items():
        filename = path.name
        mime = mimetypes.guess_type(filename)[0] or "text/plain"
        chunks.extend(
            [
                f"--{boundary}\r\n".encode(),
                f'Content-Disposition: form-data; name="{key}"; filename="{filename}"\r\n'.encode(),
                f"Content-Type: {mime}\r\n\r\n".encode(),
                path.read_bytes(),
                b"\r\n",
            ]
        )

    chunks.append(f"--{boundary}--\r\n".encode())
    body = b"".join(chunks)
    content_type = f"multipart/form-data; boundary={boundary}"
    return body, content_type


def poll_until(fetcher, *, accepted: set[str], timeout: int, interval: float, label: str) -> dict:
    deadline = time.time() + timeout
    last_payload: dict | None = None

    while time.time() < deadline:
        payload = fetcher()
        last_payload = payload
        data = payload.get("data", {})
        status = data.get("status") or data.get("runner_status")
        if status in accepted:
            return payload
        if status in {"failed", "error"}:
            raise RuntimeError(f"{label} failed: {json.dumps(payload, ensure_ascii=False)}")
        time.sleep(interval)

    raise TimeoutError(f"{label} did not reach {sorted(accepted)} before timeout. last={last_payload}")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", default="http://127.0.0.1:3000", help="Gateway base URL")
    parser.add_argument("--project-name", default="Hybrid benchmark smoke", help="Project name for ontology generation")
    parser.add_argument("--timeout-seconds", type=int, default=900)
    args = parser.parse_args()

    seed_path = SEED_PATH
    if not seed_path.exists():
        raise FileNotFoundError(f"missing benchmark seed: {seed_path}")

    base_url = args.base_url.rstrip("/")
    simulation_requirement = DEFAULT_REQUIREMENT

    multipart_body, content_type = encode_multipart(
        {
            "project_name": args.project_name,
            "simulation_requirement": simulation_requirement,
        },
        {"files": seed_path},
    )

    ontology = request_json(
        f"{base_url}/api/graph/ontology/generate",
        method="POST",
        body=multipart_body,
        headers={"Content-Type": content_type},
    )
    project_id = ontology["data"]["project_id"]

    ontology_task_id = ontology["data"].get("task_id")
    if ontology_task_id:
        poll_until(
            lambda: request_json(f"{base_url}/api/graph/task/{ontology_task_id}"),
            accepted={"completed"},
            timeout=args.timeout_seconds,
            interval=2,
            label="ontology task",
        )
    elif not ontology["data"].get("ontology"):
        raise RuntimeError("ontology generation succeeded without task_id or ontology payload")

    graph_build = request_json(
        f"{base_url}/api/graph/build",
        method="POST",
        body=json.dumps({"project_id": project_id}).encode("utf-8"),
        headers={"Content-Type": "application/json"},
    )
    graph_task_id = graph_build["data"]["task_id"]

    graph_task = poll_until(
        lambda: request_json(f"{base_url}/api/graph/task/{graph_task_id}"),
        accepted={"completed"},
        timeout=args.timeout_seconds,
        interval=2,
        label="graph build task",
    )

    graph_id = graph_task["data"].get("graph_id")
    if not graph_id:
        project_info = request_json(f"{base_url}/api/graph/project/{project_id}")
        graph_id = project_info["data"].get("graph_id")
    if not graph_id:
        raise RuntimeError("graph build completed without graph_id")

    simulation_create = request_json(
        f"{base_url}/api/simulation/create",
        method="POST",
        body=json.dumps({"project_id": project_id, "graph_id": graph_id}).encode("utf-8"),
        headers={"Content-Type": "application/json"},
    )
    simulation_id = simulation_create["data"]["simulation_id"]

    prepare = request_json(
        f"{base_url}/api/simulation/prepare",
        method="POST",
        body=json.dumps(
            {
                "simulation_id": simulation_id,
                "graph_id": graph_id,
                "use_llm_for_profiles": False,
            }
        ).encode("utf-8"),
        headers={"Content-Type": "application/json"},
    )
    prepare_task_id = prepare["data"]["task_id"]

    poll_until(
        lambda: request_json(
            f"{base_url}/api/simulation/prepare/status",
            method="POST",
            body=json.dumps({"task_id": prepare_task_id, "simulation_id": simulation_id}).encode("utf-8"),
            headers={"Content-Type": "application/json"},
        ),
        accepted={"ready", "completed"},
        timeout=args.timeout_seconds,
        interval=3,
        label="simulation prepare",
    )

    request_json(
        f"{base_url}/api/simulation/run",
        method="POST",
        body=json.dumps({"simulation_id": simulation_id, "max_rounds": 3}).encode("utf-8"),
        headers={"Content-Type": "application/json"},
    )

    simulation_status = poll_until(
        lambda: request_json(f"{base_url}/api/simulation/{simulation_id}/status"),
        accepted={"completed", "stopped"},
        timeout=args.timeout_seconds,
        interval=5,
        label="simulation run",
    )

    report_generate = request_json(
        f"{base_url}/api/report/generate",
        method="POST",
        body=json.dumps({"simulation_id": simulation_id}).encode("utf-8"),
        headers={"Content-Type": "application/json"},
    )
    report_id = report_generate["data"]["report_id"]
    report_status = poll_until(
        lambda: request_json(f"{base_url}/api/report/generate/status?report_id={urllib.parse.quote(report_id)}"),
        accepted={"completed"},
        timeout=args.timeout_seconds,
        interval=5,
        label="report generation",
    )

    report = request_json(f"{base_url}/api/report/{report_id}")

    summary = {
        "project_id": project_id,
        "graph_id": graph_id,
        "simulation_id": simulation_id,
        "report_id": report_id,
        "simulation_status": simulation_status["data"].get("runner_status"),
        "report_status": report_status["data"].get("status"),
        "report_non_empty": bool(report.get("data", {}).get("content")),
    }
    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except (AssertionError, FileNotFoundError, RuntimeError, TimeoutError, urllib.error.URLError) as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        raise SystemExit(1)
