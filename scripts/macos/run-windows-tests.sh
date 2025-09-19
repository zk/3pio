#!/usr/bin/env bash
set -euo pipefail

# Start a UTM Windows VM (by bundle path or name), wait for SSH, run tests via PowerShell,
# and pull results back to .3pio/windows-run on the host.

usage() {
  cat <<'USAGE'
Usage: scripts/macos/run-windows-tests.sh \
  --host <win-host-or-ip> --user <win-user> --repo <win-path> [options]

Required:
  --host    Windows VM hostname or IP (e.g., 192.168.64.3 or winvm.local)
  --user    Windows username for SSH (ensure OpenSSH Server is enabled)
  --repo    Path to the repo inside Windows (e.g., C:\Users\you\code\3pio)

Optional:
  --port <22>              SSH port
  --vm-bundle <path.utm>   Start VM by opening this UTM bundle
  --vm-name <name>         Start VM by name (searches default UTM docs folder)
  --timeout <300>          Seconds to wait for SSH
  --results <dir>          Host results dir (default: .3pio/windows-run/<timestamp>)
  --no-start               Do not attempt to start the VM
  --insecure-ssh           Skip host key checks (convenient for local VMs)

Example:
  scripts/macos/run-windows-tests.sh \
    --host 192.168.64.3 --user edie --repo 'C:\\Users\\edie\\code\\3pio' \
    --vm-name 'Windows 11' --insecure-ssh
USAGE
}

HOST=""; USER=""; REPO_WIN=""; PORT=22; VM_BUNDLE=""; VM_NAME=""; TIMEOUT=300; RESULTS_DIR=""; NO_START=0; INSECURE_SSH=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --host) HOST="$2"; shift 2;;
    --user) USER="$2"; shift 2;;
    --repo) REPO_WIN="$2"; shift 2;;
    --port) PORT="$2"; shift 2;;
    --vm-bundle) VM_BUNDLE="$2"; shift 2;;
    --vm-name) VM_NAME="$2"; shift 2;;
    --timeout) TIMEOUT="$2"; shift 2;;
    --results) RESULTS_DIR="$2"; shift 2;;
    --no-start) NO_START=1; shift;;
    --insecure-ssh) INSECURE_SSH=1; shift;;
    -h|--help) usage; exit 0;;
    *) echo "Unknown arg: $1" >&2; usage; exit 2;;
  esac
done

if [[ -z "$HOST" || -z "$USER" || -z "$REPO_WIN" ]]; then
  echo "--host, --user, and --repo are required" >&2
  usage
  exit 2
fi

if [[ -z "$RESULTS_DIR" ]]; then
  ts=$(date +%Y-%m-%dT%H-%M-%S)
  RESULTS_DIR=".3pio/windows-run/$ts"
fi
mkdir -p "$RESULTS_DIR"

# Convert a Windows path (e.g. C:\Users\you\code\3pio) to SFTP path (/c/Users/you/code/3pio)
win_to_posix() {
  local p="$1"
  p="${p//\\/\/}" # backslashes to forward slashes
  # Drive letter C: -> /c
  if [[ "$p" =~ ^([A-Za-z]):(.*)$ ]]; then
    local drive=${BASH_REMATCH[1],,}
    local rest=${BASH_REMATCH[2]}
    echo "/$drive$rest"
  else
    echo "$p"
  fi
}

REPO_POSIX=$(win_to_posix "$REPO_WIN")
OUT_DIR_WIN="$REPO_WIN\\.3pio\\windows-run"
OUT_DIR_POSIX="$REPO_POSIX/.3pio/windows-run"

start_vm() {
  if [[ $NO_START -eq 1 ]]; then return; fi
  if [[ -n "$VM_BUNDLE" ]]; then
    echo "Starting UTM VM bundle: $VM_BUNDLE"
    open -a UTM "$VM_BUNDLE"
  elif [[ -n "$VM_NAME" ]]; then
    # Try to locate the VM by name in the default UTM documents directory
    local base="$HOME/Library/Containers/com.utmapp.UTM/Data/Documents"
    local candidate=$(find "$base" -maxdepth 1 -name "$VM_NAME*.utm" -print -quit 2>/dev/null || true)
    if [[ -n "$candidate" ]]; then
      echo "Starting UTM VM: $candidate"
      open -a UTM "$candidate"
    else
      echo "Could not find VM named '$VM_NAME' in $base; starting UTM app only. Please ensure the VM auto-starts or provide --vm-bundle." >&2
      open -a UTM || true
    fi
  else
    echo "No --vm-bundle/--vm-name provided; assuming VM is already running."
  fi
}

wait_for_ssh() {
  echo "Waiting for SSH on $HOST:$PORT (timeout ${TIMEOUT}s)..."
  local deadline=$(( $(date +%s) + TIMEOUT ))
  local ssh_opts=( -p "$PORT" -o ConnectTimeout=5 )
  if [[ $INSECURE_SSH -eq 1 ]]; then
    ssh_opts+=( -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null )
  fi
  until ssh "${ssh_opts[@]}" "$USER@$HOST" 'echo ready' >/dev/null 2>&1; do
    if (( $(date +%s) > deadline )); then
      echo "Timed out waiting for SSH on $HOST:$PORT" >&2
      exit 1
    fi
    sleep 3
  done
}

run_remote_tests() {
  echo "Running tests on Windows..."
  local ps_cmd
  ps_cmd=$(cat <<EOF
& powershell -NoProfile -ExecutionPolicy Bypass -File "$REPO_WIN\\scripts\\windows\\run-tests.ps1" -Repo "$REPO_WIN" -OutDir "$REPO_WIN\\.3pio\\windows-run"
EOF
)
  local ssh_opts=( -p "$PORT" )
  if [[ $INSECURE_SSH -eq 1 ]]; then
    ssh_opts+=( -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null )
  fi
  set +e
  ssh "${ssh_opts[@]}" "$USER@$HOST" "$ps_cmd"
  rc=$?
  set -e
  return $rc
}

pull_results() {
  echo "Copying results to $RESULTS_DIR ..."
  local scp_opts=( -P "$PORT" )
  if [[ $INSECURE_SSH -eq 1 ]]; then
    scp_opts+=( -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null )
  fi
  # Copy only new/last run files (sorted by mtime) — but simplest is to copy all
  scp -r "${scp_opts[@]}" "$USER@$HOST:$OUT_DIR_POSIX/" "$RESULTS_DIR/" || {
    echo "Warning: Could not copy results from $OUT_DIR_POSIX" >&2
  }
}

start_vm
wait_for_ssh
if run_remote_tests; then
  status=0
else
  status=$?
fi
pull_results

echo "Done. Results in: $RESULTS_DIR"
exit $status

