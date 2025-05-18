package engine

type Move struct {
	FromSquare     Square
	ToSquare       Square
	Piece          uint8
	CapturedPiece  uint8
	PromotionPiece uint8
	IsCastle       bool
	IsEnPassant    bool
}
