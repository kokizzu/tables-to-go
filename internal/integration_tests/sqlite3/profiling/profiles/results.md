# Profiling Results (SQLite 3)

## Scope

- Scenario: `TestIntegrationProfiling/sqlite 3`
- Workload: full testdata set (30 tables)
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

1. `runtime.cgocall` (100% cumulative in this sample)
   - Sample duration is very small; CPU profile has low signal.
2. `pkg/output.FileWriter.Write` (100% cumulative path overlap)
   - Visible in call path but dominated by short run noise.
3. `os.WriteFile` (100% cumulative path overlap)
   - Same note: overlap from low sample count.
4. `os.OpenFile` path (100% cumulative path overlap)
   - Same caveat.
5. `internal/cli.(*App).Run` (100% cumulative path overlap)
   - Same caveat.

## Top allocation hotspots (top 5)

1. `golang.org/x/text/transform.String` (14.49% flat)
   - Name casing conversion remains top allocator.
2. `pkg/output` formatting path via `go/format.Source` (17.61% cumulative)
   - Formatting still significant despite smaller table count.
3. `sqlx` scan path (`scanAll`) (11.38% cumulative)
   - Metadata scan remains visible.
4. `internal/cli.(*App).formatColumnName` (11.38% cumulative)
   - Repeated per-column name normalization.
5. `runtime/pprof` profile writing path (up to 12.87% cumulative)
   - Profiling overhead itself is prominent at this short runtime.

## Interpretation

- CPU signal is too short for high-confidence ranking.
- Allocation profile is still useful and consistent with MySQL/Postgres:
  naming conversion + formatting + metadata mapping dominate internal costs.
- For SQLite, run multiple measured iterations in one profile session to
  improve CPU signal quality before optimization decisions.
- Since profiling scenarios run in one test process, treat alloc/heap values as
  scenario-guided rather than perfectly isolated.

## Prioritized optimization candidates (1-5)

1. Re-run SQLite profiling with repeated measured loops to increase CPU signal.
   - Impact: high for confidence; Risk: low.
2. Reduce formatting overhead per generated file.
   - Impact: medium; Risk: medium.
3. Cache repeated casing transformations.
   - Impact: medium; Risk: low.
4. Reduce allocation churn in DB scan/mapping path.
   - Impact: low/medium; Risk: low.
5. Keep concurrency experiments lower priority for SQLite workload size.
   - Impact: low; Risk: medium.

## Success metrics for follow-up optimizations

- Generate a higher-signal CPU profile (target >= 200ms sampled CPU).
- End-to-end profiled run time: target >= 10-15% reduction.
- `alloc_space` in naming + formatting paths: target >= 10% reduction.
- Output correctness: no diff against expected generated files.
