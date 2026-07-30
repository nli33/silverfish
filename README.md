# Silverfish
[![Build and Test](https://github.com/carpetmaker3162/silverfish/actions/workflows/go.yml/badge.svg)](https://github.com/carpetmaker3162/silverfish/actions/workflows/go.yml)

![Logo](https://raw.githubusercontent.com/carpetmaker3162/silverfish/refs/heads/master/logo.svg)

UCI Chess Engine (work-in-progress)

## Features

- Hybrid bitboard & mailbox board representation
- Magic bitboard move generation
- Negamax search with alpha-beta pruning
- Iterative deepening
- Quiescence search
- Transposition table (Zobrist hashing)
- Late move reductions
- Null-move pruning
- Killer moves & history heuristic move ordering
- NNUE Evaluation, (768->256)x2->1 architecture, vertical mirroring, trained with PyTorch
    - Previously: evaluation using material counting + piece-square tables

## Quickstart

The engine itself has no external dependencies besides Go 1.22.x, so it should just work.

```bash
make run
```
