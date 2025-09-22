#!/usr/bin/env bash
set -euo pipefail

# Generate curated report samples into docs/report-samples/
# This script runs selected fixtures with 3pio and copies key artifacts.

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BUILD_BIN="$ROOT_DIR/build/3pio"
SAMPLES_DIR="$ROOT_DIR/docs/report-samples"

require_bin() {
  if [[ ! -x "$BUILD_BIN" ]]; then
    echo "Building 3pio…" >&2
    (cd "$ROOT_DIR" && make build)
  fi
}

# Run a fixture and copy artifacts
# Args: <scenario-name> <fixture-dir> <command>
run_scenario() {
  local scenario="$1"; shift
  local fixture_rel="$1"; shift
  local cmd="$*"

  local fixture_dir="$ROOT_DIR/tests/fixtures/$fixture_rel"
  local out_dir="$SAMPLES_DIR/$scenario"

  echo "==> Scenario: $scenario"
  echo "    Fixture:  $fixture_rel"
  echo "    Command:  $cmd"

  mkdir -p "$out_dir"

  # Run tests via 3pio
  (cd "$fixture_dir" && "$BUILD_BIN" $cmd || true)

  # Find latest run directory
  local runs_dir="$fixture_dir/.3pio/runs"
  if [[ ! -d "$runs_dir" ]]; then
    echo "No runs directory found: $runs_dir" >&2
    return 1
  fi

  local run_id
  run_id=$(ls -1t "$runs_dir" | head -n1)
  local run_dir="$runs_dir/$run_id"

  echo "    Run ID:   $run_id"

  # Copy root test-run.md
  cp "$run_dir/test-run.md" "$out_dir/test-run.md"

  # Heuristic: copy first few group index.md files to illustrate state
  # You can tailor this by editing the glob or adding explicit copies per scenario.
  local count=0
  while IFS= read -r -d '' f; do
    local rel
    rel=$(python3 - <<PY
import os, sys
print(os.path.relpath(sys.argv[1], sys.argv[2]))
PY
"$f" "$run_dir")
    local dest_dir="$out_dir/groups/$(dirname "$rel")"
    mkdir -p "$dest_dir"
    cp "$f" "$dest_dir/"
    count=$((count+1))
    [[ $count -ge 5 ]] && break
  done < <(find "$run_dir" -type f -name index.md -print0 | sort -z)

  # Write/update manifest.yaml
  cat > "$out_dir/manifest.yaml" <<YAML
scenario: $scenario
fixture: $fixture_rel
command: "$cmd"
collected_at: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
run_id: "$run_id"
runner_hint: "(auto-detected in frontmatter)"
notes: |
  Curated subset of group reports (up to 5 index.md files) plus test-run.md.
  Adjust selection per scenario if you need specific states.
YAML
}

main() {
  require_bin
  mkdir -p "$SAMPLES_DIR"

  # Examples (uncomment what you need or add your own):
  # run_scenario pass-jest basic-jest "npm test --silent"
  # run_scenario empty-jest empty-jest "npm test --silent"
  # run_scenario config-error-jest jest-config-error "npm test --silent"
  # run_scenario pass-vitest basic-vitest "npx vitest run"
  # run_scenario empty-vitest empty-vitest "npx vitest run"
  # run_scenario pytest-xfail pytest-xfail "pytest -q"
  # run_scenario empty-pytest empty-pytest "pytest -q"
  # run_scenario fail-mocha failing-mocha "npx mocha"
  # run_scenario pass-go basic-go "go test ./..."

  echo "Edit scripts/generate-report-samples.sh to enable scenarios, then rerun." >&2
}

main "$@"

