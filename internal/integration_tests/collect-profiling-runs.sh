#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: collect-profiling-runs.sh [options]

Options:
  --db <all|mysql|postgres|sqlite3>     Database scenario(s) to run (default: all)
  --phase <baseline|changed>            Output phase folder (default: baseline)
  --runs <n>                            Number of runs per selected DB (default: 7)
  --batch <name>                        Batch name (default: current timestamp)
  -h, --help                            Show this help

Examples:
  ./internal/integration_tests/collect-profiling-runs.sh \
    --db all --phase baseline --runs 7 --batch exp-001

  ./internal/integration_tests/collect-profiling-runs.sh \
    --db postgres --phase changed --runs 10 --batch exp-002
EOF
}

db="all"
phase="baseline"
runs=7
batch=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --db)
      db="${2:-}"
      shift 2
      ;;
    --phase)
      phase="${2:-}"
      shift 2
      ;;
    --runs)
      runs="${2:-}"
      shift 2
      ;;
    --batch)
      batch="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

case "$db" in
  all|mysql|postgres|sqlite3) ;;
  *)
    echo "Invalid --db value: $db" >&2
    exit 1
    ;;
esac

case "$phase" in
  baseline|changed) ;;
  *)
    echo "Invalid --phase value: $phase" >&2
    exit 1
    ;;
esac

if ! [[ "$runs" =~ ^[1-9][0-9]*$ ]]; then
  echo "Invalid --runs value: $runs (must be positive integer)" >&2
  exit 1
fi

if [[ -z "$batch" ]]; then
  batch="$(date +%Y%m%d-%H%M%S)"
fi

if ! command -v go >/dev/null 2>&1; then
  echo "Could not find 'go' command in PATH" >&2
  exit 1
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/../.." && pwd)"
integration_tests_path="$repo_root/internal/integration_tests"

if [[ ! -f "$integration_tests_path/integration_profiling_test.go" ]]; then
  echo "Could not find integration_profiling_test.go under $integration_tests_path" >&2
  exit 1
fi

scenario_names=(mysql postgres sqlite3)

run_pattern_for() {
  local name="$1"
  case "$name" in
    mysql) printf '^TestIntegrationProfiling$/mysql 8$' ;;
    postgres) printf '^TestIntegrationProfiling$/postgres 18$' ;;
    sqlite3) printf '^TestIntegrationProfiling$/sqlite 3$' ;;
    *)
      return 1
      ;;
  esac
}

profiles_dir_for() {
  local name="$1"
  printf '%s' "$integration_tests_path/$name/profiling/profiles"
}

selected_scenarios=()
if [[ "$db" == "all" ]]; then
  selected_scenarios=("${scenario_names[@]}")
else
  selected_scenarios=("$db")
fi

run_once() {
  local scenario="$1"
  local pattern
  pattern="$(run_pattern_for "$scenario")"

  go test -mod=vendor -tags=integration,profiling ./internal/integration_tests -run "$pattern" -count=1
}

run_seconds() {
  local start end
  start="$(date +%s)"
  run_once "$1" 1>&2
  end="$(date +%s)"
  echo $((end - start))
}

compute_stats() {
  local summary_path="$1"
  awk -F'\t' '
    BEGIN { n=0; min=-1; max=-1; sum=0 }
    NR > 1 && $1 != "average" && $1 != "minimum" && $1 != "maximum" {
      n++
      v=$5+0
      sum+=v
      if (min < 0 || v < min) min=v
      if (max < 0 || v > max) max=v
    }
    END {
      if (n > 0) {
        avg=sum/n
        printf "average\t%.3f\n", avg
        printf "minimum\t%.3f\n", min
        printf "maximum\t%.3f\n", max
      }
    }
  ' "$summary_path" >>"$summary_path"
}

cd "$repo_root"

for scenario in "${selected_scenarios[@]}"; do
  profiles_dir="$(profiles_dir_for "$scenario")"
  target_root="$profiles_dir/runs/$phase/$batch"

  if [[ -e "$target_root" ]]; then
    echo "Target directory already exists: $target_root" >&2
    exit 1
  fi

  mkdir -p "$target_root"

  summary_path="$target_root/summary.tsv"
  printf 'db\tphase\tbatch\trun\tseconds\tpath\n' >"$summary_path"

  i=1
  while [[ "$i" -le "$runs" ]]; do
    run_name="run-$(printf '%02d' "$i")"
    run_path="$target_root/$run_name"
    mkdir -p "$run_path"

    seconds="$(run_seconds "$scenario")"

    for profile_name in cpu.pprof heap.pprof allocs.pprof; do
      source_profile="$profiles_dir/$profile_name"
      if [[ ! -f "$source_profile" ]]; then
        echo "Missing profile file: $source_profile" >&2
        exit 1
      fi
      cp "$source_profile" "$run_path/$profile_name"
    done

    printf '%s\t%s\t%s\t%s\t%.3f\t%s\n' \
      "$scenario" "$phase" "$batch" "$run_name" "$seconds" "$run_path" >>"$summary_path"

    i=$((i + 1))
  done

  compute_stats "$summary_path"
done

echo "Done. Stored runs under profiles/runs/$phase/$batch for selected DBs."
