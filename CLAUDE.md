# Project Instructions

## Tech Stack
- Go 1.22.2, no external dependencies (stdlib only)
- Single module `silverfish`, package `engine` holds all core logic; `cmd/silverfish` is the UCI binary entry point

## Code Style
- Standard Go formatting (gofmt); exported UCI helpers use `Uci`-prefixed names (`UciBestMove`, `UciInfo`, etc.)
- Bitboard-heavy code (`uint64` masks) alongside a mailbox board representation in `Position`
- Magic bitboards for sliding piece move generation (`engine/magics.go`, generated via `tools/generate_magics.go`)

## Testing
- Run tests: `make test` (equivalent to `go test ./engine`)
- Test files: `*_test.go` colocated with source in `engine/`
- `perft` is used as the correctness check for move generation (`engine/perft.go`): run `make perft` (or `go run tools/perft_bench.go`), which checks startpos plus several hand-picked positions (castling rights, en passant/promotion-heavy, pinned pieces, etc.) against known-correct node counts, no UCI/subprocess involved

## Build & Run
- Dev/run UCI loop: `make run` (or `go run ./cmd/silverfish`)
- Build binary: `make build` → `bin/silverfish`
- Clean: `make clean`
- `cmd/silverfish/main.go` supports `--profile` to write `cpu.prof` via pprof

## Project Structure
```
cmd/silverfish/   → main.go: UCI stdin/stdout loop, wires UciClientMessage → engine.Search
engine/           → all engine logic (single package)
  position.go     → board state, FromFEN, DoMove/UndoMove
  movegen.go      → pseudo-legal move generation
  magics.go       → magic bitboard tables for sliders
  search.go       → Search struct: negamax + alpha-beta, iterative deepening, quiescence
  evaluation.go   → classical eval (material + PST), used pre-NNUE
  nnue.go/nnue_loader.go → NNUE evaluation (768->256)x2->1, loaded from embedded/tmp file
  zobrist.go      → Zobrist hashing (for repetition/TT, in progress on this branch)
  uci.go          → UCI protocol message parsing/output
  types.go        → Search struct, move ordering (MVV-LVA), time management
  perft.go        → perft node-counting for movegen verification
models/           → .nnue weight files (1024.nnue, 256.nnue)
tools/            → Python/Go helper scripts (visualizers, perft comparison, magic generation)
bin/              → built binaries (various experiment snapshots)
```

## Dev tools for verifying chess positions/tactics

When hand-constructing FENs or tactics for tests (mate-in-N, hanging pieces,
move ordering), verify with these rather than eyeballing the board -- board
geometry mistakes are easy to make by hand and easy to catch with these:
- `python3 tools/chess_check.py "<fen>" [uci_move...]` -- python-chess position
  oracle: validity, check/checkmate/stalemate, full legal move list. Rejects
  illegal moves outright. (`pip install -r tools/requirements.txt`)
- `python3 tools/stockfish_eval.py "<fen>" [depth]` -- queries the system
  Stockfish install (must be on `PATH`) for bestmove/score, as an independent
  oracle when sanity-checking a Silverfish move/eval you suspect is wrong.

## Conventions
- Commit style: short imperative prefix sometimes used (`fix:`, `feat:`, `style:`, `test:`, `chore:`), but not strictly enforced — many commits are plain descriptions
- Work happens on feature branches (e.g. `zobrist-threefold`, `king-safety`, `nnue`, `mvv-lva`, `tt`) merged into `master` via PRs
- No CI-enforced formatting beyond `.github/workflows/go.yml` (build + test)
- Large `.pgn` files at repo root are SPRT test game logs, not source — ignore unless asked about engine testing history
