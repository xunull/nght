package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// Client flags — bound to clientCmd via pflag in init() below.
// Client flags — bound to clientCmd via pflag in init() below.
var (
	clientHost        string
	clientPort        int
	clientConcurrency int
	clientDuration    time.Duration
	clientTotal       int
	clientRPS         int
	clientOutput      string
	clientTimeout     time.Duration
	clientPath        string
)

// result is one per-request record sent from worker → aggregator over resultCh.
// Status follows the taxonomy in the v0.0.5 design doc:
//
//	 200..599 — HTTP response code
//	 0        — per-request timeout
//	-1        — other network error (DNS, connection refused, EOF, etc.)
type clientResult struct {
	Latency time.Duration
	Status  int
}

// clientConfig is the resolved (validated) parameter set passed into runLoad.
type clientConfig struct {
	Host        string
	Port        int
	Concurrency int
	Duration    time.Duration // 0 = unlimited
	Total       int           // 0 = unlimited
	RPS         int           // 0 = full speed; non-zero = global pacing
	Timeout     time.Duration
	Path        string // must start with "/"
}

// clientReport is the JSON-serializable output (matches design doc schema).
type clientReport struct {
	Config          map[string]any `json:"config"`
	Summary         clientSummary  `json:"summary"`
	LatencyMs       clientLatency  `json:"latency_ms"`
	StatusHistogram map[string]int `json:"status_histogram"`
}

type clientSummary struct {
	TotalRequests int64   `json:"total_requests"`
	ActualRPS     float64 `json:"actual_rps"`
	DurationMs    int64   `json:"duration_ms"`
	Errors        int64   `json:"errors"`
}

