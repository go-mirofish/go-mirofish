const { spawn, spawnSync } = require("node:child_process");
const path = require("node:path");

const root = path.resolve(__dirname, "..", "..");
const gatewayDir = path.join(root, "gateway");
const isWin = process.platform === "win32";

function commandExists(command) {
  const checker = isWin ? "where" : "which";
  const result = spawnSync(checker, [command], { stdio: "ignore" });
  return result.status === 0;
}

function spawnGateway(command, args) {
  const child = spawn(command, args, {
    cwd: gatewayDir,
    stdio: "inherit",
    env: {
      ...process.env,
      BACKEND_URL: process.env.BACKEND_URL || "http://127.0.0.1:5001",
      GATEWAY_BIND_HOST: process.env.GATEWAY_BIND_HOST || "127.0.0.1",
      GATEWAY_PORT: process.env.GATEWAY_PORT || "3001",
      FRONTEND_DEV_URL: process.env.FRONTEND_DEV_URL || "http://127.0.0.1:3000",
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

spawnGateway("go", ["run", "./cmd/mirofish-gateway"]);
