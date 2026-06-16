package cmd

import (
	"math"
	"testing"
	"time"
)

// TestParseClientConfig covers the safety-net validation paths.
func TestParseClientConfig(t *testing.T) {
	// Mutate package-level flags in a controlled way; restore at end.
	origHost, origPort, origConc, origDur, origTotal, origRPS, origOut, origTO, origPath :=
		clientHost, clientPort, clientConcurrency, clientDuration, clientTotal, clientRPS, clientOutput, clientTimeout, clientPath
	t.Cleanup(func() {
		clientHost, clientPort, clientConcurrency, clientDuration, clientTotal, clientRPS, clientOutput, clientTimeout, clientPath =
			origHost, origPort, origConc, origDur, origTotal, origRPS, origOut, origTO, origPath
	})

	type tc struct {
		name    string
		mutate  func()
		wantErr string // substring of expected error; "" = expect OK
	}
	cases := []tc{
		{
			name: "happy path",
			mutate: func() {
				clientHost, clientPort = "127.0.0.1", 8080
				clientConcurrency = 5
				clientDuration = 2 * time.Second
				clientTotal = 0
				clientRPS = 0
				clientOutput = "json"
				clientTimeout = 1 * time.Second
				clientPath = "/x"
			},
			wantErr: "",
		},
		{
			name: "safety net: d=0 n=0",
			mutate: func() {
				clientHost, clientPort = "127.0.0.1", 8080
				clientConcurrency = 1
				clientDuration = 0
				clientTotal = 0
				clientTimeout = time.Second
			},
			wantErr: "must specify --duration or --total",
		},
		{
			name: "concurrency 0",
			mutate: func() {
				clientHost, clientPort = "h", 80
				clientConcurrency = 0
				clientDuration = time.Second
				clientTotal = 0
			},
			wantErr: "--concurrency must be >= 1",
		},
		{
			name: "concurrency over max",
			mutate: func() {
				clientHost, clientPort = "h", 80
				clientConcurrency = 10001
				clientDuration = time.Second
				clientTotal = 0
			},
			wantErr: "--concurrency must be <= 10000",
		},
		{
			name: "output invalid",
			mutate: func() {
				clientHost, clientPort = "h", 80
				clientConcurrency = 1
				clientDuration = time.Second
				clientTotal = 0
				clientOutput = "yaml"
			},
			wantErr: "--output must be 'text' or 'json'",
		},
		{
			name: "path missing leading slash",
			mutate: func() {
				clientHost, clientPort = "h", 80
				clientConcurrency = 1
				clientDuration = time.Second
				clientTotal = 0
				clientOutput = "text"
				clientPath = "x"
			},
			wantErr: "--path must start with '/'",
		},
		{
			name: "port 0",
			mutate: func() {
				clientHost, clientPort = "h", 0
				clientConcurrency = 1
				clientDuration = time.Second
				clientTotal = 0
				clientPath = "/x"
			},
			wantErr: "--port must be 1..65535",
		},
		{
			name: "port too high",
			mutate: func() {
				clientHost, clientPort = "h", 70000
				clientConcurrency = 1
				clientDuration = time.Second
				clientTotal = 0
				clientPath = "/x"
			},
			wantErr: "--port must be 1..65535",
		},
		{
			name: "timeout zero",
			mutate: func() {
				clientHost, clientPort = "h", 80
				clientConcurrency = 1
				clientDuration = time.Second
				clientTotal = 0
				clientPath = "/x"
				clientTimeout = 0
			},
			wantErr: "--timeout must be > 0",
		},
		{
			name: "rps negative",
			mutate: func() {
				clientHost, clientPort = "h", 80
				clientConcurrency = 1
				clientDuration = time.Second
				clientTotal = 0
				clientPath = "/x"
				clientTimeout = time.Second
				clientRPS = -1
			},
			wantErr: "--rps must be >= 0",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.mutate()
			_, err := parseClientConfig()
			switch {
			case c.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case c.wantErr != "" && err == nil:
				t.Fatalf("expected error containing %q, got nil", c.wantErr)
			case c.wantErr != "" && !contains(err.Error(), c.wantErr):
				t.Fatalf("error = %v, want substring %q", err, c.wantErr)
			}
		})
	}
}

func TestPercentileIndex_NearestRank(t *testing.T) {
	// 100 samples indexed 0..99. Nearest-rank: idx = floor(p * n); clamped to [0,n-1].
	n := 100
	cases := []struct {
		p    float64
		want int
	}{
		{0.0, 0},
		{0.5, 50},
		{0.95, 95},
		{0.99, 99},
		{1.0, 99},
		{1.5, 99},
		{-0.1, 0},
	}
	for _, c := range cases {
		got := percentileIndex(c.p, n)
		if got != c.want {
			t.Errorf("percentileIndex(%v, %d) = %d, want %d", c.p, n, got, c.want)
		}
	}
	// n=0 must return 0 (no panic, no divide-by-zero).
	if got := percentileIndex(0.5, 0); got != 0 {
		t.Errorf("percentileIndex(0.5, 0) = %d, want 0", got)
	}
}

func TestAtoiOrNeg(t *testing.T) {
	cases := []struct {
		in   string
		want int
		isOk bool
	}{
		{"0", 0, true},
		{"200", 200, true},
		{"599", 599, true},
		{"", 0, false},
		{"abc", 0, false},
		{"-1", 0, false}, // negative disallowed
		{"12a", 0, false},
		{"10000000000", 0, false}, // > 10 digits
	}
	for _, c := range cases {
		got, ok := atoiOrNeg(c.in)
		if ok != c.isOk {
			t.Errorf("atoiOrNeg(%q) ok = %v, want %v", c.in, ok, c.isOk)
		}
		if c.isOk && got != c.want {
			t.Errorf("atoiOrNeg(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestRoundTo(t *testing.T) {
	cases := []struct {
		in     float64
		places int
		want   float64
	}{
		{1.23456, 2, 1.23},
		{1.235, 2, 1.24}, // bankers? No — we use int64(f*pow+0.5)/pow, so .5 rounds up
		{0.0, 3, 0.0},
		{3.14159, 0, 3.0},
		{3.5, 0, 4.0},   // .5 rounds away from zero
		{-1.5, 0, -1.0}, // int64(-1.5) = -1, +0.5 => -1, so no change. -1.0 expected.
	}
	for _, c := range cases {
		got := roundTo(c.in, c.places)
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("roundTo(%v, %d) = %v, want %v", c.in, c.places, got, c.want)
		}
	}
}

func TestMs(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want float64
	}{
		{0, 0},
		{time.Millisecond, 1},
		{500 * time.Microsecond, 0.5},
		{2 * time.Second, 2000},
	}
	for _, c := range cases {
		got := ms(c.in)
		if math.Abs(got-c.want) > 1e-6 {
			t.Errorf("ms(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestBuildTarget(t *testing.T) {
	cases := []struct {
		cfg  clientConfig
		want string
	}{
		{clientConfig{Host: "127.0.0.1", Port: 8080, Path: "/"}, "http://127.0.0.1:8080/"},
		{clientConfig{Host: "nginx", Port: 80, Path: "/api/v1/x"}, "http://nginx:80/api/v1/x"},
	}
	for _, c := range cases {
		got, err := buildTarget(c.cfg)
		if err != nil {
			t.Fatalf("buildTarget(%+v) error: %v", c.cfg, err)
		}
		if got != c.want {
			t.Errorf("buildTarget(%+v) = %q, want %q", c.cfg, got, c.want)
		}
	}
}

// contains is a tiny strings.Contains substitute to avoid an extra import
// in this test-only file.
func contains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
