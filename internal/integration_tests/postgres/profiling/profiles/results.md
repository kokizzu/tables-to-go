# Profiling Results (PostgreSQL 18)

## Scope

- Scenario: `TestIntegrationProfiling/postgres 18`
- Workload: full testdata set (519 tables)
- Artifacts: `cpu.pprof`, `heap.pprof`, `allocs.pprof`

## Commands used

```bash
go tool pprof -top -cum -nodecount=30 cpu.pprof
go tool pprof -top -cum -nodecount=30 -focus \
  github.com/fraenky8/tables-to-go/v2 cpu.pprof
go tool pprof -top -cum -sample_index=alloc_space -nodecount=30 -focus \
  github.com/fraenky8/tables-to-go/v2 allocs.pprof
go tool pprof -top -sample_index=inuse_space heap.pprof
```

## Top CPU hotspots (top 5)

1. `runtime.cgocall` (64.29% cumulative)
   - Dominant runtime share at syscall/driver boundary.
2. `pkg/output.FileWriter.Write` (53.57% cumulative)
   - Per-file write path is major CPU consumer.
3. `os.WriteFile` (46.43% cumulative)
   - File writes are expensive at this schema size.
4. `pkg/database.(*Postgresql).GetColumnsOfTable` (35.71% cumulative)
   - DB metadata retrieval is a clear bottleneck.
5. `database/sql.(*Stmt).QueryContext` path (28.57% cumulative)
   - Statement execution and query loop overhead is significant.

## Top allocation hotspots (top 5)

1. `golang.org/x/text/transform.String` (16.55% flat)
   - Name casing conversion has high aggregate allocation cost.
2. `pkg/output` formatting path via `go/format.Source` (20.12% cumulative)
   - Parser/printer path allocates heavily per file.
3. `sqlx` scan path in `GetColumnsOfTable` (13.01% cumulative)
   - Per-table metadata mapping is allocation-heavy.
4. `internal/cli.(*App).formatColumnName` (13.01% cumulative)
   - Column name normalization/conversion contributes notably.
5. `compress/flate.NewWriter` in pprof writing (7.94% cumulative)
   - Profiling artifact generation adds measurable overhead.

## Interpretation

- Postgres profile shows two major runtime centers:
  - DB metadata loop per table
  - output file write/format per table
- Allocation profile reinforces that naming + formatting + scan paths are
  the main internal cost centers.
- Since profiling scenarios run in one test process, treat alloc/heap values as
  scenario-guided rather than perfectly isolated.

## Prioritized optimization candidates (1-5)

1. Batch Postgres column metadata fetch to reduce per-table query overhead.
   - Impact: high; Risk: medium.
2. Reduce per-file format overhead in `pkg/output`.
   - Impact: high; Risk: medium.
3. Cache naming transforms for repeated table/column name operations.
   - Impact: medium; Risk: low.
4. Minimize allocations in DB scan/mapping path (`sqlx` usage patterns).
   - Impact: medium; Risk: low/medium.
5. Evaluate bounded worker pool for generation/writes.
   - Impact: medium/high; Risk: medium/high.

## Success metrics for follow-up optimizations

- End-to-end profiled run time: target >= 20% reduction.
- `GetColumnsOfTable` cumulative CPU: target >= 30% reduction.
- `FileWriter.Write` cumulative CPU: target >= 20% reduction.
- `alloc_space` in `internal/cli` + `pkg/output`: target >= 15% reduction.
- Output correctness: no diff against expected generated files.
