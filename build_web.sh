#!/bin/sh
set -eu

mode="${1:---dev}"

goroot="$(go env GOROOT)"
cp "${goroot}/lib/wasm/wasm_exec.js" web/wasm_exec.js

# Go's js/wasm output uses post-MVP WebAssembly features that wasm-opt must be
# told to accept, or it rejects the module outright before optimizing anything:
#
#   bulk-memory               memory.init / data.drop
#   bulk-memory-opt           memory.copy / memory.fill, which Binaryen splits
#                             out from the base bulk-memory feature
#   nontrapping-float-to-int  i64.trunc_sat_* saturating conversions
#
# The list is empirically the complete set for this module: with all three the
# validator reports no errors, and removing any one of them fails the build.
set -- --enable-bulk-memory --enable-bulk-memory-opt --enable-nontrapping-float-to-int

case "${mode}" in
--dev)
	# Remove the previous artifact first so that a failed build leaves no
	# stale module behind for a local server to serve as if it were current.
	rm -f web/main.wasm
	GOOS=js GOARCH=wasm go build -o web/main.wasm .
	;;
--release)
	command -v wasm-opt >/dev/null 2>&1 || {
		echo "wasm-opt is required for --release" >&2
		exit 1
	}
	# Fail with the flag's name rather than with a wall of validator output if
	# the available Binaryen predates one of these feature gates.
	for feature in "$@"; do
		wasm-opt --help 2>&1 | grep -qE -- "${feature}([[:space:]]|$)" || {
			echo "this wasm-opt does not support ${feature}; use the pinned Binaryen release" >&2
			exit 1
		}
	done
	rm -f web/main.wasm web/main.unoptimized.wasm
	GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o web/main.unoptimized.wasm .
	wasm-opt -O3 "$@" web/main.unoptimized.wasm -o web/main.wasm
	rm web/main.unoptimized.wasm
	;;
*)
	echo "usage: ./build_web.sh [--dev|--release]" >&2
	exit 2
	;;
esac
