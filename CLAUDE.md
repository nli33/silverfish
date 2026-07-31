# Project Instructions

## Stack
- Go 1.22.2, stdlib only. Single package `engine` (all core logic); `cmd/silverfish` is the UCI binary entry.
- Bitboard + mailbox hybrid `Position`; magic bitboards for sliders (`engine/magics.go`, generated via `tools/generate_magics.go`).
- `Uci`-prefixed exported names for UCI helpers (`UciBestMove`, `UciInfo`, ...).

## Project structure (non-obvious parts)
- `engine/zobrist.go` — Zobrist hashing, feeds both threefold repetition and the TT.
- `engine/nnue.go`/`nnue_loader.go` — NNUE eval ((768→256)×2→1), weights embedded in the binary via `go:embed`, not loaded from disk at runtime.
- `engine/evaluation.go` — old material+PST eval, kept only as NNUE's predecessor/fallback.
- `engine/perft.go` — node-counting movegen verification, separate from the test suite.
- `bin/` — built binaries from past experiments, not source.
- Large `.pgn` files at repo root — SPRT game logs, not source; ignore unless asked about testing history.

## Commands
- `make test` (`go test ./engine`) — unit tests, colocated `*_test.go`.
- `make perft` (`go run tools/perft_bench.go`) — movegen correctness vs. known node counts (startpos + hand-picked edge-case positions). Run after any movegen/position change.
- `make run` / `make build` / `make clean` — UCI loop, binary to `bin/`, cleanup. `--profile` flag on the binary writes `cpu.prof`.

## Verifying chess positions/tactics
Hand-built FENs/tactics are easy to get wrong by eye — verify instead:
- `python3 tools/chess_check.py "<fen>" [uci_move...]` — python-chess oracle (validity, check/mate/stalemate, legal move list).
- `python3 tools/stockfish_eval.py "<fen>" [depth]` — independent eval/bestmove oracle via system Stockfish.

## Strength testing
Two distinct workflows, don't conflate them:
- **SPRT** (relative strength, build vs. build) — `tools/sprt_run.sh [-a refA] [-b refB] [--elo0 N] [--elo1 N] [-n rounds] [-c concurrency] [-t tc] [--host orca|local] [--label name] [--wait|--no-wait]`. Used to validate any change to move selection/ordering before merging. `--help` for full flags.
- **Elo sweep** (absolute strength vs. Stockfish) — `tools/elo_sweep.sh [-r ref] [-l levels] [-n rounds] [-c concurrency] [-t tc] [--host orca|local] [--label name] [--wait|--no-wait]`, gauntlets a build against Stockfish `UCI_Elo` levels. `--help` for full flags.
- Both wrap `fastchess` (remote host `orca` via `ssh orca`, `~/sprt/`, by default; `--host local` if not available) and use CLI flags (`-engine ... option.X=Y`), never a JSON `-config` file — fastchess's JSON schema rejects per-engine `options` as `{name,value}` objects. `-pgnout`/`-log` also need `key=value` form (`-pgnout file=x.pgn`, not `-pgnout x.pgn`)
- Prefer these scripts over hand-written `ssh`/`fastchess` invocations — less to get wrong.
- Stockfish's `UCI_Elo` is CCRL-calibrated, not FIDE — treat resulting Elo estimates as a same-species floor, not a human-equivalent number.

## Process
- Don't write substantial one-off code and run it ad hoc on the command line (e.g. `python3 -c "..."`) for things like training/benchmarking — not reproducible, not reviewed. Fix or extend the actual pipeline/tool script and run that instead.

## Conventions
- Feature branches → PR → merge to `master`. Commit prefixes (`fix:`, `feat:`, etc.) used loosely, not enforced.
- Any change to move selection/ordering needs an SPRT result before merging, same bar as past search features.
- **Opening a PR for a strength-affecting change implicitly signals "SPRT'd, OK to merge" — never open one before the SPRT is done.** Non-strength-affecting changes (docs, tooling, refactors, unwired-in foundational code) don't need PR ceremony at all — commit straight to `master`. Reserve PRs for changes that need review/SPRT-gating before merge.

## PR messages
- Brief summary up top; skip what's obvious from the diff. Spend words on *why* (design tradeoffs, non-obvious reasoning).
- Include full SPRT stats (Elo/LLR/games) for engine-strength changes; skip for pure refactors.
