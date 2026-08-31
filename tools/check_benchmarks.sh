#!/bin/sh
set -eu

output="$(mktemp)"
trap 'rm -f "$output"' EXIT HUP INT TERM

go test ./internal/domain ./internal/application \
	-run '^$' \
	-bench 'Benchmark(AdvanceTurnMaximumWorkload|FrameProjectionMaximumWorkload|Calibration)$' \
	-benchmem -benchtime=100ms -count=5 | tee "$output"

go run ./tools/check_benchmarks "$output" testdata/performance_baseline.json
