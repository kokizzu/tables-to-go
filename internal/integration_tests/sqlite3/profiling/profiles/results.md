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
3. If stable, reduce output formatting/decorator overhead in `pkg/output`.
   - Impact: medium; Risk: medium.
4. Reduce file write/open overhead for generated outputs.
   - Impact: low/medium; Risk: medium.
5. Reassess SQLite DB mapping alloc path only after higher-signal rerun.
   - Impact: low/medium; Risk: low.

## Success metrics for follow-up optimizations

- CPU sample volume per run: target >= 200ms sampled CPU.
- Stable hotspot ordering across repeated runs (at least 5 runs).
- End-to-end wall time: target >= 10% reduction after signal improvement.
- `alloc_space` in formatting path: target >= 10% reduction.
- Output correctness: no diff against expected generated files.
