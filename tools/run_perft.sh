#!/usr/bin/env bash

ENGINE="./bin/silverfish"

tests=( # fen:depth:ans
  "startpos:4:197281"
  "fen r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1:3:97862"
  "fen 8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1:4:43238"
  "fen r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1:4:422333"
  "fen rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8:3:62379"
  "fen r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10:3:89890"
)

for test in "${tests[@]}"; do
  POSITION="${test%%:*}"
  REMAINDER="${test#*:}"
  DEPTH="${REMAINDER%%:*}"
  EXPECTED="${REMAINDER##*:}"

  OUTPUT=$(
    printf "uci\nposition %s\ngo perft depth %d\n" "$POSITION" "$DEPTH" \
    | $ENGINE 2>/dev/null \
    | awk -F'Perft result: ' '/Perft result:/ { print $2; exit }'
  )

  if [[ ${#POSITION} -gt 30 ]]; then
    DISPLAY_POS="${POSITION:0:30}..."
  else
    DISPLAY_POS="$POSITION"
  fi

  if [[ -z "$OUTPUT" ]]; then
    echo "ERROR: no perft result for [${DISPLAY_POS}] @ depth $DEPTH"
    continue
  fi

  if [[ "$OUTPUT" -eq "$EXPECTED" ]]; then
    echo "PASS: [${DISPLAY_POS}] @ depth $DEPTH → $OUTPUT"
  else
    echo "FAIL: [${DISPLAY_POS}] @ depth $DEPTH → got $OUTPUT but expected $EXPECTED"
  fi
done