#!/bin/sh
set -eu

mode="${1:---dev}"
goroot="$(go env GOROOT)"
cp "${goroot}/lib/wasm/wasm_exec.js" web/wasm_exec.js

case "${mode}" in
  --dev)
    GOOS=js GOARCH=wasm go build -o web/main.wasm .
    ;;
  --release)
    GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o web/main.unoptimized.wasm .
    command -v wasm-opt >/dev/null 2>&1 || { echo "wasm-opt is required for --release" >&2; exit 1; }
    wasm-opt -O3 web/main.unoptimized.wasm -o web/main.wasm
    rm web/main.unoptimized.wasm
    ;;
  *)
    echo "usage: ./build_web.sh [--dev|--release]" >&2
    exit 2
    ;;
esac
