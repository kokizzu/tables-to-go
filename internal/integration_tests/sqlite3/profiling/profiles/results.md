# Profiling Results (SQLite 3)

## Scope

- Scenario: `TestIntegrationProfiling/sqlite 3`
- Workload: full testdata set (30 tables)
- Comparison mode: repeated isolated runs (7 baseline + 7 changed)
- Batches:
  - baseline: `runs/baseline/20260505-173911`
  - changed: `runs/changed/20260505-174330`
- Artifacts per run: `cpu.pprof`, `heap.pprof`, `allocs.pprof`

## Method

- Collected runs via `internal/integration_tests/collect-profiling-runs.sh` with
  `-count=1`, one scenario per test invocation.
- Compared wall time using median + average from `summary.tsv`.
- Representative profiles used for hotspot comparison:
  - baseline representative: `run-04` (2s run)
  - changed representative: `run-05` (2s run)
- Diff check used changed profile with baseline as `-base`.

## Run-time comparison (baseline vs changed)

- Median wall time: `1s` -> `1s` (**0%**)
- Average wall time: `1.286s` -> `1.143s` (**-11.11%**)
- Range:
  - baseline: `1s..2s`
  - changed: `1s..2s`

## Top 5 CPU hotspots (baseline, low confidence)

1. `runtime.cgocall`
2. `pkg/output.FileWriter.Write`
3. `os.WriteFile`
4. `os.OpenFile` / open path overlap
5. top-level run path overlap (`internal/cli.(*App).Run`)

## Top 5 allocation hotspots (baseline, low confidence)

1. `runtime/pprof.StartCPUProfile` (profiling overhead)
2. `runtime/pprof` artifact emission path
3. `pkg/database.(*SQLite).GetColumnsOfTable` + `sqlx` scan
4. `pkg/output` formatting path (`go/format.Source`)
5. directory/glob loading path during fixture setup

## Top 5 CPU hotspots (changed, low confidence)

1. `runtime.cgocall`
2. `pkg/output.FileWriter.Write` cumulative write/open path
3. `os.OpenFile`/`os.WriteFile` overlap
4. top-level run path overlap (`internal/cli.(*App).Run`)
5. no stable fifth hotspot beyond syscall noise (tiny 10ms sample)

## Top 5 allocation hotspots (changed, low confidence)

1. `runtime/pprof.StartCPUProfile` (profiling overhead)
2. compression/profiling writer path (`compress/flate.newDeflateFast`, `compress/flate.NewWriter`)
3. regex/parser support (`regexp/syntax.(*compiler).inst`)
4. generic allocator and syscall/string conversion (`runtime.mallocgc`, `syscall.UTF16FromString`)
5. residual formatting/transform support (`golang.org/x/text/transform.String`)

## Priority shift (baseline -> changed)

- CPU: no reliable reordering; sample size remains too small (typically ~10ms).
- Alloc: changed representative run is still dominated by profiler/compression/runtime overhead, with no stable SQLite-specific application hotspot.
- Overall: latest changed batch trends slower than earlier pre-pooling changed batches, but confidence remains low due coarse 1s/2s granularity.

## Prioritized optimization candidates (1-5)

1. Increase SQLite measured work per profiling run for higher-signal CPU data.
   - Impact: high (confidence); Risk: low.
2. Re-profile and re-rank hotspots after signal quality improves.
   - Impact: high (decision quality); Risk: low.
3. If signal stabilizes, reduce output formatting/decorator overhead in `pkg/output`.
   - Impact: medium; Risk: medium.
4. Reduce profiling artifact overhead in measurement harness where possible (without changing scenario semantics).
   - Impact: low/medium; Risk: medium.
5. Defer SQLite DB-specific mapping optimizations until a stable non-overhead hotspot appears.
   - Impact: medium (focus); Risk: low.

## Update: Changed batch `20260505-190911`

SQLite adapter did not receive a comparable stdlib scan streaming change (it was
already per-table and non-StructScan in the critical path), but the latest run
set is included for completeness.

### Run-time comparison (latest changed)

- Latest changed (`20260505-190911`) median: `1s`
- Latest changed (`20260505-190911`) average: `1.429s`

Compared with baseline (`20260505-173911`):

- Median: `1s` -> `1s` (**0%**)
- Average: `1.286s` -> `1.429s` (**worse**)

Compared with previous changed (`20260505-174330`):

- Median: `1s` -> `1s` (**no change**)
- Average: `1.143s` -> `1.429s` (**worse**)

### Verdict for this change window

- Still **inconclusive / low confidence** for SQLite due very short runtimes and
  noisy profile samples.
- No actionable optimization conclusion changes for SQLite from this batch.

## Update: Changed batch `20260506-093308`

### Run-time comparison (latest changed)

- Latest changed (`20260506-093308`) median: `1s`
- Latest changed (`20260506-093308`) average: `1.143s`

Compared with baseline (`20260505-173911`):

- Median: `1s` -> `1s` (**0%**)
- Average: `1.286s` -> `1.143s` (**-11.11%**)

Compared with previous changed (`20260505-174330`):

- Median: `1s` -> `1s` (**no change**)
- Average: `1.143s` -> `1.143s` (**no change**)

Compared with stashed changed (`20260505-190911`):

