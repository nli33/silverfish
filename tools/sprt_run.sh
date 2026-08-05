#!/usr/bin/env bash
# Run an SPRT between two silverfish builds (relative strength -- for
# absolute strength vs. Stockfish, use elo_sweep.sh instead). Wraps the
# fastchess CLI-flag invocation used for every SPRT in this repo's history
# (PRs #12/#13/#14 etc).
#
# Usage:
#   tools/sprt_run.sh [options]
#
# Options:
#   -a, --ref-a REF         Git ref/branch for engine A (default: current branch)
#   -b, --ref-b REF         Git ref/branch for engine B (default: master)
#       --opt-a "X=Y,..."   UCI options for engine A, e.g. "Threads=4"
#       --opt-b "X=Y,..."   UCI options for engine B
#       --elo0 N            SPRT lower bound, "nothing to see" hypothesis (default: 0)
#       --elo1 N            SPRT upper bound, "worth it" hypothesis (default: 5)
#   -n, --rounds N          Max rounds, 2 games/round with colors swapped (default: 800)
#   -c, --concurrency N     fastchess -concurrency (default: 8)
#   -t, --tc TC             Time control, fastchess format (default: 10+0.1)
#       --host HOST         "orca" (default, runs remotely via ssh) or "local"
#       --label NAME        Label for log/pgn filenames (default: sprt_<a>_vs_<b>)
#       --wait / --no-wait  Block and poll until SPRT concludes, printing the
#                            final result (default: --wait)
#
# Remote (--host orca) layout assumptions, matching prior manual SPRT runs:
#   ~/sprt/books/8moves_v3.pgn   -- opening book already present
#   ~/sprt/fastchess/fastchess   -- fastchess binary already present
# Both refs are cross-compiled locally (GOOS=linux GOARCH=amd64) and scp'd to
# ~/sprt/engines/silverfish_<label>_a / _b before launch.
#
# Local (--host local) mode assumes `fastchess` is on PATH.
#
# Known fastchess gotchas (already hit once, don't re-discover):
#   - No JSON -config file for per-engine UCI options -- use CLI flags
#     (-engine ... option.X=Y) instead; the JSON schema rejects
#     {"name":..,"value":..} option objects.
#   - -pgnout (like -log) takes a key=value pair -- `-pgnout file=x.pgn`, NOT
#     `-pgnout x.pgn`. The latter fails with "expects key=value pairs".

set -euo pipefail

REF_A=""
REF_B="master"
ELO0=0
ELO1=5
ROUNDS=800
CONCURRENCY=8
TC="10+0.1"
HOST="orca"
LABEL=""
WAIT=1
OPT_A=""
OPT_B=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -a|--ref-a) REF_A="$2"; shift 2 ;;
    -b|--ref-b) REF_B="$2"; shift 2 ;;
    --opt-a) OPT_A="$2"; shift 2 ;;
    --opt-b) OPT_B="$2"; shift 2 ;;
    --elo0) ELO0="$2"; shift 2 ;;
    --elo1) ELO1="$2"; shift 2 ;;
    -n|--rounds) ROUNDS="$2"; shift 2 ;;
    -c|--concurrency) CONCURRENCY="$2"; shift 2 ;;
    -t|--tc) TC="$2"; shift 2 ;;
    --host) HOST="$2"; shift 2 ;;
    --label) LABEL="$2"; shift 2 ;;
    --wait) WAIT=1; shift ;;
    --no-wait) WAIT=0; shift ;;
    -h|--help) sed -n '2,36p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "unknown option: $1" >&2; exit 1 ;;
  esac
done

# opt_flags converts "X=Y,Z=W" into " option.X=Y option.Z=W" (leading space,
# empty string for no options) for appending to a fastchess -engine line.
opt_flags() {
  local spec="$1"
  [[ -z "$spec" ]] && return 0
  local out="" pair
  IFS=',' read -ra pairs <<< "$spec"
  for pair in "${pairs[@]}"; do
    out+=" option.${pair}"
  done
  echo "$out"
}

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

if [[ -z "$REF_A" ]]; then
  REF_A="$(git branch --show-current)"
  if [[ -z "$REF_A" ]]; then
    echo "error: detached HEAD and no --ref-a given" >&2
    exit 1
  fi
fi

sanitize() { echo "$1" | tr '/' '_'; }

if [[ -z "$LABEL" ]]; then
  LABEL="sprt_$(sanitize "$REF_A")_vs_$(sanitize "$REF_B")"
fi

NAME_A="$(sanitize "$REF_A")"
NAME_B="$(sanitize "$REF_B")"

echo "== sprt_run: a=$REF_A b=$REF_B elo0=$ELO0 elo1=$ELO1 rounds=$ROUNDS concurrency=$CONCURRENCY tc=$TC host=$HOST label=$LABEL"

build_binary() {
  local out="$1" goos="$2" goarch="$3" ref="$4"
  local build_dir="$REPO_ROOT"
  local worktree=""
  local current_ref
  current_ref="$(git branch --show-current)"

  if [[ "$ref" != "$current_ref" ]]; then
    worktree="$(mktemp -d)/wt"
    git worktree add "$worktree" "$ref" -f >&2
    build_dir="$worktree"
  fi

  (cd "$build_dir" && GOOS="$goos" GOARCH="$goarch" go build -o "$out" ./cmd/silverfish)

  if [[ -n "$worktree" ]]; then
    git worktree remove "$worktree" --force >&2
  fi
}

