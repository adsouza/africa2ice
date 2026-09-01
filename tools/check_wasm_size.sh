#!/bin/sh
# Measure the deployable wasm module and apply DESIGN.md §10's compressed-size
# gate in both directions.
#
# Growth is the obvious half: brotli bytes must not exceed
# MaxCompressedWasmBytes. The other half is what makes this a ratchet rather
# than a number that records whatever the build happens to weigh — when the
# build has shrunk far enough that the ceiling no longer constrains anything,
# the check fails and names the exact replacement value.
#
#   usage: tools/check_wasm_size.sh [wasm-path] [record-path]
#
# Defaults to web/main.wasm. Writes a JSON record for the release log, and
# appends a summary table when running under GitHub Actions.
#
# Exit codes: 0 pass, 1 gate failure, 2 setup error (missing tool or input).
# Lint with `shellcheck -x`, so that the sourced budget file is followed.

set -eu

wasm="${1:-web/main.wasm}"
record="${2:-wasm-size.json}"
budget="$(dirname "$0")/wasm_size_budget.env"

for tool in brotli gzip; do
	command -v "$tool" >/dev/null 2>&1 || {
		echo "check_wasm_size: $tool is required" >&2
		exit 2
	}
done
test -s "$wasm" || {
	echo "check_wasm_size: $wasm is missing or empty" >&2
	exit 2
}
test -f "$budget" || {
	echo "check_wasm_size: $budget is missing" >&2
	exit 2
}

# shellcheck source=tools/wasm_size_budget.env
. "$budget"

# The original ceiling is the bootstrap bound. The comparison against the
# trusted base revision below is what makes every later reduction permanent.
if [ "$MaxCompressedWasmBytes" -gt "$AppendixCOriginalCeiling" ]; then
	echo "check_wasm_size: MaxCompressedWasmBytes=$MaxCompressedWasmBytes exceeds Appendix C's original ceiling of $AppendixCOriginalCeiling." >&2
	echo "  The direction rule for this row is tighten only. Raising it is not a way to pass a failing build." >&2
	exit 1
fi

requested_base_ref="${WASM_SIZE_BASE_REF:-}"
base_ref="${requested_base_ref:-HEAD^}"
case "$base_ref" in
	0000000000000000000000000000000000000000) base_ref=HEAD^ ;;
esac
base_budget=""
if git rev-parse --verify "$base_ref^{commit}" >/dev/null 2>&1; then
	base_budget="$(git show "$base_ref:tools/wasm_size_budget.env" 2>/dev/null || true)"
elif [ -n "$requested_base_ref" ]; then
	echo "check_wasm_size: trusted base revision $base_ref is unavailable" >&2
	exit 2
fi
if [ -n "$base_budget" ]; then
	base_ceiling="$(printf '%s\n' "$base_budget" | sed -n 's/^MaxCompressedWasmBytes=//p')"
	base_headroom="$(printf '%s\n' "$base_budget" | sed -n 's/^CompressedWasmHeadroom=//p')"
	base_quantum="$(printf '%s\n' "$base_budget" | sed -n 's/^RatchetQuantum=//p')"
	case "$base_ceiling:$base_headroom:$base_quantum" in
		*[!0-9:]* | :* | *: | *::*) echo "check_wasm_size: $base_ref has an invalid wasm budget" >&2; exit 2 ;;
	esac
	if [ "$MaxCompressedWasmBytes" -gt "$base_ceiling" ]; then
		echo "check_wasm_size: MaxCompressedWasmBytes increased from $base_ceiling at $base_ref to $MaxCompressedWasmBytes." >&2
		echo "  This policy is tighten-only; restore or lower the trusted-base ceiling." >&2
		exit 1
	fi
	if [ "$CompressedWasmHeadroom" -gt "$base_headroom" ]; then
		echo "check_wasm_size: CompressedWasmHeadroom increased from $base_headroom at $base_ref to $CompressedWasmHeadroom." >&2
		echo "  This policy is tighten-only; restore or lower the trusted-base headroom." >&2
		exit 1
	fi
	if [ "$RatchetQuantum" -gt "$base_quantum" ]; then
		echo "check_wasm_size: RatchetQuantum increased from $base_quantum at $base_ref to $RatchetQuantum." >&2
		echo "  This policy is tighten-only; restore or lower the trusted-base quantum." >&2
		exit 1
	fi
else
	echo "check_wasm_size: no prior wasm budget at $base_ref; applying the bootstrap ceiling"
fi

raw_bytes="$(wc -c <"$wasm" | tr -d ' ')"
brotli_bytes="$(brotli -q 11 -c "$wasm" | wc -c | tr -d ' ')"
gzip_bytes="$(gzip -9 -c "$wasm" | wc -c | tr -d ' ')"

# The ratchet target: the measured build plus its headroom, rounded up to the
# next quantum so small drift does not churn the checked-in ceiling.
target=$(((brotli_bytes + CompressedWasmHeadroom + RatchetQuantum - 1) / RatchetQuantum * RatchetQuantum))

image_version="${ImageVersion:-unrecorded}"
commit="${GITHUB_SHA:-$(git rev-parse HEAD 2>/dev/null || echo unknown)}"

cat >"$record" <<EOF
{
  "raw_bytes": $raw_bytes,
  "brotli_bytes": $brotli_bytes,
  "gzip_bytes": $gzip_bytes,
  "max_compressed_wasm_bytes": $MaxCompressedWasmBytes,
  "compressed_wasm_headroom": $CompressedWasmHeadroom,
  "ratchet_target_bytes": $target,
  "runner_image_version": "$image_version",
  "commit": "$commit"
}
EOF

echo "wasm transfer size: raw=$raw_bytes brotli=$brotli_bytes gzip=$gzip_bytes"
echo "  ceiling=$MaxCompressedWasmBytes headroom=$CompressedWasmHeadroom ratchet target=$target"
echo "  runner image=$image_version"

if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
	{
		echo "### Compressed wasm size"
		echo
		echo "| measurement | bytes |"
		echo "| --- | ---: |"
		echo "| raw | $raw_bytes |"
		echo "| brotli -q 11 (gated) | $brotli_bytes |"
		echo "| gzip -9 | $gzip_bytes |"
		echo "| ceiling | $MaxCompressedWasmBytes |"
		echo "| ratchet target | $target |"
		echo
		echo "Runner image: \`$image_version\`"
	} >>"$GITHUB_STEP_SUMMARY"
fi

status=0
if [ "$brotli_bytes" -gt "$MaxCompressedWasmBytes" ]; then
	over=$((brotli_bytes - MaxCompressedWasmBytes))
	echo "check_wasm_size: FAIL — the build grew past the ceiling by $over bytes." >&2
	echo "  Reclaim the size. Raising MaxCompressedWasmBytes is forbidden by its direction rule." >&2
	status=1
fi

if [ "$MaxCompressedWasmBytes" -gt "$target" ]; then
	slack=$((MaxCompressedWasmBytes - target))
	echo "check_wasm_size: FAIL — the ceiling is stale by $slack bytes and no longer constrains this build." >&2
	echo "  Ratchet it down: set MaxCompressedWasmBytes=$target in tools/wasm_size_budget.env" >&2
	echo "  (measured brotli $brotli_bytes + headroom $CompressedWasmHeadroom, rounded up to $RatchetQuantum)." >&2
	status=1
fi

exit "$status"
