const { spawn, spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const root = path.resolve(__dirname, "..", "..");
const gatewayDir = path.join(root, "gateway");
const isWin = process.platform === "win32";

function loadEnvFile(filePath) {
  const out = {};
  if (!fs.existsSync(filePath)) {
    return out;
  }
  const text = fs.readFileSync(filePath, "utf8");
  for (const line of text.split("\n")) {
    const t = line.trim();
    if (!t || t.startsWith("#")) {
      continue;
    }
    const idx = t.indexOf("=");
    if (idx === -1) {
      continue;
    }
    const key = t.slice(0, idx).trim();
    let val = t.slice(idx + 1).trim();
    if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) {
      val = val.slice(1, -1);
    }
    if (key) {
      out[key] = val;
    }
  }
  return out;
}

function commandExists(command) {
  const checker = isWin ? "where" : "which";
  const result = spawnSync(checker, [command], { stdio: "ignore" });
  return result.status === 0;
}

function spawnGateway(command, args) {
  const fileEnv = loadEnvFile(path.join(root, ".env"));
  const child = spawn(command, args, {
    cwd: root,
    stdio: "inherit",
    env: {
      ...fileEnv,
      ...process.env,
      GATEWAY_BIND_HOST: process.env.GATEWAY_BIND_HOST || "127.0.0.1",
      GATEWAY_PORT: process.env.GATEWAY_PORT || "3000",
      FRONTEND_DEV_URL: process.env.FRONTEND_DEV_URL || "http://127.0.0.1:5173",
    },
  });

  child.on("exit", (code, signal) => {
    if (signal) process.kill(process.pid, signal);
    process.exit(code ?? 0);
  });
}

if (!commandExists("go")) {
  console.error("Go is required to run the dev gateway. Install Go 1.21+ and try again.");
  process.exit(1);
}

spawnGateway("go", ["-C", gatewayDir, "run", "./cmd/mirofish-gateway"]);
