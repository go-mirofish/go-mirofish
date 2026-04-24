#!/usr/bin/env python3
"""
Local runtime validation: boot timings, health smoke, full hybrid API E2E,
endpoint benchmark (p50/p95/p99), concurrent stress, optional Playwright UI checks.

Modes are explicit and output to separate JSON files under
benchmark/results/runtime-sprint/<UTC>/ so static vs runtime is never conflated.

Usage (repo root):
  python scripts/hybrid/runtime_sprint.py --mode all
  python scripts/hybrid/runtime_sprint.py --mode static
  python scripts/hybrid/runtime_sprint.py --mode smoke --no-stack
  python scripts/hybrid/runtime_sprint.py --mode e2e-api --no-frontend
"""
from __future__ import annotations

import argparse
import concurrent.futures
import json
import os
import pathlib
import shutil
import signal
import socket
import subprocess
import sys
import threading
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import asdict, dataclass, field
from datetime import datetime, timezone
from typing import Any

ROOT = pathlib.Path(__file__).resolve().parents[2]

BACKEND_HEALTH = "http://127.0.0.1:5001/health"
VITE_PORT_DEFAULT = "3000"
VITE_BASE = f"http://127.0.0.1:{VITE_PORT_DEFAULT}"


def pick_free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        s.bind(("127.0.0.1", 0))
        return int(s.getsockname()[1])


@dataclass
class PopenInfo:
    popen: subprocess.Popen[Any]
    name: str
    log: pathlib.Path | None = None


@dataclass
class SprintState:
    mode: str
    out_dir: pathlib.Path
    children: list[PopenInfo] = field(default_factory=list)
    blocker: str | None = None
    gateway_base: str = "http://127.0.0.1:3001"


def utc_stamp() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H%M%SZ")


def load_dotenv_file(path: pathlib.Path) -> None:
    """Load repo .env, overriding process env so a populated .env wins over empty host vars."""
    if not path.is_file():
        return
    for raw in path.read_text(encoding="utf-8", errors="replace").splitlines():
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        key, value = key.strip(), value.strip()
        if value.startswith('"') and value.endswith('"'):
            value = value[1:-1]
        if value.startswith("'") and value.endswith("'"):
            value = value[1:-1]
        if key:
            os.environ[key] = value


def preflight_blockers() -> list[str]:
    issues: list[str] = []
    if not (os.environ.get("LLM_API_KEY") or "").strip():
        issues.append("LLM_API_KEY is not set (required by backend Config.validate; set in repo root .env)")
    if not (os.environ.get("ZEP_API_KEY") or "").strip():
        issues.append("ZEP_API_KEY is not set (required by backend Config.validate; set in repo root .env)")
    return issues


def default_backend_python() -> str | None:
    win = sys.platform == "win32"
    cands = [
        ROOT / "backend" / ".venv" / "Scripts" / "python.exe",
        ROOT / "backend" / ".venv" / "bin" / "python",
    ]
    for p in cands:
        if p.is_file():
            return str(p)
    return None


def kill_tree(pid: int) -> None:
    if pid <= 0:
        return
    if sys.platform == "win32":
        subprocess.run(
            ["taskkill", "/PID", str(pid), "/T", "/F"],
            capture_output=True,
            text=True,
        )
    else:
        try:
            os.killpg(os.getpgid(pid), signal.SIGTERM)
        except (ProcessLookupError, PermissionError, OSError):
            try:
                os.kill(pid, signal.SIGTERM)
            except OSError:
                pass


def popen_with_group(
    args: list[str], *, cwd: pathlib.Path, env: dict[str, str], log_path: pathlib.Path, name: str
) -> PopenInfo:
    log_path.parent.mkdir(parents=True, exist_ok=True)
    fh = open(log_path, "w", encoding="utf-8", errors="replace")
    kw: dict[str, Any] = {
        "args": args,
        "cwd": str(cwd),
        "env": env,
        "stdout": fh,
        "stderr": subprocess.STDOUT,
    }
    if sys.platform == "win32":
        kw["creationflags"] = subprocess.CREATE_NEW_PROCESS_GROUP
    else:
        kw["start_new_session"] = True
    p = subprocess.Popen(**kw)
    return PopenInfo(popen=p, name=name, log=log_path)


