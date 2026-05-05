# Profiling Results (PostgreSQL 18)

## Scope

- Scenario: `TestIntegrationProfiling/postgres 18`
- Workload: full testdata set (519 tables)
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
  - changed representative: `run-01`
- Diff check used changed profile with baseline as `-base`.

## Run-time comparison (baseline vs changed)

- Median wall time: `6s` -> `5s` (**-16.67%**)
- Average wall time: `5.571s` -> `4.429s` (**-20.50%**)
- Range:
  - baseline: `5s..6s`
  - changed: `4s..5s`

## Top 5 CPU hotspots (baseline)

1. `pkg/database.(*Postgresql).GetColumnsOfTable`
2. stmt/query path (`database/sql.(*Stmt).QueryContext`, `lib/pq` stmt path)
3. `pkg/output.FileWriter.Write` / `os.WriteFile`
4. `runtime.cgocall` / syscall boundary
5. `os.OpenFile` path

## Top 5 allocation hotspots (baseline)

1. `golang.org/x/text/transform.String`
2. `pkg/output` format/decorator path (`go/format.Source`)
3. `pkg/database.(*Postgresql).GetColumnsOfTable` + `sqlx` scan
4. `internal/cli.(*App).formatColumnName`
5. parser/printer allocations under formatting path

## Top 5 CPU hotspots (changed)

1. `pkg/output.FileWriter.Write` / `os.WriteFile`
2. `runtime.cgocall` / syscall boundary
3. `os.OpenFile` path
4. output decorator/format path (`go/format.Source`)
5. remaining write/close syscall path

## Top 5 allocation hotspots (changed)

1. `golang.org/x/text/transform.String`
2. `internal/cli` naming path (`formatColumnName`, `camelCaseString`)
3. `pkg/output` format/decorator path (`go/format.Source`)
4. `pkg/database.(*Postgresql).GetColumnsOfTables` + `sqlx` scan
5. `pkg/database.attachPostgresqlColumnsToTables`

## Priority shift (baseline -> changed)

- CPU: dominant DB bottleneck shifted away from per-table singular query path;
  output write/format now dominates.
- Alloc: singular DB query alloc reduced; bulk mapping alloc introduced but
  overall representative alloc is lower.
- Overall: strongest wall-time win among DBs with clear hotspot reordering.

## Prioritized optimization candidates (1-5)

1. Reduce output formatting/decorator overhead in `pkg/output`.
   - Impact: high; Risk: medium.
2. Reduce file I/O overhead (open/write path) for generated files.
   - Impact: high; Risk: medium.
3. Cache repeated naming/casing transforms.
   - Impact: medium; Risk: low.
4. Reduce allocation in `attachPostgresqlColumnsToTables`.
   - Impact: medium; Risk: low/medium.
5. Tune bulk DB scan/mapping memory structures.
   - Impact: medium; Risk: medium.

## Success metrics for follow-up optimizations

- End-to-end wall time: additional >= 10% reduction from changed baseline.
- `FileWriter.Write` cumulative CPU: >= 20% reduction.
- `alloc_space` in `pkg/output` + `internal/cli`: >= 15% reduction.
- `alloc_space` in `attachPostgresqlColumnsToTables`: >= 20% reduction.
- Output correctness: no diff against expected generated files.
