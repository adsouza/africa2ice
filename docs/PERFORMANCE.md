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

Measured 2026-09-01 by the `web-release` job on `NativeBenchmarkReference` from a full
`./build_web.sh --release` build — stripped, trimmed, and `wasm-opt -O3` optimized:

| measurement            |      bytes |
|------------------------|-----------:|
| raw                    | 16,889,516 |
| `brotli -q 11` (gated) |  3,059,373 |
| `gzip -9`              |  4,226,859 |

The recorded GitHub-hosted runner image was `20260823.283.1`.

**`wasm-opt` is primarily a decompressed-size optimization.** Measuring the same build with and
without the optimizer on the local cross-check machine shows a mixed, roughly one-percent
compressed effect rather than a second-order transfer-size lever:

|                         |        raw |     brotli |      gzip |
|-------------------------|-----------:|-----------:|----------:|
| stripped, no `wasm-opt` | 19,110,814 |  3,194,787 | 4,469,521 |
| `wasm-opt -O3`          | 17,877,252 |  3,209,520 | 4,450,862 |
| change                  |     −6.45% |  **+0.46%** | **−0.42%** |

The optimizer removes about six percent of the decompressed module. Brotli already finds all of
that redundancy and compresses this optimized build slightly worse, while gzip retains a small
benefit. Keep `wasm-opt` for decompressed size and startup; application/dependency reachability
remains the route for transfer-size reductions.

**Provenance.** Built with Homebrew Binaryen **132** on darwin/arm64. Appendix C locks the toolchain
at `version_132` and records a SHA-256 for the Linux x86-64 tarball. This local measurement uses the
same release but remains a cross-check rather than the release artifact. The release measurement is
the one the `web-release` job takes.

The `web-release` job re-measures the real optimized build on `NativeBenchmarkReference`
(GitHub-hosted `ubuntu-24.04`, `linux/amd64`) with pinned Binaryen `version_132`, uploads the result
as a `wasm-size` artifact — on failure as well as success — and writes it to the job summary along
with the runner's reported image version. If the ceiling below is wrong for the true release build,
that job fails and names the exact replacement value.

`MaxCompressedWasmBytes` was ratcheted from Appendix C's pre-implementation `5_500_000` to
`3_600_000` (measured Brotli plus the `500_000` headroom, rounded up to the next `50_000`). The
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

Measured 2026-09-01 from the optimized artifact in pinned Chromium `151.0.7922.34`:

| view | DPR | median FPS | required floor | p95 frame gap | maximum `EndTurn` latency | JS heap |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| top-down | 1 | 59.88 | 30 | 16.8 ms | 68.71 ms | 26.0 MB |
| top-down | 2 | 59.88 | 20 | 16.8 ms | 212.36 ms | 24.5 MB |

Both floors, the 150 ms p95 frame-gap ceiling, and the two-second turn-latency ceiling pass.
The key optimization is appropriate to a turn-based presentation: production disables automatic
screen clearing, caches one complete immutable presentation frame, and leaves the screen untouched
until either the accepted frame or UI-local presentation key changes. This lets Ebitengine skip idle
GPU work rather than continually redrawing an unchanged high-DPI canvas. The DPR 2 result above uses
native 2560 × 1440 presentation and scale-specific terrain targets; it is not an upscale of a
completed 1280 × 720 frame.

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

The maximum-workload `World.AdvanceTurn` and frame-projection benchmarks exercise exactly 6,144
tiles and 256 bands, alongside the fixed pure-Go calibration benchmark.

**The ratios below are provisional.** `BenchmarkCalibration` was replaced on 2026-09-05 (see
"Why the calibration allocates"), so every previously recorded ratio is expressed in a unit that no
longer exists and none of them carry over. These medians are five samples on the interactive
reference Mac; §12 step 8 admits a baseline only from `NativeBenchmarkReference`, so the next
successful `release-readiness` run supplies the authoritative record and the ceilings must then be
tightened to it.

| benchmark | provisional median ratio | bytes/op | allocs/op |
| --- | ---: | ---: | ---: |
| maximum turn | 404.20 | 1,881,977 | 3,244 |
| maximum frame projection | 77.43 | 3,777,561 | 1,998 |

