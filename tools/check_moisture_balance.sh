#!/bin/sh
set -eu

report_file="$(mktemp)"
trap 'rm -f "$report_file"' EXIT HUP INT TERM

go run ./tools/report_moisture_balance >"$report_file"
if ! cmp -s docs/MOISTURE_BALANCE.json "$report_file"; then
	echo "moisture balance report changed; review it and refresh docs/MOISTURE_BALANCE.json" >&2
	diff -u docs/MOISTURE_BALANCE.json "$report_file" || true
	exit 1
fi

echo "moisture-driven biome, resource, desert-residency, and recovery gates passed"
