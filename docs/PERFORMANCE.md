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
increase from CI's trusted base revision. The original `5_500_000` ceiling remains only the bootstrap
upper bound for a history with no prior budget.

Measured 2026-08-31 from a full `./build_web.sh --release` build — stripped, trimmed, and
`wasm-opt -O3` optimized:

| measurement            |      bytes |
|------------------------|-----------:|
| raw                    | 17,615,683 |
| `brotli -q 11` (gated) |  3,147,440 |
| `gzip -9`              |  4,380,773 |

**`wasm-opt` is primarily a decompressed-size optimization.** Measuring the same build with and
without the optimizer shows a modest compressed improvement rather than a second-order transfer-size lever:

|                         |        raw |     brotli |      gzip |
|-------------------------|-----------:|-----------:|----------:|
| stripped, no `wasm-opt` | 18,830,163 |  3,129,279 | 4,401,248 |
| `wasm-opt -O3`          | 17,615,683 |  3,147,440 | 4,380,773 |
| change                  |     −6.45% |  **+0.58%** | **−0.47%** |

The optimizer removes about six percent of the decompressed module. Brotli already finds all of
that redundancy and compresses this optimized build slightly worse, while gzip retains a small
benefit. Keep `wasm-opt` for decompressed size and startup; application/dependency reachability
remains the route for transfer-size reductions.

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
`3_650_000` (measured Brotli plus the `500_000` headroom, rounded up to the next `50_000`). The
original ceiling was set from a dependency skeleton rather than from this game and, as §10 puts it,
was "generous enough that it would pass without ever constraining anything."

The ratchet's `50_000` quantum prevents small toolchain and compression drift from churning the
checked-in ceiling. A target computed to the exact byte would turn harmless variation into failures.

## Frame rate

The checked-in `testdata/performance_profile_save.json` is a valid schema-v1 turn-300 save with all
6,144 tiles explored, 256 live bands, and the 96-band archaic sub-cap. The Playwright harness injects
that exact save into the drawing seam while simulation actions and the semantic observer remain on
the ordinary live campaign. Each locked configuration receives a five-second warmup and a 30-second
sample with one successful `EndTurn` every five seconds.

Measured 2026-08-31 from the optimized artifact in pinned Chromium `151.0.7922.34`:

| view | DPR | median FPS | required floor | p95 frame gap | maximum `EndTurn` latency | JS heap |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| top-down | 1 | 72.46 | 30 | 26.2 ms | 100.17 ms | 24.5 MB |
| top-down | 2 | 70.42 | 20 | 26.2 ms | 145.19 ms | 24.5 MB |

Both floors, the 150 ms p95 frame-gap ceiling, and the two-second turn-latency ceiling pass.
The key optimization is appropriate to a turn-based presentation: production disables automatic
screen clearing, caches one complete immutable presentation frame, and leaves the screen untouched
until either the accepted frame or UI-local presentation key changes. This lets Ebitengine skip idle
GPU work rather than continually redrawing an unchanged high-DPI canvas.

### Reference machine identity

The interactive reference machine §8 names is available and its identity is confirmed as an exact
match for the specified configuration:

| property         | required by §8                         | this machine                 |
|------------------|----------------------------------------|------------------------------|
| Model identifier | `Mac16,10` (2024 Mac mini)             | `Mac16,10`                   |
| Chip             | Apple M4, 10-core CPU / integrated GPU | Apple M4, 10 cores (4P + 6E) |
| Memory           | 16 GB                                  | 16 GB                        |
| OS               | macOS 26.6.1                           | macOS 26.6.1 (build 25G76)   |

Browser verification and profiling pin Playwright `1.62.1` and Chromium `151.0.7922.34`.

## Native benchmark baseline

The maximum-workload `World.AdvanceTurn` and frame-projection benchmarks now exercise exactly 6,144
tiles and 256 bands, alongside the fixed pure-Go calibration benchmark. Five-sample medians on the
interactive reference Mac are:

| benchmark | median ratio to calibration | bytes/op | allocs/op |
| --- | ---: | ---: | ---: |
| maximum turn | 1,550 | 7,010,038 | 43,756 |
| maximum frame projection | 134 | 3,635,899 | 2,254 |

The migration-candidate batch changed the maximum frame-projection median on this machine from
approximately `1.470 ms/op`, `3,626,425 B/op`, and `2,253 allocs/op` immediately before the change
to `0.945 ms/op`, `3,635,899 B/op`, and `2,254 allocs/op`: about 36% less elapsed time for one small
outer-batch allocation and 0.3% more bytes. The `ratio_to_calibration` ceiling was consequently
ratcheted from `260` to `165`; the byte and allocation ceilings already remain within 25% of the new
measurement.

The seed-independent canonical-grid cache is measured separately because its cold initialization
happens only once per process and does not belong inside the maximum-turn workload. Three one-second
samples on this machine put a warm `NewWorld` at `438–439 µs/op` and a warm `RestoreWorld` at
`111–112 µs/op`; the latter is the synchronous reconstruction work on the save-load path. These
diagnostic benchmarks are not release gates, but they prevent a future edit from hiding the static
grid rebuild inside ordinary construction again.

`tools/check_benchmarks.sh` repeats the samples and gates normalized time, bytes, and allocations
against `testdata/performance_baseline.json`. The checked-in ceilings are no more than 25% above
these reviewed local medians. The `release-readiness` job repeats the same gate on
`NativeBenchmarkReference`; its GitHub runner image and medians become the authoritative release
record when that job first runs.

The CI reference baseline is recordable because
`NativeBenchmarkReference` is defined as the GitHub-hosted `ubuntu-24.04` `linux/amd64` runner, and
normalising against a calibration benchmark in the same process is precisely what makes a shared
runner a legitimate reference — §8 keeps unnormalized wall-clock time as telemetry to avoid "a gate
that mistakes GitHub-host hardware variation for a game regression". What CI cannot do is the review
step: §12 step 8 admits a baseline only after the workloads themselves have been reviewed for
exactly 6,144 tiles, 256 bands, deterministic inputs, and complete turn/projection work, since "a
fast benchmark that silently omits a subsystem is a test defect, not a performance improvement".