def wait_url(url: str, label: str, timeout_s: float = 120.0, interval: float = 0.5) -> tuple[bool, str | None]:
    deadline = time.time() + timeout_s
    last_err: str | None = None
    while time.time() < deadline:
        try:
            with urllib.request.urlopen(url, timeout=3) as r:
                r.read(64)
            return True, None
        except Exception as e:
            last_err = repr(e)
        time.sleep(interval)
    return False, f"timeout waiting for {label}: {url} last={last_err}"


def http_get_json(url: str, timeout: float = 30.0) -> Any:
    req = urllib.request.Request(url, method="GET")
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.loads(resp.read().decode("utf-8"))


def http_get_ms(url: str, timeout: float = 30.0) -> dict[str, Any]:
    t0 = time.perf_counter()
    try:
        with urllib.request.urlopen(url, timeout=timeout) as resp:
            n = len(resp.read())
        return {"ok": True, "ms": (time.perf_counter() - t0) * 1000, "bytes": n}
    except Exception as e:
        return {"ok": False, "ms": (time.perf_counter() - t0) * 1000, "error": repr(e)}


def percentile_nearest_sorted(sorted_vals: list[float], p: float) -> float | None:
    if not sorted_vals:
        return None
    if len(sorted_vals) == 1:
        return round(sorted_vals[0], 3)
    k = (len(sorted_vals) - 1) * p / 100.0
    f = int(k)
    c = k - f
    if f + 1 < len(sorted_vals):
        return round(sorted_vals[f] * (1 - c) + sorted_vals[f + 1] * c, 3)
    return round(sorted_vals[f], 3)


def bench_endpoint(url: str, samples: int = 30) -> dict[str, Any]:
    times: list[float] = []
    errors: list[str] = []
    for _ in range(samples):
        r = http_get_ms(url, timeout=60.0)
        if r.get("ok"):
            times.append(float(r["ms"]))
        else:
            errors.append(r.get("error", "?"))
    times.sort()
    return {
        "url": url,
        "samples": samples,
        "ok_count": len(times),
        "error_count": len(errors),
        "errors_sample": errors[:5],
        "latency_ms": {
            "min": times[0] if times else None,
            "p50": percentile_nearest_sorted(times, 50),
            "p95": percentile_nearest_sorted(times, 95),
            "p99": percentile_nearest_sorted(times, 99),
            "max": times[-1] if times else None,
        },
    }


def stress_mixed(
    url_weights: list[tuple[str, float]], total_requests: int, workers: int
) -> dict[str, Any]:
    lock = threading.Lock()
    n = 0
    results: list[dict[str, Any]] = []
    wsum = sum(w for _, w in url_weights)
    urls = [u for u, w in url_weights for _ in range(max(1, int(w * 10 / wsum)))]

    def pick() -> str:
        nonlocal n
        with lock:
            u = urls[n % len(urls)]
            n += 1
            return u

    def one() -> dict[str, Any]:
        u = pick()
        return {**http_get_ms(u, timeout=45.0), "url": u}

    oks: list[float] = []
    failures: list[dict[str, Any]] = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as ex:
        futs = [ex.submit(one) for _ in range(total_requests)]
        for f in concurrent.futures.as_completed(futs):
            r = f.result()
            results.append(r)
            if r.get("ok"):
                oks.append(float(r["ms"]))
            else:
                failures.append(r)
    oks.sort()
    return {
        "request_count": total_requests,
        "success_count": len(oks),
        "failure_count": len(failures),
        "workers": workers,
        "latency_ms": {
            "min": oks[0] if oks else None,
            "p50": percentile_nearest_sorted(oks, 50),
            "p95": percentile_nearest_sorted(oks, 95),
            "p99": percentile_nearest_sorted(oks, 99),
            "max": oks[-1] if oks else None,
        },
        "failures": failures[:20],
    }


