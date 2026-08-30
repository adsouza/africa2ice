# Performance record

DESIGN.md §8 makes performance a measured release property rather than an impression from one
developer machine. This file is where those measurements are recorded, along with the identity of
the machines they were taken on. Each section states what has actually been measured and what has
not, because a partially recorded baseline that reads as complete is worse than an empty one.

## Transfer size

`tools/check_wasm_size.sh` is the gate. It measures `brotli -q 11`, records the raw and `gzip -9`
sizes alongside, and fails in **both** directions: on growth past `MaxCompressedWasmBytes`, and on a
ceiling that has gone stale enough to stop constraining the build. The budget lives in
`tools/wasm_size_budget.env`; §10's direction rule makes it tighten-only, and the script refuses any
value above Appendix C's original `5_500_000` ceiling.

Measured 2026-08-30 from a full `./build_web.sh --release` build — stripped, trimmed, and
`wasm-opt -O3` optimized:

| measurement            |      bytes |
|------------------------|-----------:|
| raw                    | 20,464,212 |
| `brotli -q 11` (gated) |  3,513,110 |
| `gzip -9`              |  5,107,877 |

**`wasm-opt` costs transfer size here rather than saving it.** Measuring the same build with and
without the optimizer settles §2's claim that `wasm-opt -O3` "is not the size lever it looks like":

|                         |        raw |     brotli |      gzip |
|-------------------------|-----------:|-----------:|----------:|
| stripped, no `wasm-opt` | 22,861,922 |  3,502,025 | 4,905,180 |
| `wasm-opt -O3`          | 20,464,212 |  3,513,110 | 5,107,877 |
| change                  |     −10.5% | **+0.32%** | **+4.1%** |

The optimizer removes about a tenth of the decompressed module — which is what it is for, and what
startup cost tracks — while leaving the compressed stream fractionally *larger*, because the
transformations that shrink the module also make it slightly less compressible. Keep `wasm-opt` for
decompressed size and startup; do not expect it to reclaim transfer-size headroom.

**Provenance.** Built with Homebrew Binaryen **132** on darwin/arm64. Appendix C Locks the toolchain
at `version_131` and records a SHA-256 for the Linux x86-64 tarball only, so this is a close
cross-check rather than the release artifact. The release measurement is the one the `web-release`
job takes.

The `web-release` job re-measures the real optimized build on `NativeBenchmarkReference`
(GitHub-hosted `ubuntu-24.04`, `linux/amd64`) with pinned Binaryen `version_131`, uploads the result
as a `wasm-size` artifact — on failure as well as success — and writes it to the job summary along
with the runner's reported image version. If the ceiling below is wrong for the true release build,
that job fails and names the exact replacement value.

`MaxCompressedWasmBytes` was ratcheted from Appendix C's pre-implementation `5_500_000` to
`4_050_000` (measured Brotli plus the `500_000` headroom, rounded up to the next `50_000`). The
original ceiling was set from a dependency skeleton rather than from this game and, as §10 puts it,
was "generous enough that it would pass without ever constraining anything."

The ratchet's `50_000` quantum earned itself immediately: the pre-`wasm-opt` estimate and the real
optimized build differ by 11,085 bytes, and both round to the same ceiling. A target computed to the
exact byte would have failed the gate on that difference alone.

## Frame rate

**Not yet measured.** §8 requires four mandatory profiles — normal and low terrain detail at DPR 1
and DPR 2 — each sustaining its median FPS floor over the 30-second `tools/web-e2e/profile.mjs`
orbit/pan script against the `testdata/performance_profile_save.json` maximum-render fixture.

None of that machinery exists yet: the 3D render layer is build step 8, the profile script and
render fixture are step 13, and neither has been built. There is nothing to measure and no number to
record. Nothing here may be filled in from a development impression.

### Reference machine identity

The interactive reference machine §8 names is available and its identity is confirmed as an exact
match for the specified configuration:

| property         | required by §8                         | this machine                 |
|------------------|----------------------------------------|------------------------------|
| Model identifier | `Mac16,10` (2024 Mac mini)             | `Mac16,10`                   |
| Chip             | Apple M4, 10-core CPU / integrated GPU | Apple M4, 10 cores (4P + 6E) |
| Memory           | 16 GB                                  | 16 GB                        |
| OS               | macOS 26.6.1                           | macOS 26.6.1 (build 25G76)   |

The exact browser version used by a release candidate is recorded here when the first profile is
actually run; the pinned Playwright Chromium the profile depends on is not yet introduced.

## Native benchmark baseline

**Not yet recorded**, because the benchmarks it would summarise do not exist. Build step 8 owns the
maximum-workload `World.AdvanceTurn` and frame-projection benchmarks (6,144 tiles, 256 bands) plus
the fixed pure-Go calibration benchmark, and `tools/check_benchmarks.sh` compares the median
target/calibration ratio and bytes/op against `testdata/performance_baseline.json`.

Note that this baseline **is** recordable in CI once those benchmarks exist:
`NativeBenchmarkReference` is defined as the GitHub-hosted `ubuntu-24.04` `linux/amd64` runner, and
normalising against a calibration benchmark in the same process is precisely what makes a shared
runner a legitimate reference — §8 keeps unnormalized wall-clock time as telemetry to avoid "a gate
that mistakes GitHub-host hardware variation for a game regression". What CI cannot do is the review
step: §12 step 8 admits a baseline only after the workloads themselves have been reviewed for
exactly 6,144 tiles, 256 bands, deterministic inputs, and complete turn/projection work, since "a
fast benchmark that silently omits a subsystem is a test defect, not a performance improvement".
