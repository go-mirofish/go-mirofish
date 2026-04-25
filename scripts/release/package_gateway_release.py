#!/usr/bin/env python3
"""Local release packaging gate: run `go test ./...` in gateway/ (no wheel to build yet)."""
from __future__ import annotations

import os
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
GATEWAY = ROOT / "gateway"


def main() -> int:
    env = {**os.environ, "CGO_ENABLED": "0"}
    r = subprocess.run(
        ["go", "test", "./..."],
        cwd=str(GATEWAY),
        env=env,
    )
    return int(r.returncode)


if __name__ == "__main__":
    sys.exit(main())