def run_static_checks() -> dict[str, Any]:
    res: dict[str, Any] = {"mode": "static", "ok": True, "checks": []}
    # Compile all gateway packages (go test ./... is platform-sensitive on Windows; sprint uses build proof).
    gw = ["go", "build", "./..."]
    r = subprocess.run(gw, cwd=ROOT / "gateway", capture_output=True, text=True, timeout=600)
    res["checks"].append(
        {
            "name": "go_build_gateway",
            "command": gw,
            "cwd": "gateway",
            "returncode": r.returncode,
            "ok": r.returncode == 0,
            "tail_stderr": (r.stderr or "")[-8000:],
            "tail_stdout": (r.stdout or "")[-2000:],
        }
    )
    if r.returncode != 0:
        res["ok"] = False
    r2 = subprocess.run(
        [sys.executable, "-m", "compileall", "-q", str(ROOT / "scripts" / "hybrid")],
        capture_output=True,
        text=True,
        timeout=120,
    )
    res["checks"].append(
        {
            "name": "python_compileall_hybrid_scripts",
            "returncode": r2.returncode,
            "ok": r2.returncode == 0,
            "stderr": (r2.stderr or "")[-4000:],
        }
    )
    if r2.returncode != 0:
        res["ok"] = False
    return res


def run_e2e_api(base: str, out: pathlib.Path, timeout: int) -> dict[str, Any]:
    script = ROOT / "scripts" / "hybrid" / "run_benchmark_smoke.py"
    log_path = out / "e2e-api-raw.log"
    started = time.perf_counter()
    p = subprocess.run(
        [sys.executable, str(script), "--base-url", base, "--timeout-seconds", str(timeout)],
        capture_output=True,
        text=True,
        timeout=timeout + 120,
    )
    duration = time.perf_counter() - started
    log_path.write_text((p.stdout or "") + "\n" + (p.stderr or ""), encoding="utf-8")
    summary: Any = None
    if p.stdout and p.stdout.strip().startswith("{"):
        try:
            summary = json.loads(p.stdout.strip())
        except json.JSONDecodeError:
            summary = {"parse_error": True, "raw": p.stdout[:2000]}
    return {
        "mode": "e2e_api",
        "ok": p.returncode == 0,
        "returncode": p.returncode,
        "duration_seconds": round(duration, 2),
        "summary": summary,
        "log_path": str(log_path.relative_to(ROOT)) if out.is_relative_to(ROOT) else str(log_path),
    }


def http_post_json(url: str, body: dict[str, Any], timeout: float = 30.0) -> dict[str, Any]:
    data = json.dumps(body).encode("utf-8")
    req = urllib.request.Request(
        url, data=data, method="POST", headers={"Content-Type": "application/json"}
    )
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        return {
            "http_error": e.code,
            "body": e.read().decode("utf-8", errors="replace")[:2000],
        }


def http_delete(url: str, timeout: float = 30.0) -> dict[str, Any]:
    req = urllib.request.Request(url, method="DELETE")
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        return {
            "http_error": e.code,
            "body": e.read().decode("utf-8", errors="replace")[:2000],
        }


def run_delete_control_probes(
    base: str, summary: dict[str, Any] | None
) -> dict[str, Any]:
    if not summary:
        return {"skipped": True, "reason": "no e2e summary"}
    sim_id = summary.get("simulation_id")
    graph_id = summary.get("graph_id")
    out: dict[str, Any] = {"simulation_delete": None, "graph_delete": None}
    if sim_id:
        try:
            out["simulation_delete"] = http_post_json(
                f"{base}/api/simulation/delete",
                {"simulation_id": sim_id},
            )
        except Exception as e:
            out["simulation_delete"] = {"error": repr(e)}
    if graph_id:
        try:
            gi = urllib.parse.quote(graph_id, safe="")
            out["graph_delete"] = http_delete(f"{base}/api/graph/delete/{gi}")
        except Exception as e:
            out["graph_delete"] = {"error": repr(e)}
    return out


