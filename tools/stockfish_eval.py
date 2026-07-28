#!/usr/bin/env python3
"""Query the system Stockfish install as an independent strong-engine oracle.

Useful for sanity-checking a Silverfish move/score you suspect is wrong, or
for finding genuinely forced tactics (mate-in-N, hanging pieces) when
hand-constructing test positions -- rather than trusting hand analysis.

Usage:
  tools/stockfish_eval.py "<fen>" [depth]
"""
import subprocess
import sys

STOCKFISH = "stockfish"


def query(fen: str, depth: int) -> None:
    proc = subprocess.Popen(
        [STOCKFISH],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        text=True,
        bufsize=1,
    )

    def send(cmd: str) -> None:
        proc.stdin.write(cmd + "\n")
        proc.stdin.flush()

    send("uci")
    while True:
        line = proc.stdout.readline()
        if not line or line.startswith("uciok"):
            break

    send(f"position fen {fen}")
    send(f"go depth {depth}")

    last_info = ""
    try:
        while True:
            line = proc.stdout.readline()
            if not line:
                break
            line = line.strip()
            if line.startswith("info") and "score" in line:
                last_info = line
            if line.startswith("bestmove"):
                print(last_info)
                print(line)
                break
    finally:
        send("quit")
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()


def main() -> None:
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)
    fen = sys.argv[1]
    depth = int(sys.argv[2]) if len(sys.argv) > 2 else 18
    query(fen, depth)


if __name__ == "__main__":
    main()
