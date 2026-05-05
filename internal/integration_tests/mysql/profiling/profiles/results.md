# Profiling Results (MySQL 8)

## Scope

- Scenario: `TestIntegrationProfiling/mysql 8`
- Workload: full testdata set (342 tables)
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
  - baseline representative: `run-01`
  - changed representative: `run-02`
- Diff check used changed profile with baseline as `-base`.

## Run-time comparison (baseline vs changed)

- Median wall time: `12s` -> `11s` (**-8.33%**)
- Average wall time: `11.857s` -> `10.714s` (**-9.64%**)
- Range:
  - baseline: `11s..13s`
  - changed: `10s..12s`

## Top 5 CPU hotspots (baseline)

1. `pkg/output.FileWriter.Write` / `os.WriteFile`
2. `runtime.cgocall` / syscall boundary
3. `os.OpenFile` path
4. `pkg/database.(*MySQL).GetColumnsOfTable`
5. `go/format.Source` via output decorators

## Top 5 allocation hotspots (baseline)

1. `golang.org/x/text/transform.String`
2. `pkg/output` format/decorator path (`go/format.Source`)
3. `pkg/database.(*MySQL).GetColumnsOfTable` + `sqlx` scan
4. `internal/cli.(*App).formatColumnName`
5. parser/printer allocations under formatting path

## Top 5 CPU hotspots (changed)

1. `pkg/output.FileWriter.Write` / `os.WriteFile`
2. `runtime.cgocall` / syscall boundary
3. `os.OpenFile` path
4. output decorator + formatting path (`go/format.Source`)
5. remaining file close/write syscall path

## Top 5 allocation hotspots (changed)

1. `golang.org/x/text/transform.String`
2. `pkg/output` format/decorator path (`go/format.Source`)
3. `pkg/database.(*MySQL).GetColumnsOfTables`
4. `pkg/database.attachMySQLColumnsToTables`
5. `internal/cli` naming path (`formatColumnName`, `camelCaseString`)

## Priority shift (baseline -> changed)

- CPU: per-table DB metadata hotspot (`GetColumnsOfTable`) dropped out of top
  CPU list; output write/format is now dominant.
- Alloc: DB allocation moved from singular query loop to bulk mapping helpers.
- Overall: wall-time improved with a modest memory/allocation tradeoff.

## Prioritized optimization candidates (1-5)

1. Reduce output formatting/decorator overhead in `pkg/output`.
   - Impact: high; Risk: medium.
2. Reduce file I/O overhead (open/write churn) in write path.
   - Impact: high; Risk: medium.
3. Cache repeated casing and naming transforms in `internal/cli`.
   - Impact: medium; Risk: low.
4. Reduce allocation churn in `attachMySQLColumnsToTables`.
   - Impact: medium; Risk: low/medium.
5. Tune bulk metadata scan/mapping structures for lower alloc footprint.
   - Impact: medium; Risk: medium.

## Success metrics for follow-up optimizations

- End-to-end wall time: additional >= 10% reduction from changed baseline.
- `FileWriter.Write` cumulative CPU: >= 20% reduction.
- `alloc_space` in `pkg/output` + `internal/cli`: >= 15% reduction.
- `alloc_space` in `attachMySQLColumnsToTables`: >= 20% reduction.
- Output correctness: no diff against expected generated files.
