#!/bin/sh
set -eu

go run ./tools/check_audits

if awk -F '|' '
	/^## Appendix C/ { appendix = 1 }
	appendix && /^\|/ {
		status = $4
		gsub(/^[[:space:]]+|[[:space:]]+$/, "", status)
		if (status == "Open") {
			print
			found = 1
		}
	}
	END { exit found ? 1 : 0 }
' docs/DESIGN.md; then
	echo "Appendix C has no Open configuration data rows"
else
	echo "release readiness: Appendix C still contains Open configuration rows" >&2
	exit 1
fi

./tools/check_moisture_balance.sh

for policy in reference toward-south-asia toward-yellow-river toward-sahul toward-beringia; do
	checkpoint="$(mktemp)"
	trap 'rm -f "$checkpoint"' EXIT HUP INT TERM
	go run . -headless -turns 400 -seed 0x9e3779b97f4a7c15 -policy "$policy" -checkpoint-json "$checkpoint" >/dev/null
	if ! python3 - "$checkpoint" "$policy" <<'PY'
import json
import sys

path, policy = sys.argv[1:]
try:
    with open(path, encoding="utf-8") as source:
        records = json.load(source)
    final = records[-1]
    valid = isinstance(final, dict) and final.get("turn") == 400 and final.get("target_established") is True
except (OSError, ValueError, IndexError, TypeError):
    valid = False
if not valid:
    print(f"release readiness: {policy} has no established target in its terminal turn-400 record", file=sys.stderr)
    raise SystemExit(1)
PY
	then
		echo "release readiness: $policy did not establish its target" >&2
		exit 1
	fi
	echo "$policy established its target"
	rm -f "$checkpoint"
	trap - EXIT HUP INT TERM
done

./tools/check_benchmarks.sh