These medians include the current reusable migration-candidate workspace and seed-independent
world data. The memory ceilings are `2,350,000 B/op` and `4,500,000 B/op`; the allocation ceilings
are `4,080` and `2,490`; those are counts rather than times and are unaffected by the calibration
change. The provisional normalized-time ceilings are `810` and `155` — twice the medians above
rather than the usual 25% margin, because a Mac-derived median cannot predict the reference
runner's. Twice is still tight enough to catch the regression class that prompted the change: the
per-tile band scan removed in `cb5296d` cost 2.56x the current maximum turn. Restore the 25% margin
against the CI median once one is recorded.

### Why the calibration allocates

`BenchmarkCalibration` was a cache-resident integer loop with zero allocations, sized by
`domain.TileCount`. Both properties were defects in a unit of measurement. Sizing it by a game
constant meant a map-size change would move the denominator and silently rescale every recorded
ratio; it now uses frozen literals and imports nothing from the game, as this section always
required.

The larger problem is that the gated workloads are allocator-, collector- and
memory-bound — roughly 1.9 MB and 3,200 allocations for a maximum turn — so a benchmark that
allocated nothing could not track them across machines, and the ratio failed to normalise the one
thing it exists to normalise. Two `ubuntu-24.04` runs of the same commit on 2026-09-05 disagreed
by nearly a factor of two: an Intel Xeon 6973P-C reported a maximum-turn ratio of `1,949` and an
AMD EPYC 9V74 `3,840`, failing the release lane on hardware alone. The calibration had slowed
`1.44x` between those runners where the maximum turn slowed `2.84x`. The same mismatch explains the
older bootstrap discrepancy recorded here: `1,900`/`165` were derived from the interactive Mac and
could not describe `NativeBenchmarkReference`, which read `2,368.25` and `2,367.88` for identical
code — a `1.59x` spread that was treated as a machine difference to be re-baselined rather than a
sign that the normaliser did not work.

The replacement allocates 96 blocks of 576 bytes per operation, putting its mean allocation within
a few percent of the maximum turn's own. Against a deliberate change in allocator and collector
cost (`GOGC=off`) the old normaliser's ratio moved `+58.4%` while the new one moved `-22.9%`,
because the old loop absorbed almost none of the change it was supposed to cancel: it went from
`9,975 ns` to `10,369 ns`, `+4%`, while the workload it normalises nearly doubled.

Note that `NativeBenchmarkReference` pins the runner image and architecture but no longer implies
fixed hardware — EPYC 7763, EPYC 9V74 and Xeon 6973P-C have all served it. Pinning the image is
therefore not sufficient on its own, and the calibration has to carry the normalisation.

The seed-independent canonical-grid cache is measured separately because its cold initialization
happens only once per process and does not belong inside the maximum-turn workload. Three one-second
samples on this machine put a warm `NewWorld` at `438–439 µs/op` and a warm `RestoreWorld` at
`111–112 µs/op`; the latter is the synchronous reconstruction work on the save-load path. These
diagnostic benchmarks are not release gates, but they prevent a future edit from hiding the static
grid rebuild inside ordinary construction again.

`tools/check_benchmarks.sh` repeats the samples and gates normalized time, bytes, and allocations
against `testdata/performance_baseline.json`. The `release-readiness` job repeats the same gate on
`NativeBenchmarkReference`; its GitHub runner image and medians above are the authoritative release
record.

The CI reference baseline is recordable because
`NativeBenchmarkReference` is defined as the GitHub-hosted `ubuntu-24.04` `linux/amd64` runner, and
normalising against a calibration benchmark in the same process is precisely what makes a shared
runner a legitimate reference — §8 keeps unnormalized wall-clock time as telemetry to avoid "a gate
that mistakes GitHub-host hardware variation for a game regression". What CI cannot do is the review
step: §12 step 8 admits a baseline only after the workloads themselves have been reviewed for
exactly 6,144 tiles, 256 bands, deterministic inputs, and complete turn/projection work, since "a
fast benchmark that silently omits a subsystem is a test defect, not a performance improvement".
