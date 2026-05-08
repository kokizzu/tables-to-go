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

1. `runtime.cgocall` / syscall boundary
2. `pkg/output.FileWriter.Write` cumulative write path (`os.WriteFile`/open)
3. output formatting path (`pkg/output.FormatDecorator.Decorate`, printer work)
4. syscall/string encoding helpers (`syscall.encodeWTF16` and related)
5. top-level generation pipeline (`internal/cli.(*App).Run`, cumulative)

## Top 5 allocation hotspots (changed)

1. `golang.org/x/text/transform.String`
2. growth/concat in generation and formatting (`strings.(*Builder).WriteString`, `bytes.growSlice`)
3. Postgres decode/driver path (`github.com/lib/pq.textDecode`)
4. allocator/profiling support (`runtime.mallocgc`, `runtime/pprof.StartCPUProfile`)
5. output/format support (`text/tabwriter.NewWriter`, `compress/flate.NewWriter`)

## Priority shift (baseline -> changed)

- CPU: remains syscall-dominant, with output write/format still the main actionable path.
- Alloc: latest changed sample is more formatting/driver-decode heavy; DB attach-specific hotspots are no longer leading.
- Overall: runtime remains clearly better than baseline, but latest batch sits slightly above the best observed changed averages.

## Prioritized optimization candidates (1-5)

1. Reduce output formatting overhead in `pkg/output` (`go/format`/printer-heavy path).
   - Impact: high; Risk: medium.
2. Reduce write-path syscall churn (`os.WriteFile`/open/close behavior).
   - Impact: high; Risk: medium.
3. Reduce string/decode allocation pressure in Postgres driver and generation path.
   - Impact: medium; Risk: medium.
4. Limit profiling/output helper overhead that appears in alloc top set during runs.
   - Impact: low/medium; Risk: low.
5. Re-check DB-side query/scan optimizations only after output-path reductions, since current DB mapping is not the dominant bottleneck.
   - Impact: medium (decision quality); Risk: low.

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

## Update: Changed batch `20260508-095608`

### Run-time comparison (latest changed)

- Latest changed (`20260508-095608`) median: `4s`
- Latest changed (`20260508-095608`) average: `4.429s`

Compared with baseline (`20260505-173911`):

- Median: `6s` -> `4s` (**-33.33%**)
- Average: `5.571s` -> `4.429s` (**-20.50%**)

Compared with previous documented changed (`20260506-104110`):

- Median: `4s` -> `4s` (**no change**)
- Average: `4.286s` -> `4.429s` (**+3.34%**, slower)

Compared with latest measured changed (`20260507-123350`):

- Median: `4s` -> `4s` (**no change**)
- Average: `4.429s` -> `4.429s` (**no change**)

Compared with best changed (`20260505-193236` / `20260506-100845` / `20260506-104110`):

- Median: `4s` -> `4s` (**no change**)
- Average: `4.286s` -> `4.429s` (**+3.34%**, slower)

### Hotspot impact check

- CPU remains strongly syscall-bound (`runtime.cgocall` dominates).
- Allocation profile remains led by `transform.String`, allocator/runtime support, and output/format support nodes; DB mapping allocs are present but secondary.
- No new high-confidence hotspot shift versus recent Postgres changed batches.

### Verdict for this batch

- Keeps the strong baseline improvement intact with stable median performance.
- Average sits in the slower side of the current changed band (same as `174330`, `093308`, `123350`).
- No new regression signature is visible beyond normal run-to-run variance.
