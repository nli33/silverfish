#!/usr/bin/env bash
# Generate self-play games (silverfish vs itself) for Phase 2 NNUE
# fine-tuning. Mirrors sprt_run.sh's conventions but with no -sprt gate --
# this just produces a PGN of games for later labeling by
# training/selfplay_label.py.
#
# Usage:
#   tools/selfplay_run.sh [-r REF] [-n GAMES] [-c CONCURRENCY] [-t TC]
#                          [--host orca|local] [--label NAME] [--wait|--no-wait]
#
# Diversity comes from the opening book (order=random, 8moves_v3.pgn) --
# same mechanism as SPRT runs. No extra move-selection noise yet.

set -euo pipefail

REF=""
GAMES=2000
CONCURRENCY=16
TC="8+0.08"
HOST="orca"
LABEL=""
WAIT=1

while [[ $# -gt 0 ]]; do
  case "$1" in
    -r|--ref) REF="$2"; shift 2 ;;
    -n|--games) GAMES="$2"; shift 2 ;;
    -c|--concurrency) CONCURRENCY="$2"; shift 2 ;;
    -t|--tc) TC="$2"; shift 2 ;;
    --host) HOST="$2"; shift 2 ;;
    --label) LABEL="$2"; shift 2 ;;
    --wait) WAIT=1; shift ;;
    --no-wait) WAIT=0; shift ;;
    -h|--help) sed -n '2,15p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "unknown option: $1" >&2; exit 1 ;;
  esac
done

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

if [[ -z "$REF" ]]; then
  REF="$(git branch --show-current)"
fi
if [[ -z "$LABEL" ]]; then
  LABEL="selfplay_$(echo "$REF" | tr '/' '_')"
fi

echo "== selfplay_run: ref=$REF games=$GAMES concurrency=$CONCURRENCY tc=$TC host=$HOST label=$LABEL"

if [[ "$HOST" != "orca" ]]; then
  echo "error: only --host orca is implemented (self-play is a heavy job)" >&2
  exit 1
fi

BIN="$(mktemp -d)/silverfish_${LABEL}"
CURRENT_REF="$(git branch --show-current)"
if [[ "$REF" != "$CURRENT_REF" ]]; then
  WORKTREE="$(mktemp -d)/wt"
  git worktree add "$WORKTREE" "$REF" -f
  (cd "$WORKTREE" && GOOS=linux GOARCH=amd64 go build -o "$BIN" ./cmd/silverfish)
  git worktree remove "$WORKTREE" --force
else
  GOOS=linux GOARCH=amd64 go build -o "$BIN" ./cmd/silverfish
fi

echo "-- uploading to orca:~/sprt/engines/"
ssh orca "pgrep -fa 'fastchess|silverfish_' || true" | awk '{print $1}' | xargs -r ssh orca kill -9 2>/dev/null || true
scp "$BIN" "orca:~/sprt/engines/silverfish_${LABEL}"
ssh orca "chmod +x ~/sprt/engines/silverfish_${LABEL}"

LOGFILE="selfplay_out_${LABEL}.log"
echo "-- launching self-play fastchess run on orca (log: ~/sprt/$LOGFILE)"
ssh orca "cd ~/sprt && nohup ./fastchess/fastchess \
  -engine cmd=engines/silverfish_${LABEL} name=a \
  -engine cmd=engines/silverfish_${LABEL} name=b \
  -each tc=$TC \
  -openings file=books/8moves_v3.pgn format=pgn order=random \
  -rounds $((GAMES / 2)) -games 2 -repeat \
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
echo "-- running. PGN will land at orca:~/sprt/pgn/${LABEL}.pgn"
echo "-- tail with: ssh orca 'tail -f ~/sprt/$LOGFILE'"

if [[ "$WAIT" == "1" ]]; then
  echo "-- waiting for self-play run to finish..."
  while ssh orca "pgrep -fa fastchess" > /dev/null 2>&1; do
    sleep 30
  done
  echo "== done. PGN at orca:~/sprt/pgn/${LABEL}.pgn =="
fi
