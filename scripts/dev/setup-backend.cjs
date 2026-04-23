const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const root = path.resolve(__dirname, "..", "..");
const backendDir = path.join(root, "backend");
const isWin = process.platform === "win32";

/** camel-oasis (and other deps) only publish wheels for 3.10.x–3.11.x today; 3.12+ fails to resolve. */
const VERSION_CHECK = [
  "-c",
  "import sys; sys.exit(0 if (3, 10) <= sys.version_info[:2] < (3, 12) else 1)",
];

const INSTALL_HINT =
  "Install Python 3.11 (64-bit) and ensure it is on PATH. On Windows, use the official installer with the \"py\" launcher, then run: py -3.11 -m venv backend\\.venv (from the repo root, after removing a broken venv if needed). See docs/getting-started/installation.md#python-uv-and-venv.";

function commandExists(command) {
  const checker = isWin ? "where" : "which";
  const result = spawnSync(checker, [command], { stdio: "ignore" });
  return result.status === 0;
}

function run(command, args, cwd = backendDir) {
  const result = spawnSync(command, args, {
    cwd,
    stdio: "inherit",
  });
  if (result.error) {
    console.error(`Failed to run ${command}: ${result.error.message}`);
    process.exit(1);
  }
  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}

function pythonCompatible(command, commandPrefix = []) {
  const result = spawnSync(command, [...commandPrefix, ...VERSION_CHECK], {
    stdio: "ignore",
  });
  return result.status === 0;
}

/**
 * @returns {{ cmd: string, prefix: string[] } | null}
 */
function findSystemPython() {
  if (isWin && commandExists("py")) {
    for (const ver of ["-3.11", "-3.10"]) {
      if (pythonCompatible("py", [ver])) {
        return { cmd: "py", prefix: [ver] };
      }
    }
  }

  const names = ["python3.11", "python3.10", "python3", "python"];
  for (const name of names) {
    if (!commandExists(name)) continue;
    if (pythonCompatible(name, [])) {
      return { cmd: name, prefix: [] };
    }
  }
  return null;
}

function venvPythonPath() {
  return isWin
    ? path.join(backendDir, ".venv", "Scripts", "python.exe")
    : path.join(backendDir, ".venv", "bin", "python");
}

if (commandExists("uv")) {
  run("uv", ["sync"], backendDir);
  process.exit(0);
}

const venvDir = path.join(backendDir, ".venv");
const venvPython = venvPythonPath();

if (fs.existsSync(venvPython)) {
  if (!pythonCompatible(venvPython, [])) {
    console.error(
      "backend/.venv was created with Python 3.12+, but this project needs Python 3.10 or 3.11 (camel-oasis and related wheels do not support 3.12 yet).\n" +
        "Delete the folder backend/.venv, then run npm run setup:backend again.\n" +
        INSTALL_HINT
    );
    process.exit(1);
  }
} else {
  if (fs.existsSync(venvDir)) {
    console.error(
      "backend/.venv exists but no usable interpreter was found. Remove the folder backend/.venv, then run npm run setup:backend again."
    );
    process.exit(1);
  }
  const resolved = findSystemPython();
  if (!resolved) {
    console.error("No compatible Python 3.10/3.11 found on PATH.\n" + INSTALL_HINT);
    process.exit(1);
  }
  run(resolved.cmd, [...resolved.prefix, "-m", "venv", ".venv"]);
}

if (!fs.existsSync(venvPython)) {
  console.error("Expected venv at " + venvPython + " but it is missing.\n" + INSTALL_HINT);
  process.exit(1);
}

run(venvPython, ["-m", "pip", "install", "--upgrade", "pip"]);
run(venvPython, ["-m", "pip", "install", "-r", "requirements.txt"]);
