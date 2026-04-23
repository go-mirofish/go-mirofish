const { spawn, spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const root = path.resolve(__dirname, "..", "..");
const backendDir = path.join(root, "backend");
const isWin = process.platform === "win32";

function existingFile(...segments) {
  const target = path.join(...segments);
  return fs.existsSync(target) ? target : null;
}

function commandExists(command) {
  const checker = isWin ? "where" : "which";
  const result = spawnSync(checker, [command], { stdio: "ignore" });
  return result.status === 0;
}

const venvPython =
  existingFile(backendDir, ".venv", "Scripts", "python.exe") ||
  existingFile(backendDir, ".venv", "bin", "python");

let child;

if (venvPython) {
  child = spawn(venvPython, ["run.py"], {
    cwd: backendDir,
    stdio: "inherit",
  });
} else if (commandExists("uv")) {
  child = spawn("uv", ["run", "python", "run.py"], {
    cwd: backendDir,
    stdio: "inherit",
  });
} else {
  console.error("Backend environment is not ready. Run `npm run setup:backend` first.");
  process.exit(1);
}

child.on("exit", (code, signal) => {
  if (signal) process.kill(process.pid, signal);
  process.exit(code ?? 0);
});
