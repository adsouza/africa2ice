package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Lines in the exact shape `go test -bench -benchmem` prints, including the
// GOMAXPROCS suffix and the surrounding noise the parser must skip.
const benchOutput = `goos: darwin
goarch: arm64
pkg: github.com/adsouza/africa2ice/internal/application
cpu: Apple M4
BenchmarkCalibration-10                       	    8485	     100 ns/op	   55296 B/op	      96 allocs/op
BenchmarkCalibration-10                       	    8616	     100 ns/op	   55296 B/op	      96 allocs/op
BenchmarkCalibration-10                       	    8708	     100 ns/op	   55296 B/op	      96 allocs/op
BenchmarkWorkload-10    	     193	    1000 ns/op	 4000 B/op	    10 allocs/op
BenchmarkWorkload-10    	     212	    3000 ns/op	 4000 B/op	    10 allocs/op
BenchmarkWorkload-10    	     213	    2000 ns/op	 4000 B/op	    10 allocs/op
PASS
ok  	github.com/adsouza/africa2ice/internal/application	1.686s
`

func testBaseline() baseline {
	return baseline{
		BenchmarkCount: 2, Calibration: "BenchmarkCalibration", SampleCount: 3,
		Maximums: map[string]limit{"BenchmarkWorkload": {RatioToCalibration: 25, BytesPerOp: 4000, AllocsPerOp: 10}},
	}
}

func parsed(t *testing.T) map[string][]sample {
	t.Helper()
	measurements, err := parseSamples(strings.NewReader(benchOutput))
	if err != nil {
		t.Fatal(err)
	}
	return measurements
}

func TestParseSamplesReadsBenchmarkLinesAndSkipsNoise(t *testing.T) {
	measurements := parsed(t)
	if len(measurements) != 2 || len(measurements["BenchmarkCalibration"]) != 3 || len(measurements["BenchmarkWorkload"]) != 3 {
		t.Fatalf("parsed %v", measurements)
	}
	if got := measurements["BenchmarkWorkload"][1]; got != (sample{Nanoseconds: 3000, Bytes: 4000, Allocs: 10}) {
		t.Fatalf("second workload sample = %+v", got)
	}
}

func TestParseSamplesRejectsAMalformedNumber(t *testing.T) {
	_, err := parseSamples(strings.NewReader("BenchmarkWorkload-10  1  1e  ns/op  4 B/op  1 allocs/op\n"))
	if err == nil || !strings.Contains(err.Error(), "ns/op") {
		t.Fatalf("err = %v, want a malformed ns/op error", err)
	}
}

func TestEvaluateAcceptsMediansAtTheirLimits(t *testing.T) {
	results, violations, err := evaluate(testBaseline(), parsed(t))
	if err != nil || len(violations) != 0 {
		t.Fatalf("violations %v, err %v; want none at exactly the limits", violations, err)
	}
	// median ns 2000 over calibration median 100.
	if len(results) != 1 || results[0].Ratio != 20 || results[0].BytesPerOp != 4000 || results[0].AllocsPerOp != 10 {
		t.Fatalf("results = %+v", results)
	}
}

// The gate is only worth its green if each limit, crossed alone, fails it.
func TestEvaluateFailsEachLimitIndependently(t *testing.T) {
	for _, tc := range []struct {
		metric  string
		tighten func(*limit)
	}{
		{"ratio", func(l *limit) { l.RatioToCalibration = 19.99 }},
		{"bytes/op", func(l *limit) { l.BytesPerOp = 3999 }},
		{"allocs/op", func(l *limit) { l.AllocsPerOp = 9 }},
	} {
		t.Run(tc.metric, func(t *testing.T) {
			configuration := testBaseline()
			maximum := configuration.Maximums["BenchmarkWorkload"]
			tc.tighten(&maximum)
			configuration.Maximums["BenchmarkWorkload"] = maximum
			_, violations, err := evaluate(configuration, parsed(t))
			if err != nil {
				t.Fatal(err)
			}
			if len(violations) != 1 || !strings.Contains(violations[0], "BenchmarkWorkload "+tc.metric) {
				t.Fatalf("violations = %v, want exactly the %s limit", violations, tc.metric)
			}
		})
	}
}

func TestEvaluateRejectsAnIncompleteRun(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*baseline, map[string][]sample)
		want   string
	}{
		{"calibration sample count", func(_ *baseline, m map[string][]sample) {
			m["BenchmarkCalibration"] = m["BenchmarkCalibration"][:2]
		}, "BenchmarkCalibration has 2 samples"},
		{"missing benchmark", func(_ *baseline, m map[string][]sample) { delete(m, "BenchmarkWorkload") }, "BenchmarkWorkload has 0 samples"},
		{"benchmark count", func(b *baseline, _ map[string][]sample) { b.BenchmarkCount = 3 }, "benchmark_count=3"},
		{"zero calibration", func(_ *baseline, m map[string][]sample) {
			for index := range m["BenchmarkCalibration"] {
				m["BenchmarkCalibration"][index].Nanoseconds = 0
			}
		}, "calibration median must be positive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configuration, measurements := testBaseline(), parsed(t)
			tc.mutate(&configuration, measurements)
			if _, _, err := evaluate(configuration, measurements); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestParseBaselineRejectsAnIncompleteBaseline(t *testing.T) {
	for _, payload := range []string{`{}`, `{"benchmark_count":1,"sample_count":1,"maximums":{"B":{}}}`, `not json`} {
		if _, err := parseBaseline([]byte(payload)); err == nil {
			t.Fatalf("parseBaseline(%s) accepted it", payload)
		}
	}
}

// The checked-in baseline is what CI gates on; it must parse and agree with
// itself, so a hand edit cannot leave benchmark_count or a name stale.
func TestCheckedInBaselineIsConsistent(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join("..", "..", "testdata", "performance_baseline.json"))
	if err != nil {
		t.Fatal(err)
	}
	configuration, err := parseBaseline(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(configuration.Maximums)+1 != configuration.BenchmarkCount {
		t.Fatalf("benchmark_count %d does not match %d maximums plus calibration", configuration.BenchmarkCount, len(configuration.Maximums))
	}
	for name, maximum := range configuration.Maximums {
		if maximum.RatioToCalibration <= 0 || maximum.BytesPerOp <= 0 || maximum.AllocsPerOp <= 0 {
			t.Fatalf("%s has a non-positive limit %+v, which no run could pass", name, maximum)
		}
	}
}