- Median: `1s` -> `1s` (**no change**)
- Average: `1.429s` -> `1.143s` (**-20.01%**)

Compared with changed (`20260505-193236`):

- Median: `1s` -> `1s` (**no change**)
- Average: `1.000s` -> `1.143s` (**+14.30%**, slower)

### Hotspot impact check

- CPU sample remains tiny (`10ms`) and fully dominated by `runtime.cgocall`.
- Allocation profile is still mostly profiler/runtime/output-path noise.

### Verdict for this batch

- Directionally similar to `20260505-174330` and better than `20260505-190911`
  on average.
- Still low-confidence for optimization decisions due coarse wall-time granularity
  and minimal CPU sample volume.

## Update: Changed batch `20260506-100845`

### Run-time comparison (latest changed)

- Latest changed (`20260506-100845`) median: `1s`
- Latest changed (`20260506-100845`) average: `1.143s`

Compared with baseline (`20260505-173911`):

- Median: `1s` -> `1s` (**0%**)
- Average: `1.286s` -> `1.143s` (**-11.11%**)

Compared with previous changed (`20260505-174330`):

- Median: `1s` -> `1s` (**no change**)
- Average: `1.143s` -> `1.143s` (**no change**)

Compared with stashed changed (`20260505-190911`):

- Median: `1s` -> `1s` (**no change**)
- Average: `1.429s` -> `1.143s` (**-20.01%**)

Compared with changed (`20260505-193236`):

- Median: `1s` -> `1s` (**no change**)
- Average: `1.000s` -> `1.143s` (**+14.30%**, slower)

Compared with changed (`20260506-093308`):

- Median: `1s` -> `1s` (**no change**)
- Average: `1.143s` -> `1.143s` (**no change**)

### Hotspot impact check

- CPU sample remains tiny (`10ms`) and fully dominated by `runtime.cgocall`.
- Allocation profile remains mostly profiler/runtime noise.

### Verdict for this batch

- Essentially identical to `20260506-093308` and `20260505-174330` at summary
  level.
- Still insufficient signal for new SQLite-specific optimization conclusions.

## Update: Changed batch `20260506-104110`

### Run-time comparison (latest changed)

- Latest changed (`20260506-104110`) median: `2s`
- Latest changed (`20260506-104110`) average: `1.571s`

Compared with baseline (`20260505-173911`):

- Median: `1s` -> `2s` (**+100%**, slower)
- Average: `1.286s` -> `1.571s` (**+22.16%**, slower)

Compared with previous changed (`20260505-174330`):

- Median: `1s` -> `2s` (**+100%**, slower)
- Average: `1.143s` -> `1.571s` (**+37.45%**, slower)

Compared with stashed changed (`20260505-190911`):

- Median: `1s` -> `2s` (**+100%**, slower)
- Average: `1.429s` -> `1.571s` (**+9.94%**, slower)

Compared with changed (`20260505-193236`):

- Median: `1s` -> `2s` (**+100%**, slower)
- Average: `1.000s` -> `1.571s` (**+57.10%**, slower)

Compared with changed (`20260506-093308`):

- Median: `1s` -> `2s` (**+100%**, slower)
- Average: `1.143s` -> `1.571s` (**+37.45%**, slower)

Compared with changed (`20260506-100845`):

- Median: `1s` -> `2s` (**+100%**, slower)
- Average: `1.143s` -> `1.571s` (**+37.45%**, slower)

### Hotspot impact check

- CPU sample remains tiny (`10ms`) and fully dominated by `runtime.cgocall`.
- Allocation profile remains dominated by profiler/runtime overhead with no
  stable SQLite-specific hotspot shift.

### Verdict for this batch

- This run set is significantly slower on summary metrics, but SQLite signal
  quality is still too low for high-confidence attribution.
- Treat as noisy/unfavorable sample and keep SQLite optimization decisions
  measurement-first.

## Update: Changed batch `20260508-095608`

### Run-time comparison (latest changed)

- Latest changed (`20260508-095608`) median: `1s`
- Latest changed (`20260508-095608`) average: `1.429s`

Compared with baseline (`20260505-173911`):

- Median: `1s` -> `1s` (**0%**)
- Average: `1.286s` -> `1.429s` (**+11.12%**, slower)

Compared with previous documented changed (`20260506-104110`):

- Median: `2s` -> `1s` (**improved**)
- Average: `1.571s` -> `1.429s` (**-9.04%**)

Compared with latest measured changed (`20260507-123350`):

- Median: `2s` -> `1s` (**improved**)
- Average: `1.571s` -> `1.429s` (**-9.04%**)

Compared with stronger changed band (`20260505-174330` / `20260506-093308` / `20260506-100845`):

- Median: `1s` -> `1s` (**no change**)
- Average: `1.143s` -> `1.429s` (**+25.02%**, slower)

### Hotspot impact check

- CPU sample is still tiny (`10ms`) and fully dominated by `runtime.cgocall`.
- Allocation profile remains heavily influenced by profiler/runtime/compression overhead.
- No stable SQLite-specific hotspot reordering is visible from this batch.

### Verdict for this batch

- Better than the two immediately previous unfavorable SQLite batches, but still not back to the stronger `1.143s` changed band.
- Signal quality remains low; treat this as directional only, not a high-confidence optimization verdict.
