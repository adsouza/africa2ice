package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

type limit struct {
	AllocsPerOp        float64 `json:"allocs_per_op"`
	BytesPerOp         float64 `json:"bytes_per_op"`
	RatioToCalibration float64 `json:"ratio_to_calibration"`
}

type baseline struct {
	BenchmarkCount int              `json:"benchmark_count"`
	Calibration    string           `json:"calibration"`
	Maximums       map[string]limit `json:"maximums"`
	SampleCount    int              `json:"sample_count"`
}

type sample struct {
	Nanoseconds float64
	Bytes       float64
	Allocs      float64
}

func main() {
	if len(os.Args) != 3 {
		fatalf("usage: check_benchmarks <benchmark-output> <baseline-json>")
	}
	payload, err := os.ReadFile(os.Args[2])
	if err != nil {
		fatalf("read baseline: %v", err)
	}
	configuration, err := parseBaseline(payload)
	if err != nil {
		fatalf("%v", err)
	}
	file, err := os.Open(os.Args[1])
	if err != nil {
		fatalf("read benchmark output: %v", err)
	}
	measurements, err := parseSamples(file)
	_ = file.Close()
	if err != nil {
		fatalf("%v", err)
	}
	results, violations, err := evaluate(configuration, measurements)
	for _, result := range results {
		fmt.Printf("%s: ratio=%.2f bytes/op=%.0f allocs/op=%.0f\n", result.Name, result.Ratio, result.BytesPerOp, result.AllocsPerOp)
	}
	if err != nil {
		fatalf("%v", err)
	}
	for _, violation := range violations {
		fmt.Fprintln(os.Stderr, violation)
	}
	if len(violations) != 0 {
		os.Exit(1)
	}
}

// result is one benchmark's medians, with time as a ratio to the calibration.
type result struct {
	Name                           string
	Ratio, BytesPerOp, AllocsPerOp float64
}

// evaluate compares each named benchmark's medians with its maximums. An error
// means the run itself cannot be judged (missing or short samples, an
// inconsistent baseline); violations are limits the run exceeded.
func evaluate(configuration baseline, measurements map[string][]sample) ([]result, []string, error) {
	calibration := measurements[configuration.Calibration]
	if len(calibration) != configuration.SampleCount {
		return nil, nil, fmt.Errorf("%s has %d samples, want %d", configuration.Calibration, len(calibration), configuration.SampleCount)
	}
	calibrationMedian := median(calibration, func(value sample) float64 { return value.Nanoseconds })
	if calibrationMedian <= 0 {
		return nil, nil, fmt.Errorf("calibration median must be positive")
	}
	if len(configuration.Maximums)+1 != configuration.BenchmarkCount {
		return nil, nil, fmt.Errorf("baseline benchmark_count=%d but names %d benchmarks", configuration.BenchmarkCount, len(configuration.Maximums)+1)
	}
	names := make([]string, 0, len(configuration.Maximums))
	for name := range configuration.Maximums {
		names = append(names, name)
	}
	sort.Strings(names)
	var results []result
	var violations []string
	for _, name := range names {
		values := measurements[name]
		if len(values) != configuration.SampleCount {
			return results, violations, fmt.Errorf("%s has %d samples, want %d", name, len(values), configuration.SampleCount)
		}
		current := result{
			Name:        name,
			Ratio:       median(values, func(value sample) float64 { return value.Nanoseconds }) / calibrationMedian,
			BytesPerOp:  median(values, func(value sample) float64 { return value.Bytes }),
			AllocsPerOp: median(values, func(value sample) float64 { return value.Allocs }),
		}
		results = append(results, current)
		maximum := configuration.Maximums[name]
		if current.Ratio > maximum.RatioToCalibration {
			violations = append(violations, fmt.Sprintf("%s ratio %.2f exceeds %.2f", name, current.Ratio, maximum.RatioToCalibration))
		}
		if current.BytesPerOp > maximum.BytesPerOp {
			violations = append(violations, fmt.Sprintf("%s bytes/op %.0f exceeds %.0f", name, current.BytesPerOp, maximum.BytesPerOp))
		}
		if current.AllocsPerOp > maximum.AllocsPerOp {
			violations = append(violations, fmt.Sprintf("%s allocs/op %.0f exceeds %.0f", name, current.AllocsPerOp, maximum.AllocsPerOp))
		}
	}
	return results, violations, nil
}

func parseBaseline(payload []byte) (baseline, error) {
	var result baseline
	if err := json.Unmarshal(payload, &result); err != nil {
		return baseline{}, fmt.Errorf("decode baseline: %w", err)
	}
	if result.SampleCount < 1 || result.BenchmarkCount < 1 || result.Calibration == "" || len(result.Maximums) == 0 {
		return baseline{}, fmt.Errorf("invalid benchmark baseline")
	}
	return result, nil
}

// parseSamples reads `go test -bench -benchmem` output, keeping one sample per
// benchmark line and skipping every other line.
func parseSamples(input io.Reader) (map[string][]sample, error) {
	result := make(map[string][]sample)
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 8 || !strings.HasPrefix(fields[0], "Benchmark") || fields[3] != "ns/op" || fields[5] != "B/op" || fields[7] != "allocs/op" {
			continue
		}
		name := fields[0]
		if dash := strings.LastIndexByte(name, '-'); dash >= 0 {
			name = name[:dash]
		}
		var measured sample
		for _, field := range []struct {
			metric string
			raw    string
			into   *float64
		}{{"ns/op", fields[2], &measured.Nanoseconds}, {"B/op", fields[4], &measured.Bytes}, {"allocs/op", fields[6], &measured.Allocs}} {
			value, err := strconv.ParseFloat(field.raw, 64)
			if err != nil {
				return nil, fmt.Errorf("%s invalid %s value %q", name, field.metric, field.raw)
			}
			*field.into = value
		}
		result[name] = append(result[name], measured)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan benchmark output: %w", err)
	}
	return result, nil
}

func median(values []sample, extract func(sample) float64) float64 {
	items := make([]float64, len(values))
	for index, value := range values {
		items[index] = extract(value)
	}
	sort.Float64s(items)
	return items[len(items)/2]
}

func fatalf(format string, arguments ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", arguments...)
	os.Exit(2)
}
