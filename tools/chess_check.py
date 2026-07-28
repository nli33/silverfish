#!/usr/bin/env python3
"""Chess position oracle for verifying hand-written test FENs/tactics.

Board-geometry mistakes (illegal positions, moves that look right but
aren't, "mate" positions that aren't actually mate) are an easy way to end
up with a wrong test case. This checks a position/move sequence against
python-chess instead of trusting hand visualization.

Usage:
  tools/chess_check.py "<fen>"                  # show status of a position
  tools/chess_check.py "<fen>" <uci_move>...    # apply moves, show resulting status
"""
import sys

import chess


def describe(board: chess.Board) -> str:
    lines = [
        str(board),
        f"fen: {board.fen()}",
        f"turn: {'white' if board.turn else 'black'}",
        f"valid: {board.is_valid()}",
    ]
    if not board.is_valid():
        lines.append(f"validity issue: {board.status()!r}")
    lines.append(f"in_check: {board.is_check()}")
    lines.append(f"checkmate: {board.is_checkmate()}")
    lines.append(f"stalemate: {board.is_stalemate()}")
    legal = sorted(m.uci() for m in board.legal_moves)
    lines.append(f"legal_moves ({len(legal)}): {' '.join(legal)}")
    return "\n".join(lines)


def main() -> None:
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)

    try:
        board = chess.Board(sys.argv[1])
    except ValueError as e:
        print(f"INVALID FEN: {e}")
        sys.exit(1)

    for move_str in sys.argv[2:]:
        try:
            move = chess.Move.from_uci(move_str)
        except ValueError:
            print(f"cannot parse {move_str!r} as a UCI move")
            sys.exit(1)
        if move not in board.legal_moves:
            print(f"ILLEGAL MOVE: {move_str} is not legal here")
            print(describe(board))
            sys.exit(1)
        board.push(move)

    print(describe(board))


if __name__ == "__main__":
    main()
