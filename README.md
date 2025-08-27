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
- Evaluation using material counting + piece-square tables

## Quickstart

The engine itself has no external dependencies besides Go 1.22.x, so it should just work.

```bash
go run ./cmd/silverfish
```
