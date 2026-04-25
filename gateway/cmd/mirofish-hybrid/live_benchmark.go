package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

func cmdLiveBenchmark(repoRoot string, args []string) int {
	fs := flag.NewFlagSet("live-benchmark", flag.ExitOnError)
	benchOut := fs.String("bench-out", "", "benchmark JSON output path (default: benchmark/results/benchmarks/live-benchmark.json)")
	reportOut := fs.String("report-out", "", "markdown report path (default: docs/report/benchmark-report.md)")
	gatewayLog := fs.String("gateway-log", "", "gateway stdout/stderr log (default: benchmark/results/logs/gateway/gateway-live.log)")
	port := fs.String("port", "3000", "gateway port")
	benchHeavy := fs.Bool("heavy", false, "pass --heavy to the benchmark tool")
	skipNpm := fs.Bool("skip-frontend-build", false, "skip npm run build (use existing frontend/dist)")
	_ = fs.Parse(args)

	benchDir := filepath.Join(repoRoot, "benchmark", "results", "benchmarks")
	_ = os.MkdirAll(benchDir, 0o755)
	logDir := filepath.Join(repoRoot, "benchmark", "results", "logs", "gateway")
	_ = os.MkdirAll(logDir, 0o755)
	dataDirs := []string{"projects", "reports", "tasks", "simulations"}
	for _, d := range dataDirs {
		_ = os.MkdirAll(filepath.Join(repoRoot, "data", d), 0o755)
	}

	outPath := *benchOut
	if outPath == "" {
		outPath = filepath.Join(benchDir, "live-benchmark.json")
	}
	repPath := *reportOut
	if repPath == "" {
		repPath = filepath.Join(repoRoot, "docs", "report", "benchmark-report.md")
	}
	logPath := *gatewayLog
	if logPath == "" {
		logPath = filepath.Join(logDir, "gateway-live.log")
	}

	gatewayDir := filepath.Join(repoRoot, "gateway")
	exeName := "go-mirofish-gateway"
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}
	binPath := filepath.Join(gatewayDir, "bin", exeName)

	build := exec.Command("go", "build", "-o", binPath, "./cmd/mirofish-gateway")
	build.Dir = gatewayDir
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "live-benchmark: go build:", err)
		return 1
	}

	if !*skipNpm {
		npm := "npm"
		if runtime.GOOS == "windows" {
			npm = "npm.cmd"
		}
		fe := exec.Command(npm, "run", "build", "--prefix", "frontend")
		fe.Dir = repoRoot
		fe.Stdout = os.Stdout
		fe.Stderr = os.Stderr
		if err := fe.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "live-benchmark: frontend build:", err)
			return 1
		}
	}

	logFile, err := os.Create(logPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "live-benchmark:", err)
		return 1
	}
	defer logFile.Close()

	dist := filepath.Join(repoRoot, "frontend", "dist")
	gw := exec.CommandContext(context.Background(), binPath)
	gw.Env = append(os.Environ(),
		"FRONTEND_DIST_DIR="+dist,
		"GATEWAY_BIND_HOST=127.0.0.1",
		"GATEWAY_PORT="+*port,
		"PROJECTS_DIR="+filepath.Join(repoRoot, "data", "projects"),
		"REPORTS_DIR="+filepath.Join(repoRoot, "data", "reports"),
		"TASKS_DIR="+filepath.Join(repoRoot, "data", "tasks"),
		"SIMULATIONS_DIR="+filepath.Join(repoRoot, "data", "simulations"),
		"SCRIPTS_DIR="+filepath.Join(repoRoot, "scripts"),
	)
	gw.Stdout = io.MultiWriter(os.Stderr, logFile)
	gw.Stderr = gw.Stdout
	if err := gw.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "live-benchmark: start gateway:", err)
		return 1
	}
	defer func() {
		_ = gw.Process.Kill()
		_, _ = gw.Process.Wait()
	}()

	base := "http://127.0.0.1:" + *port
	if err := waitHealth(base + "/health", 60*time.Second); err != nil {
		fmt.Fprintln(os.Stderr, "live-benchmark:", err)
		return 1
	}

	bargs := []string{
		"run", "./cmd/benchmark",
		"--base-url", base,
		"--out", outPath,
		"--release", envOr("RELEASE", "dev"),
	}
	if *benchHeavy {
		bargs = append(bargs, "--heavy")
	}
	benc := exec.Command("go", bargs...)
	benc.Dir = gatewayDir
	benc.Stdout = os.Stdout
	benc.Stderr = os.Stderr
	if err := benc.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "live-benchmark: benchmark:", err)
		return 1
	}

	rep := exec.Command("go", "run", "./cmd/benchmark-report",
		"--input", outPath,
		"--output", repPath,
		"--title", "go-mirofish benchmark report",
	)
	rep.Dir = gatewayDir
	rep.Stdout = os.Stdout
	rep.Stderr = os.Stderr
	_ = rep.Run()

	fmt.Println("[live-benchmark] done. artifact:", outPath)
	fmt.Println("[live-benchmark] report:  ", repPath)
	return 0
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func waitHealth(url string, total time.Duration) error {
	deadline := time.Now().Add(total)
	c := &http.Client{Timeout: 3 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := c.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode < 500 {
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("gateway not ready: %s", url)
}
