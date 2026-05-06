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
4. Reduce DB scan-path overhead in `GetColumnsOfTables` (`database/sql` arg/scan churn).
   - Impact: low/medium; Risk: low.
5. Measure output-only optimizations first, then re-rank DB-side work.
   - Impact: medium (decision quality); Risk: low.

## Success metrics for follow-up optimizations

- End-to-end wall time: additional >= 10% reduction from changed baseline.
- `FileWriter.Write` cumulative CPU: >= 20% reduction.
- `alloc_space` in `pkg/output` + `internal/cli`: >= 15% reduction.
- `alloc_space` in `attachMySQLColumnsToTables`: >= 20% reduction.
- Output correctness: no diff against expected generated files.

## Update: Changed batch `20260505-190911` (stdlib `rows.Scan` streaming)

This batch reflects the follow-up change that removed `StructScan` and switched
to stdlib row scanning while streaming rows directly into `Table.Columns`.

### Run-time comparison (latest changed)

- Latest changed (`20260505-190911`) median: `12s`
- Latest changed (`20260505-190911`) average: `11.714s`

Compared with baseline (`20260505-173911`):

- Median: `12s` -> `12s` (**0%**)
- Average: `11.857s` -> `11.714s` (**-1.20%**)

Compared with previous changed (`20260505-174330`):

- Median: `11s` -> `12s` (**regression**)
- Average: `10.714s` -> `11.714s` (**regression**)

### Hotspot impact check

- CPU remains dominated by output write path (`FileWriter.Write`, `os.WriteFile`,
  `runtime.cgocall`).
- Old singular DB hotspot remains absent.
- Allocation profile no longer shows `attachMySQLColumnsToTables` as a top node,
  which is expected after direct streaming attach.

### Verdict for this change

- **No clear end-to-end performance gain** from the stdlib scan streaming change
  on MySQL in this run set.
- It simplified DB mapping internals and removed one mapping hotspot, but wall
  time did not improve versus the earlier changed batch.

## Update: Changed batch `20260506-093308`

This batch reflects the current direct `rows.Scan` path after dropping the
over-allocation prealloc experiment.

### Run-time comparison (latest changed)

- Latest changed (`20260506-093308`) median: `11s`
- Latest changed (`20260506-093308`) average: `11.000s`

Compared with baseline (`20260505-173911`):

- Median: `12s` -> `11s` (**-8.33%**)
- Average: `11.857s` -> `11.000s` (**-7.23%**)

Compared with previous changed (`20260505-174330`):

- Median: `11s` -> `11s` (**no change**)
- Average: `10.714s` -> `11.000s` (**+2.67%**, slower)

Compared with stashed changed (`20260505-190911`):

- Median: `12s` -> `11s` (**improved**)
- Average: `11.714s` -> `11.000s` (**-6.10%**)

Compared with changed (`20260505-193236`):

- Median: `12s` -> `11s` (**improved**)
- Average: `11.857s` -> `11.000s` (**-7.23%**)

### Hotspot impact check

- CPU remains dominated by syscall/output path (`runtime.cgocall`).
- Allocation representative total is lower than baseline representative:
  `21.58MB` -> `18.49MB`.
- The prior `attachMySQLColumnsToTables` alloc blow-up from `20260505-193236`
  is no longer present as a top hotspot.

### Verdict for this batch

- Restores a clear MySQL improvement over baseline.
- Slightly slower than the best earlier changed batch (`20260505-174330`) on
  average, but in the same general performance band.
- Allocation behavior is materially healthier than `20260505-193236`.

## Update: Changed batch `20260506-100845`

### Run-time comparison (latest changed)

- Latest changed (`20260506-100845`) median: `11s`
- Latest changed (`20260506-100845`) average: `11.143s`

Compared with baseline (`20260505-173911`):

- Median: `12s` -> `11s` (**-8.33%**)
- Average: `11.857s` -> `11.143s` (**-6.02%**)

Compared with previous changed (`20260505-174330`):

- Median: `11s` -> `11s` (**no change**)
- Average: `10.714s` -> `11.143s` (**+4.00%**, slower)

Compared with stashed changed (`20260505-190911`):

- Median: `12s` -> `11s` (**improved**)
- Average: `11.714s` -> `11.143s` (**-4.87%**)

Compared with changed (`20260505-193236`):

- Median: `12s` -> `11s` (**improved**)
- Average: `11.857s` -> `11.143s` (**-6.02%**)

Compared with changed (`20260506-093308`):

- Median: `11s` -> `11s` (**no change**)
- Average: `11.000s` -> `11.143s` (**+1.30%**, slower)

### Hotspot impact check

- CPU remains dominated by syscall/output path (`runtime.cgocall`).
- Allocation representative total remains below baseline representative:
  `21.58MB` -> `15.39MB`.
- No recurrence of `attachMySQLColumnsToTables` allocation blow-up.

### Verdict for this batch

- Maintains a solid MySQL gain over baseline.
- Slightly regresses versus `20260506-093308` and clearly trails best batch
  `20260505-174330` on average.
- Allocation profile remains healthy and better than baseline.

## Update: Changed batch `20260506-104110`

### Run-time comparison (latest changed)

- Latest changed (`20260506-104110`) median: `11s`
- Latest changed (`20260506-104110`) average: `11.286s`

Compared with baseline (`20260505-173911`):

- Median: `12s` -> `11s` (**-8.33%**)
- Average: `11.857s` -> `11.286s` (**-4.82%**)

Compared with previous changed (`20260505-174330`):

- Median: `11s` -> `11s` (**no change**)
- Average: `10.714s` -> `11.286s` (**+5.34%**, slower)

Compared with stashed changed (`20260505-190911`):

- Median: `12s` -> `11s` (**improved**)
- Average: `11.714s` -> `11.286s` (**-3.65%**)

Compared with changed (`20260505-193236`):

- Median: `12s` -> `11s` (**improved**)
- Average: `11.857s` -> `11.286s` (**-4.82%**)

Compared with changed (`20260506-093308`):

- Median: `11s` -> `11s` (**no change**)
- Average: `11.000s` -> `11.286s` (**+2.60%**, slower)

Compared with changed (`20260506-100845`):

- Median: `11s` -> `11s` (**no change**)
- Average: `11.143s` -> `11.286s` (**+1.28%**, slower)

### Hotspot impact check

- CPU remains dominated by syscall/output path (`runtime.cgocall`).
- Allocation representative total remains below baseline representative:
  `21.58MB` -> `17.54MB`.
- No recurrence of `attachMySQLColumnsToTables` allocation blow-up.

### Verdict for this batch

- Keeps MySQL better than baseline, but this batch is slightly slower than the
  two immediately previous changed batches (`093308`, `100845`).
- Allocation behavior stays healthy and below baseline representative levels.
