# Profiling Results (MySQL 8)

## Scope

- Scenario: `TestIntegrationProfiling/mysql 8`
- Workload: full testdata set (342 tables)
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

1. `runtime.cgocall` (80.00% cumulative)
   - Dominant cost; mostly syscall/driver boundary work.
2. `pkg/output.FileWriter.Write` (75.00% cumulative)
   - Per-file write path is expensive across many output files.
3. `os.WriteFile` (70.00% cumulative)
   - File I/O dominates generation runtime.
4. `os.OpenFile` path (40.00% cumulative)
   - Open/create cost is substantial due many files.
5. `pkg/database.(*MySQL).GetColumnsOfTable` (20.00% cumulative)
   - Per-table metadata query loop is a visible DB hotspot.

## Top allocation hotspots (top 5)

1. `golang.org/x/text/transform.String` (18.43% flat)
   - Called by casing conversion in naming pipeline.
2. `go/format.Source` pipeline (16.12% cumulative)
   - Includes parser/printer allocations per generated file.
3. `pkg/database.(*MySQL).GetColumnsOfTable` + `sqlx` scan (13.82% cumulative)
   - Metadata scan allocates heavily for large schema runs.
4. `internal/cli.(*App).formatColumnName` (11.52% cumulative)
   - Repeated case conversion and string shaping.
5. `go/parser.(*parser).parseFieldDecl` (9.21% cumulative)
   - Part of formatting overhead in `go/format.Source`.

## Interpretation

- CPU is dominated by many small file writes plus DB metadata calls.
- Memory is dominated by name casing, formatting, and DB scanning.
- This profile gives strong signal for both DB and output stages.
- Since profiling scenarios run in one test process, treat alloc/heap values as
  scenario-guided rather than perfectly isolated.

## Prioritized optimization candidates (1-5)

1. Batch column metadata retrieval for all selected tables in one query.
   - Impact: high; Risk: medium.
2. Reduce per-file format overhead in `pkg/output`.
   - Impact: high; Risk: medium.
3. Cache repeated casing/name transformations in `internal/cli`.
   - Impact: medium; Risk: low.
4. Reduce scan/mapping allocations in DB metadata fetch path.
   - Impact: medium; Risk: low/medium.
5. Evaluate bounded concurrency for independent table generation/writes.
   - Impact: medium/high; Risk: medium/high.

## Success metrics for follow-up optimizations

- End-to-end profiled run time: target >= 20% reduction.
- `GetColumnsOfTable` cumulative CPU: target >= 30% reduction.
- `FileWriter.Write` cumulative CPU: target >= 20% reduction.
- `alloc_space` under `internal/cli` + `pkg/output`: target >= 15% reduction.
- Output correctness: no diff against expected generated files.
