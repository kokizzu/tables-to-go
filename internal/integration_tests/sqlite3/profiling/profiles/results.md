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
2. `pkg/output.FileWriter.Write`
3. `os.WriteFile`
4. `os.OpenFile` / open path overlap
5. top-level run path overlap (`internal/cli.(*App).Run`)

## Top 5 allocation hotspots (changed, low confidence)

1. `runtime/pprof.StartCPUProfile` (profiling overhead)
2. `pkg/output` formatting path (`go/format.Source`)
3. printer allocations in format path (`go/printer` internals)
4. `internal/cli` struct generation path
5. remaining profiling/writer support allocations

## Priority shift (baseline -> changed)

- CPU: no reliable reordering; sample size remains too small (typically ~10ms).
- Alloc: changed representative run shows more formatter/profiler overhead,
  but confidence is low due short runtime and overhead proportion.
- Overall: optimization priorities remain measurement-first for SQLite.

## Prioritized optimization candidates (1-5)

1. Increase SQLite measured work per profiling run for higher-signal CPU data.
   - Impact: high (confidence); Risk: low.
2. Re-profile and re-rank hotspots after signal quality improves.
   - Impact: high (decision quality); Risk: low.
3. If signal stabilizes, reduce output formatting/decorator overhead in `pkg/output`.
   - Impact: medium; Risk: medium.
4. Reduce file write/open overhead for generated outputs.
   - Impact: low/medium; Risk: medium.
5. Defer SQLite DB-specific mapping optimizations until a stable hotspot appears.
   - Impact: medium (focus); Risk: low.

## Success metrics for follow-up optimizations

- CPU sample volume per run: target >= 200ms sampled CPU.
- Stable hotspot ordering across repeated runs (at least 5 runs).
- End-to-end wall time: target >= 10% reduction after signal improvement.
- `alloc_space` in formatting path: target >= 10% reduction.
- Output correctness: no diff against expected generated files.

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
