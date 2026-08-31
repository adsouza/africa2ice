package main

import (
	"bufio"
	"encoding/json"
	"fmt"
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
	configuration := readBaseline(os.Args[2])
	measurements := readSamples(os.Args[1])
	calibration := measurements[configuration.Calibration]
	if len(calibration) != configuration.SampleCount {
		fatalf("%s has %d samples, want %d", configuration.Calibration, len(calibration), configuration.SampleCount)
	}
	calibrationMedian := median(calibration, func(value sample) float64 { return value.Nanoseconds })
	if calibrationMedian <= 0 {
		fatalf("calibration median must be positive")
	}
	if len(configuration.Maximums)+1 != configuration.BenchmarkCount {
		fatalf("baseline benchmark_count=%d but names %d benchmarks", configuration.BenchmarkCount, len(configuration.Maximums)+1)
	}
	names := make([]string, 0, len(configuration.Maximums))
	for name := range configuration.Maximums {
		names = append(names, name)
	}
	sort.Strings(names)
	failed := false
	for _, name := range names {
		values := measurements[name]
		if len(values) != configuration.SampleCount {
			fatalf("%s has %d samples, want %d", name, len(values), configuration.SampleCount)
		}
		ns := median(values, func(value sample) float64 { return value.Nanoseconds })
		bytesPerOp := median(values, func(value sample) float64 { return value.Bytes })
		allocsPerOp := median(values, func(value sample) float64 { return value.Allocs })
		ratio := ns / calibrationMedian
		maximum := configuration.Maximums[name]
		fmt.Printf("%s: ratio=%.2f bytes/op=%.0f allocs/op=%.0f\n", name, ratio, bytesPerOp, allocsPerOp)
		if ratio > maximum.RatioToCalibration {
			fmt.Fprintf(os.Stderr, "%s ratio %.2f exceeds %.2f\n", name, ratio, maximum.RatioToCalibration)
			failed = true
		}
		if bytesPerOp > maximum.BytesPerOp {
			fmt.Fprintf(os.Stderr, "%s bytes/op %.0f exceeds %.0f\n", name, bytesPerOp, maximum.BytesPerOp)
			failed = true
		}
		if allocsPerOp > maximum.AllocsPerOp {
			fmt.Fprintf(os.Stderr, "%s allocs/op %.0f exceeds %.0f\n", name, allocsPerOp, maximum.AllocsPerOp)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func readBaseline(path string) baseline {
	payload, err := os.ReadFile(path)
	if err != nil {
		fatalf("read baseline: %v", err)
	}
	var result baseline
	if err := json.Unmarshal(payload, &result); err != nil {
		fatalf("decode baseline: %v", err)
	}
	if result.SampleCount < 1 || result.BenchmarkCount < 1 || result.Calibration == "" || len(result.Maximums) == 0 {
		fatalf("invalid benchmark baseline")
	}
	return result
}

func readSamples(path string) map[string][]sample {
	file, err := os.Open(path)
	if err != nil {
		fatalf("read benchmark output: %v", err)
	}
	defer func() { _ = file.Close() }()
	result := make(map[string][]sample)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 8 || !strings.HasPrefix(fields[0], "Benchmark") || fields[3] != "ns/op" || fields[5] != "B/op" || fields[7] != "allocs/op" {
			continue
		}
		name := fields[0]
		if dash := strings.LastIndexByte(name, '-'); dash >= 0 {
			name = name[:dash]
		}
		result[name] = append(result[name], sample{
			Nanoseconds: parseNumber(name, "ns/op", fields[2]),
			Bytes:       parseNumber(name, "B/op", fields[4]),
			Allocs:      parseNumber(name, "allocs/op", fields[6]),
		})
	}
	if err := scanner.Err(); err != nil {
		fatalf("scan benchmark output: %v", err)
	}
	return result
}

func parseNumber(name, metric, raw string) float64 {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		fatalf("%s invalid %s value %q", name, metric, raw)
	}
	return value
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
