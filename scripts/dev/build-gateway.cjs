const { spawnSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const root = path.resolve(__dirname, "..", "..");
const gatewayDir = path.join(root, "gateway");
const binDir = path.join(gatewayDir, "bin");
const isWin = process.platform === "win32";
const output = path.join(binDir, `go-mirofish-gateway${isWin ? ".exe" : ""}`);

fs.mkdirSync(binDir, { recursive: true });

const result = spawnSync(
  "go",
  ["build", "-o", output, "./cmd/mirofish-gateway"],
  {
    cwd: gatewayDir,
    stdio: "inherit",
    env: {
      ...process.env,
      GOCACHE: process.env.GOCACHE || "/tmp/go-build-cache",
    },
  }
);

if (result.error) {
  console.error(`Failed to run go build: ${result.error.message}`);
  process.exit(1);
}

process.exit(result.status ?? 0);