def run_playwright_sprint(frontend_url: str, out: pathlib.Path) -> dict[str, Any]:
    node = shutil.which("node")
    cjs = ROOT / "scripts" / "hybrid" / "frontend_sprint.cjs"
    if not node or not cjs.is_file():
        return {"mode": "frontend_e2e", "ok": False, "error": "node or frontend_sprint.cjs missing"}
    art = out / "frontend-e2e-raw.json"
    env = os.environ.copy()
    env["FRONTEND_URL"] = frontend_url
    env["ARTIFACT"] = str(art)
    p = subprocess.run(
        [node, str(cjs)],
        env=env,
        capture_output=True,
        text=True,
        timeout=300,
    )
    raw: dict[str, Any] = {"stdout": p.stdout, "stderr": (p.stderr or "")[-2000:]}
    art_obj: dict[str, Any] | None = None
    if art.is_file():
        try:
            art_obj = json.loads(art.read_text(encoding="utf-8"))
            raw["artifact"] = art_obj
        except json.JSONDecodeError:
            raw["artifact_parse_error"] = art.read_text(encoding="utf-8")[:2000]
    if art_obj is None and (p.stdout or "").strip().startswith("{"):
        try:
            art_obj = json.loads(p.stdout.strip())
            raw["artifact"] = art_obj
        except json.JSONDecodeError:
            pass
    ok = (
        p.returncode == 0
        and isinstance(art_obj, dict)
        and art_obj.get("ok") is True
        and not art_obj.get("fatal")
    )
    return {"mode": "frontend_e2e", "ok": bool(ok), "returncode": p.returncode, "detail": raw}


def start_stack(
    st: SprintState, *, with_frontend: bool, build_gateway_bin: bool
) -> dict[str, Any]:
    # with_frontend is True when Playwright UI sprint is in the same run (Vite on 3000, gateway 3001).
    out = st.out_dir
    err = preflight_blockers()
    if err:
        st.blocker = "; ".join(err)
        return {"boot": None, "blocker": st.blocker}

    py = default_backend_python()
    if not py:
        st.blocker = "backend venv not found; run npm run setup:backend"
        return {"boot": None, "blocker": st.blocker}

    if build_gateway_bin:
        b = subprocess.run(
            ["node", str(ROOT / "scripts" / "dev" / "build-gateway.cjs")],
            cwd=ROOT,
            capture_output=True,
            text=True,
        )
        if b.returncode != 0:
            st.blocker = "gateway build failed: " + (b.stderr or b.stdout or "")[:2000]
            return {"boot": None, "blocker": st.blocker}
    is_win = sys.platform == "win32"
    gexe = ROOT / "gateway" / "bin" / ("go-mirofish-gateway.exe" if is_win else "go-mirofish-gateway")
    if not gexe.is_file():
        st.blocker = f"gateway binary missing at {gexe} after build"
        return {"boot": None, "blocker": st.blocker}

    env_backend = os.environ.copy()
    env_backend.setdefault("FLASK_HOST", "127.0.0.1")
    env_backend.setdefault("FLASK_PORT", "5001")
    t_backend = time.perf_counter()
    st.children.append(
        popen_with_group(
            [py, "run.py"],
            cwd=ROOT / "backend",
            env=env_backend,
            log_path=out / "process-backend.log",
            name="backend",
        )
    )
    ok_b, err_b = wait_url(BACKEND_HEALTH, "backend", 180)
    backend_boot_ms = (time.perf_counter() - t_backend) * 1000
    if not ok_b:
        st.blocker = err_b or "backend failed"
        return {
            "boot": {"backend_boot_ms": round(backend_boot_ms, 2), "backend_reachable": False},
            "blocker": st.blocker,
        }

    boot: dict[str, Any] = {
        "backend_boot_ms": round(backend_boot_ms, 2),
        "backend_reachable": True,
    }

    gw_port = pick_free_port()
    st.gateway_base = f"http://127.0.0.1:{gw_port}"
    boot["gateway_chosen_port"] = gw_port

    if with_frontend:
        env_fe = os.environ.copy()
        env_fe["CI"] = "1"
        env_fe["BROWSER"] = "none"
        env_fe["VITE_GATEWAY_PROXY_TARGET"] = st.gateway_base
        t_fe = time.perf_counter()
        fe_args = [shutil.which("npm") or "npm", "run", "dev", "--prefix", "frontend"]
        st.children.append(
            popen_with_group(fe_args, cwd=ROOT, env=env_fe, log_path=out / "process-frontend.log", name="frontend")
        )
        ok_f, err_f = wait_url(VITE_BASE + "/", "vite", 300)
        boot["frontend_boot_ms"] = round((time.perf_counter() - t_fe) * 1000, 2)
        boot["frontend_reachable"] = ok_f
        if not ok_f:
            st.blocker = err_f or "frontend failed"
            return {"boot": boot, "blocker": st.blocker, "gateway_base": st.gateway_base}

    env_gw = os.environ.copy()
    env_gw["BACKEND_URL"] = "http://127.0.0.1:5001"
    env_gw["GATEWAY_PORT"] = str(gw_port)
    env_gw["GATEWAY_BIND_HOST"] = "127.0.0.1"
    if with_frontend:
        env_gw["FRONTEND_DEV_URL"] = VITE_BASE
    else:
        env_gw.pop("FRONTEND_DEV_URL", None)
        fe_dist = ROOT / "frontend" / "dist"
        if (fe_dist / "index.html").is_file():
            env_gw["FRONTEND_DIST_DIR"] = str(fe_dist)

    for reqk in ("LLM_BASE_URL", "LLM_API_KEY", "LLM_MODEL_NAME"):
        if not (str(env_gw.get(reqk, "")).strip()):
            st.blocker = f"gateway env missing {reqk} after loading .env (required for Go ontology); check repo root .env and shell overrides"
            return {"boot": boot, "blocker": st.blocker, "gateway_base": st.gateway_base}
    t_gw = time.perf_counter()
    ginfo = popen_with_group(
        [str(gexe)],
        cwd=ROOT,
        env=env_gw,
        log_path=out / "process-gateway.log",
        name="gateway",
    )
    st.children.append(ginfo)
    time.sleep(0.3)
    if ginfo.popen.poll() is not None:
        tail = (out / "process-gateway.log").read_text(encoding="utf-8", errors="replace")[-4000:]
        st.blocker = f"gateway process exited before health (e.g. port bind). log tail: {tail}"
        return {"boot": boot, "blocker": st.blocker, "gateway_base": st.gateway_base}
    gh = f"{st.gateway_base}/health"
    ok_g, err_g = wait_url(gh, "gateway", 120)
    boot["gateway_boot_ms"] = round((time.perf_counter() - t_gw) * 1000, 2)
    boot["gateway_reachable"] = ok_g
    if not ok_g:
        st.blocker = err_g or "gateway failed"
        return {"boot": boot, "blocker": st.blocker, "gateway_base": st.gateway_base}
    return {"boot": boot, "blocker": None, "gateway_base": st.gateway_base}


