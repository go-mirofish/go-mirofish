package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

func cmdStressProbe(args []string) int {
	fs := flag.NewFlagSet("stress-probe", flag.ExitOnError)
	gatewayURL := fs.String("gateway-url", "http://127.0.0.1:3000/health", "URL to request")
	rounds := fs.Int("rounds", 40, "total requests")
	workers := fs.Int("workers", 16, "concurrent workers")
	_ = fs.Parse(args)
	return runStress(*gatewayURL, *rounds, *workers)
}

type stressSample struct {
	OK        bool    `json:"ok"`
	URL       string  `json:"url"`
	ElapsedMS float64 `json:"elapsed_ms"`
	Status    int     `json:"status,omitempty"`
	Bytes     int     `json:"bytes,omitempty"`
	Error     string  `json:"error,omitempty"`
}

func runStress(gatewayURL string, rounds, workers int) int {
	if workers < 1 {
		workers = 1
	}
	if rounds < 1 {
		rounds = 1
	}
	todo := make(chan int, rounds)
	for i := 0; i < rounds; i++ {
		todo <- 1
	}
	close(todo)

	var mu sync.Mutex
	var out []stressSample
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range todo {
				t0 := time.Now()
				resp, err := http.Get(gatewayURL)
				elapsed := float64(time.Since(t0).Milliseconds())
				if err != nil {
					mu.Lock()
					out = append(out, stressSample{OK: false, URL: gatewayURL, ElapsedMS: round2f(elapsed), Error: err.Error()})
					mu.Unlock()
					continue
				}
				b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
				_ = resp.Body.Close()
				ok := resp.StatusCode < 500
				mu.Lock()
				out = append(out, stressSample{OK: ok, URL: gatewayURL, ElapsedMS: round2f(elapsed), Status: resp.StatusCode, Bytes: len(b)})
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	var oks, fails []stressSample
	for _, s := range out {
		if s.OK {
			oks = append(oks, s)
		} else {
			fails = append(fails, s)
		}
	}
	latencies := make([]float64, 0, len(oks))
	for _, s := range oks {
		latencies = append(latencies, s.ElapsedMS)
	}
	sort.Float64s(latencies)
	rep := map[string]interface{}{
		"request_count": len(out),
		"success_count": len(oks),
		"failure_count": len(fails),
		"latency_ms":    map[string]interface{}{},
	}
	lm := rep["latency_ms"].(map[string]interface{})
	if len(latencies) > 0 {
		lm["min"] = latencies[0]
		lm["p50"] = latencies[len(latencies)/2]
		i95 := int(float64(len(latencies))*0.95) - 1
		if i95 < 0 {
			i95 = 0
		}
		lm["p95"] = latencies[i95]
		lm["max"] = latencies[len(latencies)-1]
	}
	failShow := any(fails)
	if len(fails) > 10 {
		failShow = fails[:10]
	}
	rep["failures"] = failShow
	enc, _ := json.MarshalIndent(rep, "", "  ")
	fmt.Fprintln(os.Stdout, string(enc))
	if len(fails) > 0 {
		return 1
	}
	return 0
}

func round2f(v float64) float64 { return float64(int(v*100+0.5)) / 100 }
