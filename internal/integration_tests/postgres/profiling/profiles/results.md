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
4. Reduce DB query/scan-path overhead in `GetColumnsOfTables` (`database/sql` argument conversion and scan work).
   - Impact: low/medium; Risk: low.
5. Re-profile after output-path changes before any deeper DB refactor.
   - Impact: medium (decision quality); Risk: low.

## Success metrics for follow-up optimizations

- End-to-end wall time: additional >= 10% reduction from changed baseline.
- `FileWriter.Write` cumulative CPU: >= 20% reduction.
- `alloc_space` in `pkg/output` + `internal/cli`: >= 15% reduction.
- `alloc_space` in `attachPostgresqlColumnsToTables`: >= 20% reduction.
- Output correctness: no diff against expected generated files.

## Update: Changed batch `20260505-190911` (stdlib `rows.Scan` streaming)

This batch reflects the follow-up change that removed `StructScan` and switched
to stdlib row scanning while streaming rows directly into `Table.Columns`.

### Run-time comparison (latest changed)

- Latest changed (`20260505-190911`) median: `4s`
- Latest changed (`20260505-190911`) average: `4.429s`

Compared with baseline (`20260505-173911`):

- Median: `6s` -> `4s` (**-33.33%**)
- Average: `5.571s` -> `4.429s` (**-20.50%**)

Compared with previous changed (`20260505-174330`):

- Median: `5s` -> `4s` (**improved**)
- Average: `4.429s` -> `4.429s` (**no net change**)

### Hotspot impact check

- CPU remains dominated by output write path; per-table singular DB hotspot stays
  absent.
- Allocation diff against previous changed batch is small/noisy, but previous
  attach helper hotspot is no longer expected after direct streaming attach.

### Verdict for this change

- **No strong additional gain** beyond the earlier bulk-query improvement, but
  no clear regression either.
- Primary bottleneck remains output formatting and file write path.

## Update: Changed batch `20260506-093308`

This batch reflects the current direct `rows.Scan` path after dropping the
over-allocation prealloc experiment.

### Run-time comparison (latest changed)

- Latest changed (`20260506-093308`) median: `4s`
- Latest changed (`20260506-093308`) average: `4.429s`

Compared with baseline (`20260505-173911`):

- Median: `6s` -> `4s` (**-33.33%**)
- Average: `5.571s` -> `4.429s` (**-20.50%**)

Compared with previous changed (`20260505-174330`):

- Median: `4s` -> `4s` (**no change**)
- Average: `4.429s` -> `4.429s` (**no change**)

Compared with stashed changed (`20260505-190911`):

- Median: `4s` -> `4s` (**no change**)
- Average: `4.429s` -> `4.429s` (**no change**)

Compared with changed (`20260505-193236`):

- Median: `4s` -> `4s` (**no change**)
- Average: `4.286s` -> `4.429s` (**+3.34%**, slower)

### Hotspot impact check

- CPU remains dominated by syscall/output path (`runtime.cgocall`).
- Allocation representative total is effectively baseline-level:
  `28.69MB` -> `28.38MB`.
- The prior `attachPostgresqlColumnsToTables` alloc blow-up from
  `20260505-193236` is gone; DB alloc now appears in normal query/scan paths.

### Verdict for this batch

- Keeps the strong Postgres win over baseline.
- Roughly ties earlier good changed batches (`174330` and `190911`).
- Slightly worse average than `193236`, but without the severe alloc regression
  seen in that batch.

## Update: Changed batch `20260506-100845`

### Run-time comparison (latest changed)

- Latest changed (`20260506-100845`) median: `4s`
- Latest changed (`20260506-100845`) average: `4.286s`

Compared with baseline (`20260505-173911`):

- Median: `6s` -> `4s` (**-33.33%**)
- Average: `5.571s` -> `4.286s` (**-23.07%**)

Compared with previous changed (`20260505-174330`):

- Median: `4s` -> `4s` (**no change**)
- Average: `4.429s` -> `4.286s` (**-3.23%**)

Compared with stashed changed (`20260505-190911`):

- Median: `4s` -> `4s` (**no change**)
- Average: `4.429s` -> `4.286s` (**-3.23%**)

Compared with changed (`20260505-193236`):

- Median: `4s` -> `4s` (**no change**)
- Average: `4.286s` -> `4.286s` (**no change**)

Compared with changed (`20260506-093308`):

- Median: `4s` -> `4s` (**no change**)
- Average: `4.429s` -> `4.286s` (**-3.23%**)

### Hotspot impact check

- CPU remains dominated by syscall/output path (`runtime.cgocall`).
- Allocation representative total stays below baseline representative:
  `28.69MB` -> `22.22MB`.
- No `attachPostgresqlColumnsToTables` allocation blow-up observed.

### Verdict for this batch

- Matches best observed Postgres average (`20260505-193236`) while retaining a
  healthy allocation profile.
- Remains clearly better than baseline and modestly better than `174330`,
  `190911`, and `093308`.

## Update: Changed batch `20260506-104110`

### Run-time comparison (latest changed)

- Latest changed (`20260506-104110`) median: `4s`
- Latest changed (`20260506-104110`) average: `4.286s`

Compared with baseline (`20260505-173911`):

- Median: `6s` -> `4s` (**-33.33%**)
- Average: `5.571s` -> `4.286s` (**-23.07%**)

Compared with previous changed (`20260505-174330`):

- Median: `4s` -> `4s` (**no change**)
- Average: `4.429s` -> `4.286s` (**-3.23%**)

Compared with stashed changed (`20260505-190911`):

- Median: `4s` -> `4s` (**no change**)
- Average: `4.429s` -> `4.286s` (**-3.23%**)

Compared with changed (`20260505-193236`):

- Median: `4s` -> `4s` (**no change**)
- Average: `4.286s` -> `4.286s` (**no change**)

Compared with changed (`20260506-093308`):

- Median: `4s` -> `4s` (**no change**)
- Average: `4.429s` -> `4.286s` (**-3.23%**)

Compared with changed (`20260506-100845`):

- Median: `4s` -> `4s` (**no change**)
- Average: `4.286s` -> `4.286s` (**no change**)

### Hotspot impact check

- CPU remains dominated by syscall/output path (`runtime.cgocall`).
- Allocation representative total stays below baseline representative:
  `28.69MB` -> `21.71MB`.
- No `attachPostgresqlColumnsToTables` allocation blow-up observed.

### Verdict for this batch

- Repeats the same best-observed Postgres runtime band as `193236` and `100845`.
- Continues to combine strong runtime with healthy allocation behavior.