def health_snapshot(gateway_base: str) -> dict[str, Any]:
    snap: dict[str, Any] = {}
    gbase = (gateway_base or "http://127.0.0.1:3001").rstrip("/")
    try:
        with urllib.request.urlopen(BACKEND_HEALTH, timeout=5) as r:
            snap["backend_health"] = json.loads(r.read().decode("utf-8"))
    except Exception as e:
        snap["backend_health"] = {"error": repr(e)}
    try:
        with urllib.request.urlopen(f"{gbase}/health", timeout=5) as r:
            snap["gateway_health"] = json.loads(r.read().decode("utf-8"))
    except Exception as e:
        snap["gateway_health"] = {"error": repr(e)}
    return snap


def build_aggregate(
    st: SprintState,
    static_res: dict[str, Any] | None,
    boot_info: dict[str, Any] | None,
    e2e: dict[str, Any] | None,
    be: dict[str, Any] | None,
    stres: dict[str, Any] | None,
    fe: dict[str, Any] | None,
    flowx: dict[str, Any] | None,
    child_pids: dict[str, int] | None,
    captured_hs: dict[str, Any] | None,
) -> dict[str, Any]:
    d = st.out_dir
    now = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
    hs = captured_hs or {}
    ag: dict[str, Any] = {
        "captured_at": now,
        "sprint": {
            "id": d.name,
            "requested_mode": st.mode,
            "output_dir": str(d.relative_to(ROOT) if d.is_relative_to(ROOT) else d),
            "gateway_url": st.gateway_base,
        },
        "release": os.environ.get("MIROFISH_RELEASE", "v0.1.0"),
        "host": {
            "platform": sys.platform,
            "cpu_count": os.cpu_count(),
        },
        "static": static_res,
    }
    if "backend_health" in hs:
        ag["backend_health"] = hs["backend_health"]
    if "gateway_health" in hs:
        ag["gateway_health"] = hs["gateway_health"]
    if boot_info and boot_info.get("boot"):
        ag["boot_timings"] = boot_info["boot"]
    if e2e:
        ag["benchmark"] = {
            "ok": e2e.get("ok"),
            "duration_seconds": e2e.get("duration_seconds"),
            "returncode": e2e.get("returncode"),
            "summary": (e2e.get("summary") or {}),
            "stderr": "",
        }
        if not e2e.get("ok"):
            lp = e2e.get("log_path")
            if lp:
                try:
                    logp = pathlib.Path(str(lp).replace("\\", os.sep))
                    if not logp.is_file():
                        logp = ROOT / logp
                    if logp.is_file():
                        ag["benchmark"]["e2e_failure_excerpt"] = logp.read_text(
                            encoding="utf-8", errors="replace"
                        )[-4000:]
                except OSError:
                    pass
    if stres and stres.get("request_count") is not None:
        ag["stress"] = {
            "request_count": stres["request_count"],
            "success_count": stres.get("success_count"),
            "failure_count": stres.get("failure_count"),
            "latency_ms": (stres.get("latency_ms") or {}),
            "failures": stres.get("failures", [])[:20],
        }
    if be:
        ag["endpoint_benchmark"] = be
    if flowx is not None:
        ag["control_delete"] = flowx
    if child_pids:
        ag["processes"] = {name: {"pid": pid} for name, pid in child_pids.items()}

    stress_ok = bool(stres and stres.get("request_count", 0) > 0 and stres.get("failure_count", 0) == 0)
    e2e_ok = bool(e2e and e2e.get("ok"))
    fe_ok = bool(fe and fe.get("ok"))

    fb = (boot_info or {}).get("boot") or {}
    fe_vis = fb.get("frontend_reachable")
    if "frontend_reachable" in fb:
        fe_pass = fe_vis is True
    else:
        fe_pass = None
    ag["proof"] = {
        "backend_boot": bool(boot_info and (boot_info.get("boot") or {}).get("backend_reachable") is True),
        "gateway_boot": bool(boot_info and (boot_info.get("boot") or {}).get("gateway_reachable") is True),
        "frontend_boot": fe_pass,
        "bounded_stress": stress_ok,
        "full_flow": e2e_ok,
        "frontend_e2e": fe_ok if fe is not None else None,
    }
    if st.blocker:
        ag["sprint_blocker"] = st.blocker
    return ag


