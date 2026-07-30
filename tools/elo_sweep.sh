#!/usr/bin/env bash
# Gauntlet silverfish against Stockfish at a range of UCI_Elo levels, to get
# a rough absolute-strength estimate (SPRT only gives strength *relative* to
# another silverfish build). Wraps the fastchess CLI-flag invocation that
# actually works -- fastchess's -config JSON schema does NOT accept
# per-engine "options" as {"name":..,"value":..} objects (it errors with
# "type must be array, but is object"); CLI flags (-engine ... option.X=Y)
# are the reliable way to set UCI_LimitStrength/UCI_Elo per engine.
#
# Usage:
#   tools/elo_sweep.sh [options]
#
# Options:
#   -r, --ref REF          Git ref/branch to build silverfish from (default: current branch)
#   -l, --levels CSV        Comma-separated Stockfish UCI_Elo levels (default: 1600,1800,2000,2200)
#   -n, --rounds N          Rounds per opponent, 2 games/round with colors swapped (default: 50)
#   -c, --concurrency N     fastchess -concurrency (default: 8)
#   -t, --tc TC             Time control, fastchess format (default: 10+0.1)
#       --host HOST         "orca" (default, runs remotely via ssh) or "local" (runs on this machine)
#       --label NAME        Label for log/pgn filenames (default: elo_sweep_<ref>)
#       --wait / --no-wait  Block and poll until the match finishes, printing the final
#                            per-pairing results (default: --wait)
#
# Remote (--host orca) layout assumptions, matching prior manual SPRT runs:
#   ~/sprt/engines/stockfish18   -- Stockfish 18 binary already present
#   ~/sprt/books/8moves_v3.pgn   -- opening book already present
#   ~/sprt/fastchess/fastchess   -- fastchess binary already present
# silverfish is cross-compiled locally (GOOS=linux GOARCH=amd64) and scp'd
# to ~/sprt/engines/silverfish_<label> before launch.
#
# Local (--host local) mode assumes `fastchess` and a `stockfish` binary are
# on PATH, and a book at tools/books/8moves_v3.pgn (override via
# ELO_SWEEP_BOOK/ELO_SWEEP_STOCKFISH env vars if that doesn't match).

set -euo pipefail

REF=""
LEVELS="1600,1800,2000,2200"
ROUNDS=50
CONCURRENCY=8
TC="10+0.1"
HOST="orca"
LABEL=""
WAIT=1

while [[ $# -gt 0 ]]; do
  case "$1" in
    -r|--ref) REF="$2"; shift 2 ;;
    -l|--levels) LEVELS="$2"; shift 2 ;;
    -n|--rounds) ROUNDS="$2"; shift 2 ;;
    -c|--concurrency) CONCURRENCY="$2"; shift 2 ;;
    -t|--tc) TC="$2"; shift 2 ;;
    --host) HOST="$2"; shift 2 ;;
    --label) LABEL="$2"; shift 2 ;;
    --wait) WAIT=1; shift ;;
    --no-wait) WAIT=0; shift ;;
    -h|--help) sed -n '2,33p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "unknown option: $1" >&2; exit 1 ;;
  esac
done

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

if [[ -z "$REF" ]]; then
  REF="$(git branch --show-current)"
  if [[ -z "$REF" ]]; then
    echo "error: detached HEAD and no --ref given" >&2
    exit 1
  fi
fi

if [[ -z "$LABEL" ]]; then
  LABEL="elo_sweep_$(echo "$REF" | tr '/' '_')"
fi

IFS=',' read -r -a LEVEL_ARR <<< "$LEVELS"

echo "== elo_sweep: ref=$REF levels=$LEVELS rounds=$ROUNDS concurrency=$CONCURRENCY tc=$TC host=$HOST label=$LABEL"

build_binary() {
  local out="$1" goos="$2" goarch="$3"
  local build_dir="$REPO_ROOT"
  local worktree=""
  local current_ref
  current_ref="$(git branch --show-current)"

  if [[ "$REF" != "$current_ref" ]]; then
    worktree="$(mktemp -d)/wt"
    git worktree add "$worktree" "$REF" -f >&2
    build_dir="$worktree"
  fi

  (cd "$build_dir" && GOOS="$goos" GOARCH="$goarch" go build -o "$out" ./cmd/silverfish)

  if [[ -n "$worktree" ]]; then
    git worktree remove "$worktree" --force >&2
  fi
}