if [[ "$HOST" == "orca" ]]; then
  BIN_A="$(mktemp -d)/silverfish_${LABEL}_a"
  BIN_B="$(mktemp -d)/silverfish_${LABEL}_b"
  echo "-- cross-compiling silverfish ($REF_A, $REF_B) for linux/amd64"
  build_binary "$BIN_A" linux amd64 "$REF_A"
  build_binary "$BIN_B" linux amd64 "$REF_B"

  SIZE_A=$(wc -c < "$BIN_A")
  SIZE_B=$(wc -c < "$BIN_B")
  if [[ "$SIZE_A" == "$SIZE_B" && "$REF_A" != "$REF_B" ]]; then
    echo "warning: both binaries are the same size ($SIZE_A bytes) -- verify they were actually built from different refs" >&2
  fi

  echo "-- uploading to orca:~/sprt/engines/"
  # Leftover fastchess/engine processes from a prior run don't always exit
  # cleanly and can hold the destination binary open, failing the scp --
  # clear them first.
  ssh orca "pgrep -fa 'fastchess|silverfish_' || true" | awk '{print $1}' | xargs -r ssh orca kill -9 2>/dev/null || true
  scp "$BIN_A" "orca:~/sprt/engines/silverfish_${LABEL}_a"
  scp "$BIN_B" "orca:~/sprt/engines/silverfish_${LABEL}_b"
  ssh orca "chmod +x ~/sprt/engines/silverfish_${LABEL}_a ~/sprt/engines/silverfish_${LABEL}_b"

  LOGFILE="sprt_out_${LABEL}.log"
  echo "-- launching fastchess on orca (log: ~/sprt/$LOGFILE)"
  # nohup+disown over ssh reliably hangs the invoking shell/tool for ~2min
  # even though the remote process detaches fine -- backgrounding locally
  # and giving it a moment is the workaround.
  ssh orca "cd ~/sprt && nohup ./fastchess/fastchess \
    -engine cmd=engines/silverfish_${LABEL}_a name=$NAME_A$(opt_flags "$OPT_A") \
    -engine cmd=engines/silverfish_${LABEL}_b name=$NAME_B$(opt_flags "$OPT_B") \
    -each tc=$TC \
    -openings file=books/8moves_v3.pgn format=pgn order=random \
    -rounds $ROUNDS -games 2 -repeat \
    -sprt elo0=$ELO0 elo1=$ELO1 alpha=0.05 beta=0.05 \
    -concurrency $CONCURRENCY -recover \
    -pgnout file=pgn/${LABEL}.pgn \
    -log file=${LABEL}.log level=warn \
    > $LOGFILE 2>&1 & disown" &
  LAUNCH_PID=$!
  sleep 5
  kill "$LAUNCH_PID" 2>/dev/null || true

  if ! ssh orca "pgrep -fa fastchess" > /dev/null; then
    echo "error: fastchess doesn't appear to be running on orca -- check ~/sprt/$LOGFILE" >&2
    ssh orca "tail -30 ~/sprt/$LOGFILE" >&2 || true
    exit 1
  fi
  echo "-- running. tail with: ssh orca 'tail -f ~/sprt/$LOGFILE'"

  if [[ "$WAIT" == "1" ]]; then
    echo "-- waiting for SPRT to conclude..."
    while ssh orca "pgrep -fa fastchess" > /dev/null 2>&1; do
      sleep 30
    done
    echo "== final result =="
    ssh orca "grep -A8 '^Results of' ~/sprt/$LOGFILE" || true
  fi

else
  FASTCHESS_BIN="${SPRT_RUN_FASTCHESS:-fastchess}"
  BOOK="${SPRT_RUN_BOOK:-$REPO_ROOT/tools/books/8moves_v3.pgn}"

  if ! command -v "$FASTCHESS_BIN" > /dev/null; then
    echo "error: fastchess not found (set SPRT_RUN_FASTCHESS or add to PATH)" >&2
    exit 1
  fi
  if [[ ! -f "$BOOK" ]]; then
    echo "error: opening book not found at $BOOK (set SPRT_RUN_BOOK)" >&2
    exit 1
  fi

  BIN_A="$(mktemp -d)/silverfish_${LABEL}_a"
  BIN_B="$(mktemp -d)/silverfish_${LABEL}_b"
  echo "-- building silverfish ($REF_A, $REF_B) natively"
  build_binary "$BIN_A" "$(go env GOOS)" "$(go env GOARCH)" "$REF_A"
  build_binary "$BIN_B" "$(go env GOOS)" "$(go env GOARCH)" "$REF_B"

  LOGFILE="$(mktemp -d)/${LABEL}.log"
  echo "-- running fastchess locally (log: $LOGFILE)"
  "$FASTCHESS_BIN" \
    -engine cmd="$BIN_A" name="$NAME_A"$(opt_flags "$OPT_A") \
    -engine cmd="$BIN_B" name="$NAME_B"$(opt_flags "$OPT_B") \
    -each tc="$TC" \
    -openings file="$BOOK" format=pgn order=random \
    -rounds "$ROUNDS" -games 2 -repeat \
    -sprt elo0="$ELO0" elo1="$ELO1" alpha=0.05 beta=0.05 \
    -concurrency "$CONCURRENCY" -recover \
    -log file="$LOGFILE" level=warn \
    | tee /dev/stderr | grep -A8 '^Results of' || true
fi
