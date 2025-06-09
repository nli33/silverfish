package main

import (
	"fmt"
	"silverfish/engine"
)

func toString(entry engine.MagicEntry) string {
	return fmt.Sprintf("	{Bitboard(%d), %d, %d},", entry.Mask, entry.Magic, entry.IndexBits)
}

func main() {
	fmt.Println("var RookMagics [64]MagicEntry = [64]MagicEntry{")
	for sq := engine.SquareA1; sq <= engine.SquareH8; sq++ {
		entry, _ := engine.FindMagic(engine.Rook, sq)
		fmt.Println(toString(entry))
	}
	fmt.Print("}\n\n")

	fmt.Println("var BishopMagics [64]MagicEntry = [64]MagicEntry{")
	for sq := engine.SquareA1; sq <= engine.SquareH8; sq++ {
		entry, _ := engine.FindMagic(engine.Bishop, sq)
		fmt.Println(toString(entry))
	}
	fmt.Println("}")
}