if [[ "$HOST" == "orca" ]]; then
  BIN="$(mktemp -d)/silverfish_$LABEL"
  echo "-- cross-compiling silverfish ($REF) for linux/amd64"
  build_binary "$BIN" linux amd64

  echo "-- uploading to orca:~/sprt/engines/"
  # Leftover fastchess/engine processes from a prior run don't always exit
  # cleanly and can hold the destination binary open, failing the scp --
  # clear them first.
  ssh orca "pgrep -fa 'fastchess|silverfish_' || true" | awk '{print $1}' | xargs -r ssh orca kill -9 2>/dev/null || true
  scp "$BIN" "orca:~/sprt/engines/silverfish_$LABEL"
  ssh orca "chmod +x ~/sprt/engines/silverfish_$LABEL"

  ENGINE_ARGS=(-engine cmd="engines/silverfish_$LABEL" name=silverfish)
  for elo in "${LEVEL_ARR[@]}"; do
    ENGINE_ARGS+=(-engine cmd=engines/stockfish18 name="sf_$elo" option.UCI_LimitStrength=true option.UCI_Elo="$elo")
  done

  LOGFILE="sprt_out_${LABEL}.log"
  echo "-- launching fastchess on orca (log: ~/sprt/$LOGFILE)"
  # nohup+disown over ssh reliably hangs the invoking shell/tool for ~2min
  # even though the remote process detaches fine -- backgrounding locally
  # and giving it a moment is the workaround.
  ssh orca "cd ~/sprt && nohup ./fastchess/fastchess \
    ${ENGINE_ARGS[*]} \
    -each tc=$TC \
    -openings file=books/8moves_v3.pgn format=pgn order=random \
    -rounds $ROUNDS -games 2 -repeat \
    -tournament gauntlet -seeds 1 \
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
    echo "-- waiting for match to finish..."
    while ssh orca "pgrep -fa fastchess" > /dev/null 2>&1; do
      sleep 30
    done
    echo "== final results =="
    ssh orca "grep -A6 '^Results of' ~/sprt/$LOGFILE" || true
  fi

else
  FASTCHESS_BIN="${ELO_SWEEP_FASTCHESS:-fastchess}"
  STOCKFISH_BIN="${ELO_SWEEP_STOCKFISH:-stockfish}"
  BOOK="${ELO_SWEEP_BOOK:-$REPO_ROOT/tools/books/8moves_v3.pgn}"

  if ! command -v "$FASTCHESS_BIN" > /dev/null; then
    echo "error: fastchess not found (set ELO_SWEEP_FASTCHESS or add to PATH)" >&2
    exit 1
  fi
  if [[ ! -f "$BOOK" ]]; then
    echo "error: opening book not found at $BOOK (set ELO_SWEEP_BOOK)" >&2
    exit 1
  fi

  BIN="$(mktemp -d)/silverfish_$LABEL"
  echo "-- building silverfish ($REF) natively"
  build_binary "$BIN" "$(go env GOOS)" "$(go env GOARCH)"

  ENGINE_ARGS=(-engine cmd="$BIN" name=silverfish)
  for elo in "${LEVEL_ARR[@]}"; do
    ENGINE_ARGS+=(-engine cmd="$STOCKFISH_BIN" name="sf_$elo" option.UCI_LimitStrength=true option.UCI_Elo="$elo")
  done

  LOGFILE="$(mktemp -d)/${LABEL}.log"
  echo "-- running fastchess locally (log: $LOGFILE)"
  "$FASTCHESS_BIN" \
    "${ENGINE_ARGS[@]}" \
    -each tc="$TC" \
    -openings file="$BOOK" format=pgn order=random \
    -rounds "$ROUNDS" -games 2 -repeat \
    -tournament gauntlet -seeds 1 \
    -concurrency "$CONCURRENCY" -recover \
    -log file="$LOGFILE" level=warn \
    | tee /dev/stderr | grep -A6 '^Results of' || true
fi
