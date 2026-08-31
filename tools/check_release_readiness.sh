#!/bin/sh
set -eu

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

for policy in reference toward-south-asia toward-yellow-river toward-sahul toward-beringia; do
	checkpoint="$(mktemp)"
	trap 'rm -f "$checkpoint"' EXIT HUP INT TERM
	go run . -headless -turns 400 -seed 0x9e3779b97f4a7c15 -policy "$policy" -checkpoint-json "$checkpoint" >/dev/null
	if ! tail -c 1600 "$checkpoint" | grep -q '"target_established":true'; then
		echo "release readiness: $policy did not establish its target" >&2
		exit 1
	fi
	echo "$policy established its target"
	rm -f "$checkpoint"
	trap - EXIT HUP INT TERM
done

./tools/check_benchmarks.sh