type clientLatency struct {
	P50     float64 `json:"p50"`
	P95     float64 `json:"p95"`
	P99     float64 `json:"p99"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	Samples int     `json:"samples"`
}

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "load-test client (sends HTTP requests, reports p50/p95/p99 latency + status histogram)",
	Long: `nght client is a single-binary load tester for nght (or any HTTP backend).

It spins up N concurrent workers, fires HTTP requests at a target host:port,
and reports p50/p95/p99 latency plus a status code histogram. RPS can be
capped via --rps. The test runs until either --duration elapses or --total
requests complete (whichever comes first — OR semantics).

Examples:
  nght client -H 127.0.0.1 -p 8080 -c 10 -d 10s
  nght client -H nginx -p 80 -c 50 -n 10000 --output json
  nght client -H nght-svc -p 8080 -c 20 -d 30s --path /api/timeout --rps 1000`,
	Run: runClient,
}

func runClient(cmd *cobra.Command, args []string) {
	cfg, err := parseClientConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}

	rpt, err := runClientLoad(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}

	if clientOutput == "json" {
		// Always emit JSON when requested, even on empty stats (zero-value report).
		b, _ := json.MarshalIndent(rpt, "", "  ")
		fmt.Println(string(b))
	} else {
		printClientTextReport(rpt)
	}

	if rpt.Summary.TotalRequests == 0 {
		if clientOutput != "json" {
			fmt.Fprintln(os.Stderr, "no requests completed")
		}
		os.Exit(1)
	}
}

// parseClientConfig validates flags and applies the safety net:
//   - At least one of --duration or --total must be > 0
//     (otherwise the test would run forever).
//   - --concurrency must be >= 1 and <= 10000.
//   - --output must be "text" or "json".
//   - --path must start with "/".
func parseClientConfig() (clientConfig, error) {
	if clientDuration == 0 && clientTotal == 0 {
		return clientConfig{}, errors.New("must specify --duration or --total (or both) — otherwise the test would run forever")
	}
	if clientConcurrency < 1 {
		return clientConfig{}, fmt.Errorf("--concurrency must be >= 1, got %d", clientConcurrency)
	}
	if clientConcurrency > 10000 {
		return clientConfig{}, fmt.Errorf("--concurrency must be <= 10000, got %d", clientConcurrency)
	}
	if clientOutput != "text" && clientOutput != "json" {
		return clientConfig{}, fmt.Errorf("--output must be 'text' or 'json', got %q", clientOutput)
	}
	if !strings.HasPrefix(clientPath, "/") {
		return clientConfig{}, fmt.Errorf("--path must start with '/', got %q", clientPath)
	}
	if clientPort < 1 || clientPort > 65535 {
		return clientConfig{}, fmt.Errorf("--port must be 1..65535, got %d", clientPort)
	}
	if clientTimeout <= 0 {
		return clientConfig{}, fmt.Errorf("--timeout must be > 0, got %s", clientTimeout)
	}
	if clientRPS < 0 {
		return clientConfig{}, fmt.Errorf("--rps must be >= 0, got %d", clientRPS)
	}
	return clientConfig{
		Host:        clientHost,
		Port:        clientPort,
		Concurrency: clientConcurrency,
		Duration:    clientDuration,
		Total:       clientTotal,
		RPS:         clientRPS,
		Timeout:     clientTimeout,
		Path:        clientPath,
	}, nil
}

// runClientLoad is the core dispatcher: spin up N workers, fire requests,
// aggregate results, return the report. Pure function over (cfg, httpClient
// against the network); no cobra / no stdio.
func runClientLoad(cfg clientConfig) (*clientReport, error) {
	target, err := buildTarget(cfg)
	if err != nil {
		return nil, err
	}

	// Single global http.Client — share connection pool across all workers.
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost: max(100, cfg.Concurrency*2),
		},
		Timeout: cfg.Timeout,
		// Don't follow redirects — load-test one endpoint, not chains.
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Cancellation: SIGINT/SIGTERM → ctx.Done() → stopCh (close-once).
	ctx, stopSig := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSig()

	var stopOnce sync.Once
	stopCh := make(chan struct{})
	closeStop := func() { stopOnce.Do(func() { close(stopCh) }) }
	go func() {
		<-ctx.Done()
		closeStop()
	}()

	// 2 stop triggers: duration timer, total counter. --rps is pacing, not stop.
	var totalSent atomic.Int64
	if cfg.Duration > 0 {
		go func() {
			timer := time.NewTimer(cfg.Duration)
			defer timer.Stop()
			select {
			case <-timer.C:
				closeStop()
			case <-stopCh:
			}
		}()
	}
	// --total counter is checked inline by each worker (avoids extra goroutine
	// + lock); see worker loop below.

	// Buffered result channel — workers push, main goroutine aggregates.
	const resultBuf = 1024
	resultCh := make(chan clientResult, resultBuf)

	// RPS pacing: global ticker drops a token into `sema` once per 1s/rps.
	// Workers consume one token before each dispatch. When RPS=0, no ticker
	// fires; workers fall through to full-speed dispatch via the default branch.
	sema := make(chan struct{}, cfg.Concurrency)
	if cfg.RPS > 0 {
		interval := time.Second / time.Duration(cfg.RPS)
		go func() {
			t := time.NewTicker(interval)
			defer t.Stop()
			for {
				select {
				case <-stopCh:
					return
				case <-t.C:
					select {
					case sema <- struct{}{}:
					default:
						// No worker waiting; drop this tick (bounded by cfg.Concurrency
						// in-flight + small over-shoot; not load-bearing).
					}
				}
			}
		}()
	}

	// Worker dispatch loop.
	worker := func() {
		for {
			if cfg.Total > 0 && totalSent.Load() >= int64(cfg.Total) {
				return
			}
			if cfg.RPS > 0 {
				select {
				case <-stopCh:
					return
				case <-sema:
				}
			} else {
				select {
				case <-stopCh:
					return
				default:
				}
			}
			if cfg.Total > 0 && totalSent.Load() >= int64(cfg.Total) {
				return
			}
			totalSent.Add(1)
			sendOne(ctx, httpClient, target, resultCh)
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker()
		}()
	}
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Aggregator: single-threaded in the caller's goroutine, so histogram /
	// reservoir / counters need no mutex.
	startTime := time.Now()
	histogram := make(map[string]int)
	var errorCount int64
	const sampleSize = 10000
	reservoir := make([]time.Duration, 0, sampleSize)
	var reservoirSeen int64
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for r := range resultCh {
		// Histogram bucket.
		var bucket string
		switch {
		case r.Status >= 200 && r.Status <= 599:
			bucket = fmt.Sprintf("%d", r.Status)
			if r.Status >= 400 {
				errorCount++
			}
		case r.Status == 0:
			bucket = "timeout"
			errorCount++
		default:
			bucket = "network_error"
			errorCount++
		}
		histogram[bucket]++

		// Reservoir sampling: Algorithm R.
		reservoirSeen++
		if int64(len(reservoir)) < sampleSize {
			reservoir = append(reservoir, r.Latency)
		} else {
			j := rng.Int63n(reservoirSeen)
			if j < sampleSize {
				reservoir[j] = r.Latency
			}
		}
	}

	elapsed := time.Since(startTime)
	actualRPS := 0.0
	if elapsed > 0 {
		actualRPS = float64(totalSent.Load()) / elapsed.Seconds()
	}

	// Compute percentiles.
	sort.Slice(reservoir, func(i, j int) bool { return reservoir[i] < reservoir[j] })
	p50, p95, p99, minMs, maxMs := 0.0, 0.0, 0.0, 0.0, 0.0
	if n := len(reservoir); n > 0 {
		minMs = ms(reservoir[0])
		maxMs = ms(reservoir[n-1])
		p50 = ms(reservoir[percentileIndex(0.50, n)])
		p95 = ms(reservoir[percentileIndex(0.95, n)])
		p99 = ms(reservoir[percentileIndex(0.99, n)])
	}

	return &clientReport{
		Config: map[string]any{
			"host":        cfg.Host,
			"port":        cfg.Port,
			"concurrency": cfg.Concurrency,
			"duration":    cfg.Duration.String(),
			"total":       cfg.Total,
			"rps":         cfg.RPS,
			"timeout":     cfg.Timeout.String(),
			"path":        cfg.Path,
		},
		Summary: clientSummary{
			TotalRequests: totalSent.Load(),
			ActualRPS:     roundTo(actualRPS, 2),
			DurationMs:    elapsed.Milliseconds(),
			Errors:        errorCount,
		},
		LatencyMs: clientLatency{
			P50:     roundTo(p50, 2),
			P95:     roundTo(p95, 2),
			P99:     roundTo(p99, 2),
			Min:     roundTo(minMs, 2),
			Max:     roundTo(maxMs, 2),
			Samples: len(reservoir),
		},
		StatusHistogram: histogram,
	}, nil
}

// sendOne fires one GET against target. It must NOT block on resultCh —
// the channel is buffered; if full, the result is dropped (rare; bounded
// by 1024 in flight + aggregator processing rate).
func sendOne(ctx context.Context, httpClient *http.Client, target string, resultCh chan<- clientResult) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		trySendResult(resultCh, clientResult{Latency: time.Since(start), Status: -1})
		return
	}
	resp, err := httpClient.Do(req)
	latency := time.Since(start)
	if err != nil {
		var nerr net.Error
		if errors.As(err, &nerr) && nerr.Timeout() {
			trySendResult(resultCh, clientResult{Latency: latency, Status: 0})
		} else {
			trySendResult(resultCh, clientResult{Latency: latency, Status: -1})
		}
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	trySendResult(resultCh, clientResult{Latency: latency, Status: resp.StatusCode})
}

// trySendResult pushes r non-blockingly. If the aggregator is too slow,
// the result is dropped (bounded by 1024 buf + per-request state on the wire).
func trySendResult(resultCh chan<- clientResult, r clientResult) {
	select {
	case resultCh <- r:
	default:
	}
}

func buildTarget(cfg clientConfig) (string, error) {
	u := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:   cfg.Path,
	}
	return u.String(), nil
}

// percentileIndex returns the nearest-rank index for percentile p in [0,1]
// over a sorted slice of length n. (Standard "nearest-rank" method.)
func percentileIndex(p float64, n int) int {
	if n <= 0 {
		return 0
	}
	idx := int(float64(n) * p)
	if idx >= n {
		idx = n - 1
	}
	if idx < 0 {
		idx = 0
	}
	return idx
}

// ms converts a Duration to a millisecond float.
func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000.0 }

// roundTo rounds f to `places` decimal places.
func roundTo(f float64, places int) float64 {
	pow := 1.0
	for i := 0; i < places; i++ {
		pow *= 10
	}
	return float64(int64(f*pow+0.5)) / pow
}

// printClientTextReport writes the human-readable summary to stdout.
func printClientTextReport(r *clientReport) {
	fmt.Printf("Target:        %s:%d%s\n", r.Config["host"], r.Config["port"], r.Config["path"])
	fmt.Printf("Concurrency:   %v\n", r.Config["concurrency"])
	fmt.Printf("Duration:      %v (actual %d ms)\n", r.Config["duration"], r.Summary.DurationMs)
	fmt.Printf("Total:         %v\n", r.Config["total"])
	fmt.Printf("RPS cap:       %v\n", r.Config["rps"])
	fmt.Println()
	fmt.Printf("Total requests: %d\n", r.Summary.TotalRequests)
	fmt.Printf("Actual RPS:     %.2f\n", r.Summary.ActualRPS)
	fmt.Printf("Errors:         %d\n", r.Summary.Errors)
	fmt.Println()
	fmt.Println("Latency (ms):")
	fmt.Printf("  p50 = %.2f  p95 = %.2f  p99 = %.2f  min = %.2f  max = %.2f  samples = %d\n",
		r.LatencyMs.P50, r.LatencyMs.P95, r.LatencyMs.P99, r.LatencyMs.Min, r.LatencyMs.Max, r.LatencyMs.Samples)
	fmt.Println()
	fmt.Println("Status histogram:")
	if len(r.StatusHistogram) == 0 {
		fmt.Println("  (none)")
		return
	}
	// Stable order: HTTP codes numeric ascending, then timeout, then network_error.
	codes := make([]string, 0, len(r.StatusHistogram))
	for k := range r.StatusHistogram {
		codes = append(codes, k)
	}
	sort.Slice(codes, func(i, j int) bool {
		ai, aIsCode := atoiOrNeg(codes[i])
		bi, bIsCode := atoiOrNeg(codes[j])
		switch {
		case aIsCode && bIsCode:
			return ai < bi
		case aIsCode:
			return true
		case bIsCode:
			return false
		default:
			return codes[i] < codes[j]
		}
	})
	for _, k := range codes {
		fmt.Printf("  %-15s %d\n", k, r.StatusHistogram[k])
	}
}

func atoiOrNeg(s string) (int, bool) {
	if len(s) == 0 {
		return 0, false
	}
	n := 0
	for i, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
		if i > 9 {
			return 0, false
		}
	}
	return n, true
}

func init() {
	rootCmd.AddCommand(clientCmd)

	f := clientCmd.Flags()
	// NB: short letter for --host is `-H` (capital) not `-h` — cobra reserves
	// `-h` for the auto-generated --help on every subcommand. Using `-h` here
	// panics on first execute. Design doc said `-h`; this is a 1-character
	// deviation documented in the commit message.
	f.StringVarP(&clientHost, "host", "H", "127.0.0.1", "target host")
	f.IntVarP(&clientPort, "port", "p", 8080, "target port")
	f.IntVarP(&clientConcurrency, "concurrency", "c", 10, "concurrent workers")
	f.DurationVarP(&clientDuration, "duration", "d", 10*time.Second, "test duration (0 = unlimited, requires --total)")
	f.IntVarP(&clientTotal, "total", "n", 0, "stop after N total requests (0 = unlimited, requires --duration)")
	f.IntVarP(&clientRPS, "rps", "q", 0, "cap requests per second (0 = full speed)")
	f.StringVarP(&clientOutput, "output", "o", "text", "report format: text or json")
	f.DurationVar(&clientTimeout, "timeout", 5*time.Second, "per-request timeout")
	f.StringVar(&clientPath, "path", "/", "URL path to hit (must start with /)")
}