def cleanup_stack(st: SprintState) -> None:
    for ch in reversed(st.children):
        try:
            if ch.popen.poll() is None:
                kill_tree(ch.popen.pid)
        except Exception:
            pass
    st.children.clear()


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument(
        "--mode",
        default="all",
        help="all | static | smoke | boot | e2e-api | bench | stress | frontend-e2e (comma-separated)",
    )
    ap.add_argument("--out-subdir", default=None, help="default: auto UTC folder under runtime-sprint")
    ap.add_argument("--e2e-timeout", type=int, default=1200, help="seconds for run_benchmark_smoke")
    ap.add_argument("--no-frontend", action="store_true", help="gateway serves dist if built; no Vite")
    ap.add_argument("--no-stack", action="store_true", help="only run modes that do not start processes")
    ap.add_argument("--skip-gateway-build", action="store_true")
    ap.add_argument(
        "--export-live",
        action="store_true",
        help="write benchmark/results/v0.1.0-live-benchmark.json (default: only when --mode all)",
    )
    args = ap.parse_args()
    load_dotenv_file(ROOT / ".env")

    modes = [m.strip() for m in args.mode.split(",") if m.strip()]
    if not modes:
        modes = ["all"]
    if "all" in modes:
        modes = [
            "static",
            "smoke",
            "boot",
            "e2e-api",
            "bench",
            "stress",
            "frontend-e2e",
            "control-delete",
        ]

    out_name = args.out_subdir or utc_stamp()
    st = SprintState(mode=args.mode, out_dir=ROOT / "benchmark" / "results" / "runtime-sprint" / out_name)
    st.out_dir.mkdir(parents=True, exist_ok=True)

    static_res: dict[str, Any] | None = None
    boot_data: dict[str, Any] | None = None
    e2e: dict[str, Any] | None = None
    be: dict[str, Any] | None = None
    stres: dict[str, Any] | None = None
    fe: dict[str, Any] | None = None
    flowx: dict[str, Any] | None = None
    child_pids: dict[str, int] = {}
    captured_hs: dict[str, Any] = {}

    if "static" in modes:
        static_res = run_static_checks()
        (st.out_dir / "static-checks.json").write_text(
            json.dumps(static_res, indent=2) + "\n", encoding="utf-8"
        )
        if not static_res.get("ok"):
            (st.out_dir / "sprint-failure.txt").write_text("static checks failed", encoding="utf-8")

    need_stack = bool(
        set(modes)
        & {
            "smoke",
            "boot",
            "e2e-api",
            "bench",
            "stress",
            "frontend-e2e",
            "control-delete",
        }
    )
    if need_stack and not args.no_stack:
        with_fe = (not args.no_frontend) and ("frontend-e2e" in modes)
        boot_data = start_stack(
            st,
            with_frontend=with_fe,
            build_gateway_bin=not args.skip_gateway_build,
        )
        (st.out_dir / "boot.json").write_text(
            json.dumps(
                {**(boot_data or {}), "blocker": st.blocker},
                indent=2,
            )
            + "\n",
            encoding="utf-8",
        )
        if st.blocker:
            (st.out_dir / "sprint-failure.txt").write_text(
                f"blocker: {st.blocker}\n", encoding="utf-8"
            )

    try:
        if not st.blocker and need_stack:
            gb = st.gateway_base
            if "smoke" in modes or "boot" in modes:
                snap = health_snapshot(gb)
                (st.out_dir / "smoke-health.json").write_text(
                    json.dumps(snap, indent=2) + "\n", encoding="utf-8"
                )
            if "e2e-api" in modes:
                e2e = run_e2e_api(gb, st.out_dir, args.e2e_timeout)
                (st.out_dir / "e2e-api.json").write_text(
                    json.dumps(e2e, indent=2) + "\n", encoding="utf-8"
                )
            resource_urls: list[tuple[str, float]] = [
                (f"{gb}/health", 1.0),
                (f"{gb}/api/graph/project/list", 1.0),
            ]
            if e2e and e2e.get("summary") and isinstance(e2e["summary"], dict):
                s = e2e["summary"]
                sid = s.get("simulation_id")
                if sid:
                    resource_urls.append((f"{gb}/api/simulation/{sid}/status", 2.0))
                rid = s.get("report_id")
                if rid:
                    resource_urls.append((f"{gb}/api/report/{rid}", 1.5))
            if "bench" in modes:
                be = {"endpoints": [bench_endpoint(u, 25) for u, _ in resource_urls]}
                (st.out_dir / "endpoint-benchmark.json").write_text(
                    json.dumps(be, indent=2) + "\n", encoding="utf-8"
                )
            if "stress" in modes:
                stres = stress_mixed(resource_urls, total_requests=100, workers=20)
                (st.out_dir / "stress.json").write_text(
                    json.dumps(stres, indent=2) + "\n", encoding="utf-8"
                )
            if "frontend-e2e" in modes and not args.no_frontend:
                fe = run_playwright_sprint(VITE_BASE, st.out_dir)
                (st.out_dir / "frontend-e2e.json").write_text(
                    json.dumps(fe, indent=2) + "\n", encoding="utf-8"
                )
            if "control-delete" in modes and e2e and e2e.get("summary"):
                flowx = run_delete_control_probes(
                    gb, e2e["summary"] if isinstance(e2e["summary"], dict) else None
                )
                (st.out_dir / "control-delete.json").write_text(
                    json.dumps(flowx, indent=2) + "\n", encoding="utf-8"
                )
        if need_stack and not st.blocker and st.children:
            captured_hs = health_snapshot(st.gateway_base)
            child_pids = {c.name: c.popen.pid for c in st.children}
    finally:
        if need_stack and not args.no_stack and st.children:
            cleanup_stack(st)

    agg = build_aggregate(st, static_res, boot_data, e2e, be, stres, fe, flowx, child_pids, captured_hs)
    (st.out_dir / "full-report.json").write_text(json.dumps(agg, indent=2) + "\n", encoding="utf-8")
    if args.export_live or (args.mode or "").strip() == "all":
        live = ROOT / "benchmark" / "results" / "v0.1.0-live-benchmark.json"
        live.write_text(json.dumps(agg, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(agg, indent=2))
    return 0 if not st.blocker and (e2e is None or e2e.get("ok")) and (static_res is None or static_res.get("ok")) else 1


if __name__ == "__main__":
    raise SystemExit(main())
