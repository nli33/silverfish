#!/usr/bin/env bash

ENGINE="${1:-./bin/silverfish}"
if [[ -z "$1" ]]; then
  make build
fi

now() { python3 -c 'import time; print(time.time())'; }

tests=( # fen:depth:ans
  "startpos:5:4865609"
  "fen r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1:4:4085603"
  "fen 8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1:6:11030083"
  "fen r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1:5:15833292"
  "fen rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8:5:89941194"
  "fen r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10:4:3894594"
)

for test in "${tests[@]}"; do
  POSITION="${test%%:*}"
  REMAINDER="${test#*:}"
  DEPTH="${REMAINDER%%:*}"
  EXPECTED="${REMAINDER##*:}"

  # The engine has no "exit on EOF" path (a separate, pre-existing UCI bug),
  # so the pipe below never closes on its own -- feed it, poll the output
  # file for the result line (bounded, so a hung/crashed engine can't wedge
  # this script forever), then kill the engine PID explicitly instead of
  # waiting on it to exit by itself.
  OUTFILE=$(mktemp)
  START=$(now)
  printf "uci\nposition %s\ngo perft depth %d\n" "$POSITION" "$DEPTH" | $ENGINE >"$OUTFILE" 2>/dev/null &
  ENGINE_PID=$!

  POLL_TIMEOUT_S=600
  OUTPUT=""
  for ((i = 0; i < POLL_TIMEOUT_S * 10; i++)); do
    LINE=$(awk -F'Perft result: ' '/Perft result:/ { print $2; exit }' "$OUTFILE" 2>/dev/null)
    if [[ -n "$LINE" ]]; then
      OUTPUT="$LINE"
      break
    fi
    sleep 0.1
  done
  END=$(now)

  kill "$ENGINE_PID" 2>/dev/null
  wait "$ENGINE_PID" 2>/dev/null
  rm -f "$OUTFILE"
  ELAPSED=$(python3 -c "print(f'{$END - $START:.3f}')")

  if [[ ${#POSITION} -gt 30 ]]; then
    DISPLAY_POS="${POSITION:0:30}..."
  else
    DISPLAY_POS="$POSITION"
  fi

  if [[ -z "$OUTPUT" ]]; then
    echo "ERROR: no perft result for [${DISPLAY_POS}] @ depth $DEPTH"
    continue
  fi

  NPS=$(python3 -c "print(int($OUTPUT / $ELAPSED)) if $ELAPSED > 0 else print('n/a')")

  if [[ "$OUTPUT" -eq "$EXPECTED" ]]; then
    echo "PASS: [${DISPLAY_POS}] @ depth $DEPTH → $OUTPUT (${ELAPSED}s, ${NPS} nps)"
  else
    echo "FAIL: [${DISPLAY_POS}] @ depth $DEPTH → got $OUTPUT but expected $EXPECTED (${ELAPSED}s)"
  fi
done
