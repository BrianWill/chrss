package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

func initMatch(m *Match) {
	// random adjective-animal
	m.Name = adjectives[rand.Intn(len(adjectives))] + "-" + animals[rand.Intn(len(animals))]
	m.LastMoveTime = time.Now().UnixNano()
	m.StartTime = m.LastMoveTime
	m.Turn = white
	m.Winner = none
	m.MaxRank = 1

	public := &m.WhitePublic
	public.Color = white
	public.Other = &m.BlackPublic
	public.King = &Piece{king, white, kingHP, kingAttack, 0, nil}
	public.Bishop = &Piece{bishop, white, bishopHP, bishopAttack, 0, nil}
	public.Knight = &Piece{knight, white, knightHP, knightAttack, 0, nil}
	public.Rook = &Piece{rook, white, rookHP, rookAttack, 0, nil}

	public = &m.BlackPublic
	public.Color = black
	public.Other = &m.WhitePublic
	public.King = &Piece{king, black, kingHP, kingAttack, 0, nil}
	public.Bishop = &Piece{bishop, black, bishopHP, bishopAttack, 0, nil}
	public.Knight = &Piece{knight, black, knightHP, knightAttack, 0, nil}
	public.Rook = &Piece{rook, black, rookHP, rookAttack, 0, nil}

	m.Log = []string{"Round 1"}

	SpawnPawns(true, &m.Board, &m.WhitePublic, &m.BlackPublic, &m.Log)
	m.UpdateStatusAndDamage()

	stock := []Card{
		Card{bishop, bishopMana},
		Card{knight, knightMana},
		Card{rook, rookMana},
	}

	m.BlackPrivate = PrivateState{SelectedCard: -1}
	// white starts ready to play king
	m.WhitePrivate = PrivateState{SelectedCard: -1}

	if m.DevMode {
		m.BlackPrivate.Cards = append(append([]Card{}, stock...), allCards...)
		m.WhitePrivate.Cards = append(append([]Card{}, stock...), allCards...)
	} else {
		m.BlackPrivate.Cards = append(append([]Card{}, stock...),
			randomCards(nCardsFirstRound, m.MaxRank)...)
		m.WhitePrivate.Cards = append(append([]Card{}, stock...),
			randomCards(nCardsFirstRound, m.MaxRank)...)
	}

	m.BlackPrivate.Other = &m.WhitePrivate
	m.WhitePrivate.Other = &m.BlackPrivate

	dimAllButFree(black, &m.Board, m.BlackPrivate.Highlights[:])
	dimAllButFree(white, &m.Board, m.WhitePrivate.Highlights[:])

	m.PlayableCards(&m.Board)

	if m.BlackAI {
		public, private := m.states(black)
		pos := kingPlacementAI(black, &m.Board)
		private.KingPos = &pos
		public.KingPlayed = true
		m.Log = append(m.Log, "black played King")
	}
	if m.WhiteAI {
		public, private := m.states(white)
		pos := kingPlacementAI(white, &m.Board)
		private.KingPos = &pos
		public.KingPlayed = true
		m.Log = append(m.Log, "white played King")
	}
}

func getPiece(p Pos, board *Board) *Piece {
	return board.Pieces[nColumns*p.Y+p.X]
}

// does not panic
func getPieceSafe(p Pos, board *Board) *Piece {
	if p.X < 0 || p.X >= nColumns || p.Y < 0 || p.Y >= nRows {
		return nil
	}
	return board.Pieces[nColumns*p.Y+p.X]
}

// returns -1 if invalid
func (p *Pos) getBoardIdx() int {
	if p.X < 0 || p.X >= nColumns {
		return -1
	}
	if p.Y < 0 || p.Y >= nRows {
		return -1
	}
	return p.X + nColumns*p.Y
}

func getBoardIdx(x int, y int) int {
	if x < 0 || x >= nColumns {
		return -1
	}
	if y < 0 || y >= nRows {
		return -1
	}
	return x + nColumns*y
}

// panics if out of bounds
func setPiece(p Pos, piece Piece, board *Board) {
	idx := nColumns*p.Y + p.X
	board.PiecesActual[idx] = piece
	board.Pieces[idx] = &board.PiecesActual[idx]
}

// panics if out of bounds
func removePieceAt(p Pos, board *Board) {
	idx := nColumns*p.Y + p.X
	board.Pieces[idx] = nil
	board.PiecesActual[idx] = Piece{}
}

func RemoveNonPawns(board *Board) {
	for i, p := range board.Pieces {
		if p.Name != pawn {
			board.PiecesActual[i] = Piece{}
			board.Pieces[i] = nil
		}
	}
}

func InflictDamage(board *Board, whitePublic *PublicState, blackPublic *PublicState) {
	for i, p := range board.Pieces {
		if p != nil {
			p.HP -= p.Damage
			p.Damage = 0
			public := whitePublic
			if p.Color == black {
				public = blackPublic
			}
			switch p.Name {
			case king:
				public.King.HP = p.HP
			case bishop:
				public.Bishop.HP = p.HP
			case knight:
				public.Knight.HP = p.HP
			case rook:
				public.Rook.HP = p.HP
			}
			if p.HP <= 0 {
				switch p.Name {
				case bishop:
					public.BishopPlayed = false
				case knight:
					public.KnightPlayed = false
				case rook:
					public.RookPlayed = false
				case pawn:
					public.NumPawns--
				}
				board.PiecesActual[i] = Piece{}
				board.Pieces[i] = nil
			}
		}
	}
}

func CalculateDamage(board *Board, squareStatuses []SquareStatus) {
	rookAttack := func(p Pos, color string, attack int, enraged bool) {
		x := p.X + 1
		y := p.Y
		for x < nColumns {
			hit := getPiece(Pos{x, y}, board)
			if hit != nil {
				if !hit.isDamageImmune() && (hit.Color != color || enraged) {
					hit.Damage += hit.armorMitigation(attack)
				}
				if !hit.isTransparent() {
					break
				}
			}
			x++
		}

		x = p.X - 1
		y = p.Y
		for x >= 0 {
			hit := getPiece(Pos{x, y}, board)
			if hit != nil {
				if !hit.isDamageImmune() && (hit.Color != color || enraged) {
					hit.Damage += hit.armorMitigation(attack)
				}
				if !hit.isTransparent() {
					break
				}
			}
			x--
		}

		x = p.X
		y = p.Y + 1
		for y < nRows {
			hit := getPiece(Pos{x, y}, board)
			if hit != nil {
				if !hit.isDamageImmune() && (hit.Color != color || enraged) {
					hit.Damage += hit.armorMitigation(attack)
				}
				if !hit.isTransparent() {
					break
				}
			}
			y++
		}

		x = p.X
		y = p.Y - 1
		for y >= 0 {
			hit := getPiece(Pos{x, y}, board)
			if hit != nil {
				if !hit.isDamageImmune() && (hit.Color != color || enraged) {
					hit.Damage += hit.armorMitigation(attack)
				}
				if !hit.isTransparent() {
					break
				}
			}
			y--
		}
	}

	bishopAttack := func(p Pos, color string, attack int, enraged bool) {
		x := p.X + 1
		y := p.Y + 1
		for x < nColumns && y < nRows {
			hit := getPiece(Pos{x, y}, board)
			if hit != nil {
				if !hit.isDamageImmune() && (hit.Color != color || enraged) {
					hit.Damage += hit.armorMitigation(attack)
				}
				if !hit.isTransparent() {
					break
				}
			}
			x++
			y++
		}

		x = p.X - 1
		y = p.Y + 1
		for x >= 0 && y < nRows {
			hit := getPiece(Pos{x, y}, board)
			if hit != nil {
				if !hit.isDamageImmune() && (hit.Color != color || enraged) {
					hit.Damage += hit.armorMitigation(attack)
				}
				if !hit.isTransparent() {
					break
				}
			}
			x--
			y++
		}

		x = p.X + 1
		y = p.Y - 1
		for x < nColumns && y >= 0 {
			hit := getPiece(Pos{x, y}, board)
			if hit != nil {
				if !hit.isDamageImmune() && (hit.Color != color || enraged) {
					hit.Damage += hit.armorMitigation(attack)
				}
				if !hit.isTransparent() {
					break
				}
			}
			x++
			y--
		}

		x = p.X - 1
		y = p.Y - 1
		for x >= 0 && y >= 0 {
			hit := getPiece(Pos{x, y}, board)
			if hit != nil {
				if !hit.isDamageImmune() && (hit.Color != color || enraged) {
					hit.Damage += hit.armorMitigation(attack)
				}
				if !hit.isTransparent() {
					break
				}
			}
			x--
			y--
		}
	}

	queenAttack := func(p Pos, color string, attack int, enraged bool) {
		rookAttack(p, color, attack, enraged)
		bishopAttack(p, color, attack, enraged)
	}

	kingAttack := func(p Pos, color string, attack int, enraged bool) {
		ps := []Pos{
			Pos{p.X + 1, p.Y + 1},
			Pos{p.X + 1, p.Y},
			Pos{p.X + 1, p.Y - 1},
			Pos{p.X, p.Y + 1},
			Pos{p.X, p.Y - 1},
			Pos{p.X - 1, p.Y + 1},
			Pos{p.X - 1, p.Y},
			Pos{p.X - 1, p.Y - 1},
		}
		for _, other := range ps {
			hit := getPieceSafe(other, board)
			if hit != nil && !hit.isDamageImmune() && (hit.Color != color || enraged) {
				hit.Damage += hit.armorMitigation(attack)
			}
		}
	}

	knightAttack := func(p Pos, color string, attack int, enraged bool) {
		ps := []Pos{
			Pos{p.X + 1, p.Y + 2},
			Pos{p.X + 1, p.Y - 2},
			Pos{p.X + 2, p.Y + 1},
			Pos{p.X + 2, p.Y - 1},
			Pos{p.X - 1, p.Y + 2},
			Pos{p.X - 1, p.Y - 2},
			Pos{p.X - 2, p.Y + 1},
			Pos{p.X - 2, p.Y - 1},
		}
		for _, other := range ps {
			hit := getPieceSafe(other, board)
			if hit != nil && !hit.isDamageImmune() && (hit.Color != color || enraged) {
				hit.Damage += hit.armorMitigation(attack)
			}
		}
	}

	jesterAttack := func(p Pos, color string, attack int, enraged bool) {
		// do nothing: jester does not attack
	}

	pawnAttack := func(p Pos, color string, attack int, enraged bool) {
		yOffset := 1
		if color == black {
			yOffset = -1
		}
		ps := []Pos{
			Pos{p.X + 1, p.Y + yOffset},
			Pos{p.X - 1, p.Y + yOffset},
		}
		for _, other := range ps {
			hit := getPieceSafe(other, board)
			if hit != nil && !hit.isDamageImmune() && (hit.Color != color || enraged) {
				hit.Damage += hit.armorMitigation(attack)
			}
		}
	}

	// reset all to 0
	for i := range board.Pieces {
		board.PiecesActual[i].Damage = 0
	}

	attackMap := map[string]func(Pos, string, int, bool){
		king:   kingAttack,
		bishop: bishopAttack,
		knight: knightAttack,
		rook:   rookAttack,
		pawn:   pawnAttack,
		queen:  queenAttack,
		jester: jesterAttack,
	}

	// visit each piece, adding the damage it inflicts on other pieces
	for i, p := range board.Pieces {
		squareStatus := squareStatuses[i]
		if squareStatus.Negative != nil && squareStatus.Negative.Distracted {
			continue
		}
		if p != nil {
			if !p.isDistracted() {
				attackMap[p.Name](
					positions[i],
					p.Color,
					p.getAmplifiedDamage(),
					p.isEnraged(),
				)
			}
		}
	}

	for _, p := range board.Pieces {
		if p != nil && p.Status != nil {
			neg := p.Status.Negative
			if neg != nil {
				if neg.Poison > 0 {
					p.Damage += neg.Poison
				}
				// vulnerability factored after poison!
				if neg.Vulnerability > 0 {
					p.Damage *= vulnerabilityFactor
				}
			}
			pos := p.Status.Positive
			if pos != nil {
			}
		}
	}
}

func SpawnSinglePawn(color string, public *PublicState, test bool, board *Board) bool {
	if public.NumPawns == maxPawns {
		return false
	}
	n := 1
	offset := 1
	if color == black {
		offset = 3
	}
	columns := freePawnColumns(color, board)
	columns = randSelect(n, columns)
	n = len(columns)
	if n < 1 {
		return false
	}
	if test {
		return true
	}
	for _, v := range columns {
		setPiece(Pos{v, rand.Intn(2) + offset}, Piece{pawn, public.Color, pawnHP, pawnAttack, 0, nil}, board)
	}
	public.NumPawns += n
	return true
}

func SpawnSinglePawnTemp(color string, public *PublicState, board *Board) bool {
	if public.NumPawns == maxPawns {
		return false
	}
	n := 1
	offset := 1
	if color == black {
		offset = 3
	}
	columns := freePawnColumns(color, board)
	columns = randSelect(n, columns)
	n = len(columns)
	if n < 1 {
		return false
	}
	for _, v := range columns {
		setPiece(Pos{v, rand.Intn(2) + offset}, Piece{pawn, public.Color, pawnHP, pawnAttack, 0, nil}, board)
	}
	return true
}

// returns indexes of the columns in which a new pawn can be placed
func freePawnColumns(color string, board *Board) []int {
	var columns []int
	front := 2
	mid := 1
	if color == black {
		front = 3
		mid = 4
	}
	for i := 0; i < nColumns; i++ {
		if getPiece(Pos{i, front}, board) == nil && getPiece(Pos{i, mid}, board) == nil {
			columns = append(columns, i)
		}
	}
	return columns
}

// spawn n random pawns in free columns
func SpawnPawns(init bool, board *Board, whitePub *PublicState, blackPub *PublicState, log *[]string) {
	public := whitePub
	for i := 0; i < 2; i++ {
		n := 1
		if init {
			n = startingPawns
		}
		if !init && public.NumPawns == 0 {
			n = 2
		} else if public.NumPawns == maxPawns {
			return // keep max pawns at 5
		}
		offset := 1
		if public.Color == black {
			offset = 3
		}
		columns := freePawnColumns(public.Color, board)
		columns = randSelect(n, columns)
		n = len(columns)
		for _, v := range columns {
			setPiece(Pos{v, rand.Intn(2) + offset}, Piece{pawn, public.Color, pawnHP, pawnAttack, 0, nil}, board)
		}
		public.NumPawns += n
		switch n {
		case 0:
			*log = append(*log, public.Color+" gained no pawns")
		case 1:
			*log = append(*log, public.Color+" gained 1 pawn")
		default:
			*log = append(*log, public.Color+" gained "+strconv.Itoa(n)+" pawns")
		}
		public = blackPub
	}
}

// returns boolean true when no free slot
func RandomFreeSquare(player string, board *Board) (Pos, bool) {
	// collect Pos of all free squares on player's side
	freeSquares := []Pos{}
	x := 0
	y := 0
	i := 0
	end := len(board.Pieces) / 2
	if player == black {
		y = nRows / 2
		i = end
		end = len(board.Pieces)
	}
	for ; i < end; i++ {
		if board.Pieces[i] == nil {
			freeSquares = append(freeSquares, Pos{x, y})
		}
		x++
		if x == nColumns {
			x = 0
			y++
		}
	}
	if len(freeSquares) == 0 {
		return Pos{}, true
	}

	// random pick from the free Pos
	return freeSquares[rand.Intn(len(freeSquares))], false
}

func (m *Match) states(color string) (*PublicState, *PrivateState) {
	if color == black {
		return &m.BlackPublic, &m.BlackPrivate
	} else {
		return &m.WhitePublic, &m.WhitePrivate
	}
}

// determine which cards are playable for each player given state of board
func (m *Match) PlayableCards(board *Board) {
	public, private := m.states(white)
	for i := 0; i < 2; i++ {
		private.PlayableCards = make([]bool, len(private.Cards))
		for j, c := range private.Cards {
			private.PlayableCards[j] = false
			switch c.Name {
			case bishop, knight, rook, queen, jester:
				if hasFreeSpace(public.Color, board) {
					private.PlayableCards[j] = true
				}
			case forceCombatCard, mirrorCard, nukeCard, vulnerabilityCard, transparencyCard, amplifyCard, enrageCard, swapFrontLinesCard:
				private.PlayableCards[j] = true
			case dispellCard:
				if len(statusEffectedPieces(none, board)) > 0 {
					private.PlayableCards[j] = true
				}
			case stunVassalCard:
				if public.Other.KnightPlayed || public.Other.RookPlayed || public.Other.BishopPlayed {
					private.PlayableCards[j] = true
				}
			case dodgeCard:
				if len(dodgeablePieces(public.Color, board)) > 0 {
					private.PlayableCards[j] = true
				}
			case castleCard:
				if public.RookPlayed || public.Other.RookPlayed {
					private.PlayableCards[j] = true
				}
			case reclaimVassalCard:
				if public.RookPlayed || public.KnightPlayed || public.BishopPlayed {
					private.PlayableCards[j] = true
				}
			case shoveCard:
				if len(shoveablePieces(board)) > 0 {
					private.PlayableCards[j] = true
				}
			case advanceCard:
				if len(advanceablePieces(board)) > 0 {
					private.PlayableCards[j] = true
				}
			case summonPawnCard:
				if public.NumPawns < maxPawns && len(freePawnColumns(public.Color, board)) > 0 {
					private.PlayableCards[j] = true
				}
			case resurrectVassalCard:
				if public.Bishop.HP <= 0 || public.Rook.HP <= 0 || public.Knight.HP <= 0 {
					private.PlayableCards[j] = true
				}
			case togglePawnCard:
				if len(toggleablePawns(board)) > 0 {
					private.PlayableCards[j] = true
				}
			case poisonCard:
				// only playable on enemy piece other than king
				if pieceCount(public.Other.Color, board) > 1 {
					private.PlayableCards[j] = true
				}
			case healCard, armorCard:
				// only playable on piece other than king
				if pieceCount(public.Color, board) > 1 {
					private.PlayableCards[j] = true
				}
			case removePawnCard:
				if public.NumPawns > 0 || public.Other.NumPawns > 0 {
					private.PlayableCards[j] = true
				}
			}
		}
		public, private = m.states(black)
	}
}

func otherColor(color string) string {
	if color == black {
		return white
	} else {
		return black
	}
}

// returns indexes of all pawns which can be toggled
func toggleablePawns(board *Board) []int {
	indexes := []int{}
	start := (nRows / 2) * nColumns // first look for black toggleable pawns
	for i := 0; i < 2; i++ {
		for j := start; j < start+nColumns; j++ {
			k := j + nColumns
			a, b := board.Pieces[j], board.Pieces[k]
			if a != nil && a.Name == pawn && b == nil {
				indexes = append(indexes, j)
			} else if b != nil && b.Name == pawn && a == nil {
				indexes = append(indexes, k)
			}
		}
		start -= 2 * nColumns // repeat for white
	}
	return indexes
}

// returns indexes of all pawns which can be toggled
func shoveablePieces(board *Board) []int {
	indexes := []int{}
	for i, p := range board.Pieces {
		if p != nil {
			if p.Color == black {
				j := i + nColumns
				if j < len(board.Pieces) && board.Pieces[j] == nil {
					indexes = append(indexes, i)
				}
			} else {
				j := i - nColumns
				if j >= 0 && board.Pieces[j] == nil {
					indexes = append(indexes, i)
				}
			}
		}
	}
	return indexes
}

func freeAdjacentSpaces(idx int, board *Board) []int {
	pos := positions[idx]
	adjacentPos := [8]Pos{
		Pos{pos.X - 1, pos.Y - 1},
		Pos{pos.X - 1, pos.Y},
		Pos{pos.X - 1, pos.Y + 1},
		Pos{pos.X, pos.Y - 1},
		Pos{pos.X, pos.Y + 1},
		Pos{pos.X + 1, pos.Y - 1},
		Pos{pos.X + 1, pos.Y},
		Pos{pos.X + 1, pos.Y + 1},
	}
	free := []int{}
	for _, pos := range adjacentPos {
		idx := pos.getBoardIdx()
		if idx != -1 {
			if board.Pieces[idx] == nil {
				free = append(free, idx)
			}
		}
	}
	return free
}

// returns indexes of all pieces which can dodge (under threat and has a free adjacent square)
func dodgeablePieces(color string, board *Board) []int {
	indexes := []int{}
	for i, p := range board.Pieces {
		if p != nil && (color == none || p.Color == color) && p.Damage > 0 {
			// find free space
			if len(freeAdjacentSpaces(i, board)) > 0 {
				indexes = append(indexes, i)
			}
		}
	}
	return indexes
}

// returns index of all pieces which have status effects (positive or negative)
func statusEffectedPieces(color string, board *Board) []int {
	indexes := []int{}
	for i, p := range board.Pieces {
		if p != nil && (color == none || p.Color == color) && p.Status != nil {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func hasFreeSpace(color string, board *Board) bool {
	var side []*Piece
	half := nColumns * nRows / 2
	if color == black {
		side = board.Pieces[half:]
	} else {
		side = board.Pieces[:half]
	}
	for _, p := range side {
		if p == nil {
			return true
		}
	}
	return false
}

func pieceCount(color string, board *Board) int {
	count := 0
	for _, p := range board.Pieces {
		if p != nil && p.Color == color {
			count++
		}
	}
	return count
}

// returns indexes of all pawns which can be toggled
func advanceablePieces(board *Board) []int {
	indexes := []int{}
	for i, p := range board.Pieces {
		if p != nil {
			if p.Color == white {
				j := i + nColumns
				if j < len(board.Pieces) && board.Pieces[j] == nil {
					indexes = append(indexes, i)
				}
			} else {
				j := i - nColumns
				if j >= 0 && board.Pieces[j] == nil {
					indexes = append(indexes, i)
				}
			}
		}
	}
	return indexes
}

func (m *Match) clickCard(player string, public *PublicState, private *PrivateState, cardIdx int) {
	switch m.Phase {
	case mainPhase:
		// ignore if not the player's turn
		if player != m.Turn {
			return
		}
		if private.PlayableCards[cardIdx] {
			if cardIdx == private.SelectedCard {
				private.SelectedCard = -1
				highlightsOff(private.Highlights[:])
			} else {
				card := private.Cards[cardIdx]
				private.SelectedCard = cardIdx
				idxs := validCardPositions(card.Name, player, m, &m.Board)
				dimAllBut(idxs, private.Highlights[:])
			}
		}
	}
}

// assumes we have already checked that the move is playable
// return true if force combat
func playCard(m *Match, card string, player string, public *PublicState, p Pos, board *Board) bool {
	piece := getPieceSafe(p, board)
	switch card {
	case castleCard:
		// find rook of same color as clicked king
		var rookPiece *Piece
		for _, p := range board.Pieces {
			if p != nil && p.Name == rook && p.Color == piece.Color {
				rookPiece = p
				break
			}
		}
		swap := *rookPiece
		*rookPiece = *piece
		*piece = swap
	case reclaimVassalCard:
		switch piece.Name {
		case bishop:
			public.BishopPlayed = false
		case knight:
			public.KnightPlayed = false
		case rook:
			public.RookPlayed = false
		}
		removePieceAt(p, board)
	case swapFrontLinesCard:
		frontIdx := (nRows/2 - 1) * nColumns
		midIdx := (nRows/2 - 2) * nColumns
		if piece.Color == black {
			frontIdx = (nRows / 2) * nColumns
			midIdx = (nRows/2 + 1) * nColumns
		}
		for i := 0; i < nColumns; i++ {
			swapBoardIndex(frontIdx, midIdx, board)
			frontIdx++
			midIdx++
		}
	case removePawnCard:
		public, _ := m.states(piece.Color)
		removePieceAt(p, board)
		public.NumPawns--
	case forceCombatCard:
		return true
	case dispellCard:
		piece.Status = nil
	case dodgeCard:
		idx := p.getBoardIdx()
		for _, val := range dodgeablePieces(player, board) {
			if val == idx {
				free := freeAdjacentSpaces(idx, board)
				newIdx := free[rand.Intn(len(free))]
				swapBoardIndex(idx, newIdx, board)
				break
			}
		}
	case mirrorCard:
		// (assumes board has even number of rows)
		row := 0
		if piece.Color == black {
			row = (nRows / 2)
		}
		for i := 0; i < (nRows / 2); i++ {
			idx := row * nColumns
			other := idx + nColumns - 1
			for j := 0; j < (nColumns / 2); j++ {
				swapBoardIndex(idx, other, board)
				idx++
				other--
			}
			row++
		}
	case healCard:
		piece.HP += healCardAmount
		switch piece.Name {
		case rook:
			public.Rook.HP += healCardAmount
		case knight:
			public.Knight.HP += healCardAmount
		case bishop:
			public.Bishop.HP += healCardAmount
		}
	case poisonCard:
		neg := pieceNegativeStatus(piece)
		neg.Poison += poisonAmount
	case togglePawnCard:
		idx := p.getBoardIdx()
		for _, val := range toggleablePawns(board) {
			if idx == val {
				const whiteMid = nRows/2 - 2
				const whiteFront = whiteMid + 1
				const blackFront = whiteMid + 2
				const blackMid = whiteMid + 3
				newPos := p
				switch p.Y {
				case whiteMid:
					newPos.Y = whiteFront
				case whiteFront:
					newPos.Y = whiteMid
				case blackFront:
					newPos.Y = blackMid
				case blackMid:
					newPos.Y = blackFront
				}
				swapBoardIndex(idx, newPos.getBoardIdx(), board)
				break
			}
		}
	case nukeCard:
		// inflict lesser damage on all within 2 squares
		minX, maxX := p.X-2, p.X+2
		minY, maxY := p.Y-2, p.Y+2
		for x := minX; x <= maxX; x++ {
			for y := minY; y <= maxY; y++ {
				target := Pos{x, y}
				if target == p {
					continue
				}
				inflictDamage(target.getBoardIdx(), nukeDamageLesser, board, public, m)
			}
		}
		// inflict (full - lesser) on all within 1 square (so these squares hit a second time)
		minX++
		maxX--
		minY++
		maxY--
		for x := minX; x <= maxX; x++ {
			for y := minY; y <= maxY; y++ {
				target := Pos{x, y}
				if target == p {
					continue
				}
				inflictDamage(target.getBoardIdx(), nukeDamageFull-nukeDamageLesser, board, public, m)
			}
		}
	case vulnerabilityCard:
		neg := pieceNegativeStatus(piece)
		neg.Vulnerability += vulnerabilityDuration
	case amplifyCard:
		positive := piecePositiveStatus(piece)
		positive.Amplify += amplifyDuration
	case transparencyCard:
		neg := pieceNegativeStatus(piece)
		neg.Transparent += transparencyDuration
	case stunVassalCard:
		positive := piecePositiveStatus(piece)
		negative := pieceNegativeStatus(piece)
		positive.DamageImmune += stunVassalDuration
		negative.Distracted += stunVassalDuration
		negative.Unreclaimable += stunVassalDuration
	case enrageCard:
		neg := pieceNegativeStatus(piece)
		neg.Enraged += enrageDuration
	case armorCard:
		positive := piecePositiveStatus(piece)
		positive.Armor += armorAmount
	case shoveCard:
		idx := p.getBoardIdx()
		for _, val := range shoveablePieces(board) {
			if idx == val {
				var newIdx int
				if piece.Color == black {
					newIdx = idx + nColumns
				} else {
					newIdx = idx - nColumns
				}
				swapBoardIndex(idx, newIdx, board)
				break
			}
		}
	case advanceCard:
		idx := p.getBoardIdx()
		for _, val := range advanceablePieces(board) {
			if idx == val {
				var newIdx int
				if piece.Color == white {
					newIdx = idx + nColumns
				} else {
					newIdx = idx - nColumns
				}
				swapBoardIndex(idx, newIdx, board)
				break
			}
		}
	case summonPawnCard:
		SpawnSinglePawn(player, public, false, board)
	case resurrectVassalCard:
		// should be the case that only one vassal is dead (because otherwise the game would be over already)
		if public.Bishop.HP <= 0 {
			public.Bishop.HP = resurrectVassalRestoreHP
			public.BishopPlayed = false
		} else if public.Knight.HP <= 0 {
			public.Knight.HP = resurrectVassalRestoreHP
			public.KnightPlayed = false
		} else if public.Rook.HP <= 0 {
			public.Rook.HP = resurrectVassalRestoreHP
			public.RookPlayed = false
		}
	case bishop, knight, rook, queen, jester:
		switch card {
		case bishop:
			setPiece(p, *public.Bishop, board)
			public.BishopPlayed = true
		case knight:
			setPiece(p, *public.Knight, board)
			public.KnightPlayed = true
		case rook:
			setPiece(p, *public.Rook, board)
			public.RookPlayed = true
		case queen:
			setPiece(p, Piece{queen, player, queenHP, queenAttack, 0, nil}, board)
		case jester:
			setPiece(p, Piece{jester, player, jesterHP, jesterAttack, 0, nil}, board)
		}
	}
	return false
}

func validCardPositions(cardName string, color string, m *Match, board *Board) []int {
	idxs := []int{}
	switch cardName {
	case castleCard:
		if m.WhitePublic.RookPlayed && m.BlackPublic.RookPlayed {
			idxs = kingIdxs(none, board)
		} else if m.WhitePublic.RookPlayed {
			idxs = kingIdxs(white, board)
		} else if m.BlackPublic.RookPlayed {
			idxs = kingIdxs(black, board)
		}
	case removePawnCard:
		idxs = pawnIdxs(none, board)
	case dodgeCard:
		idxs = dodgeablePieces(color, board)
	case forceCombatCard:
		idxs = kingIdxs(color, board)
	case dispellCard:
		idxs = statusEffectedPieces(none, board)
	case mirrorCard:
		idxs = kingIdxs(none, board)
	case togglePawnCard:
		idxs = toggleablePawns(board)
	case nukeCard:
		idxs = kingIdxs(none, board)
	case vulnerabilityCard:
		idxs = pieceIdxs(otherColor(color), board)
	case amplifyCard:
		idxs = pieceIdxs(color, board)
	case transparencyCard:
		idxs = pieceIdxs(otherColor(color), board)
	case stunVassalCard:
		idxs = vassalIdxs(otherColor(color), board)
	case armorCard:
		idxs = pieceIdxs(color, board)
		// remove king's idx
		for i, idx := range idxs {
			if board.Pieces[idx].Name == king {
				idxs = append(idxs[:i], idxs[i+1:]...)
				break
			}
		}
	case enrageCard:
		idxs = pieceIdxs(otherColor(color), board)
	case shoveCard:
		idxs = shoveablePieces(board)
	case advanceCard:
		idxs = advanceablePieces(board)
	case summonPawnCard:
		idxs = kingIdxs(color, board)
	case resurrectVassalCard:
		idxs = kingIdxs(color, board)
	case healCard:
		idxs = pieceIdxs(color, board)
		// remove king's idx
		for i, idx := range idxs {
			if board.Pieces[idx].Name == king {
				idxs = append(idxs[:i], idxs[i+1:]...)
				break
			}
		}
	case poisonCard:
		idxs = pieceIdxs(otherColor(color), board)
		// remove king's idx
		for i, idx := range idxs {
			if board.Pieces[idx].Name == king {
				idxs = append(idxs[:i], idxs[i+1:]...)
				break
			}
		}
	case swapFrontLinesCard:
		idxs = kingIdxs(none, board)
	case reclaimVassalCard:
		idxs = vassalIdxs(color, board)
	case rook, bishop, knight, queen, jester:
		idxs = freeIdxs(color, board)
	}
	return idxs
}

func canPlayCard(m *Match, card string, player string, public *PublicState, p Pos, board *Board) bool {
	piece := getPieceSafe(p, board)
	switch card {
	case castleCard:
		if piece == nil || piece.Name != king {
			return false
		}
		// find rook of same color as clicked king
		var rookPiece *Piece
		for _, p := range board.Pieces {
			if p != nil && p.Name == rook && p.Color == piece.Color {
				rookPiece = p
				break
			}
		}
		if rookPiece == nil {
			return false // no rook of matching color found
		}
	case reclaimVassalCard:
		if piece == nil || piece.Color != player {
			return false
		}
		switch piece.Name {
		case bishop:
		case knight:
		case rook:
		default:
			return false
		}
	case swapFrontLinesCard:
		if piece == nil || piece.Name != king {
			return false
		}
	case removePawnCard:
		if piece == nil || piece.Name != pawn {
			return false
		}
	case forceCombatCard:
		if piece == nil || piece.Name != king || piece.Color != player {
			return false
		}
	case dispellCard:
		if piece == nil || piece.Status == nil {
			return false
		}
	case dodgeCard:
		if piece == nil || piece.Color != player {
			return false
		}
		idx := p.getBoardIdx()
		for _, val := range dodgeablePieces(player, board) {
			if val == idx {
				return true
			}
		}
		return false
	case mirrorCard:
		// (assumes board has even number of rows)
		if piece == nil || piece.Name != king {
			return false
		}
	case healCard:
		if piece == nil || piece.Name == king || piece.Color != player {
			return false
		}
	case poisonCard:
		if piece == nil || piece.Color == player || piece.Name == king {
			return false
		}
	case togglePawnCard:
		if piece == nil || piece.Name != pawn {
			return false
		}
		idx := p.getBoardIdx()
		for _, val := range toggleablePawns(board) {
			if idx == val {
				return true
			}
		}
		return false
	case nukeCard:
		if piece == nil || piece.Name != king {
			return false
		}
	case vulnerabilityCard:
		if piece == nil || piece.Color == player {
			return false
		}
	case amplifyCard:
		if piece == nil || piece.Color != player {
			return false
		}
	case transparencyCard:
		if piece == nil || piece.Color == player {
			return false
		}
	case stunVassalCard:
		if piece == nil || piece.Color == player ||
			(piece.Name != knight && piece.Name != bishop && piece.Name != rook) {
			return false
		}
	case enrageCard:
		if piece == nil || piece.Color == player {
			return false
		}
	case armorCard:
		if piece == nil || piece.Color != player || piece.Name == king {
			return false
		}
	case shoveCard:
		if piece == nil {
			return false
		}
		idx := p.getBoardIdx()
		for _, val := range shoveablePieces(board) {
			if idx == val {
				return true
			}
		}
		return false
	case advanceCard:
		if piece == nil {
			return false
		}
		idx := p.getBoardIdx()
		for _, val := range advanceablePieces(board) {
			if idx == val {
				return true
			}
		}
		return false
	case summonPawnCard:
		if piece == nil || piece.Name != king || piece.Color != player {
			return false
		}
		return SpawnSinglePawn(player, public, true, board)
	case resurrectVassalCard:
		if piece == nil || piece.Name != king || piece.Color != public.Color {
			return false
		}
	case bishop, knight, rook, queen, jester:
		// ignore clicks on occupied spaces
		if getPieceSafe(p, board) != nil {
			return false
		}
		// square must be on player's side of board
		if player == white && p.Y >= nColumns/2 {
			return false
		}
		if player == black && p.Y < nColumns/2 {
			return false
		}
	}
	return true
}

func (m *Match) clickBoard(player string, public *PublicState, private *PrivateState, p Pos, board *Board) (newTurn bool, notifyOpponent bool) {
	switch m.Phase {
	case mainPhase:
		// ignore if not the player's turn and/or no card selected
		if player != m.Turn || private.SelectedCard == -1 {
			return
		}
		card := private.Cards[private.SelectedCard]
		if !canPlayCard(m, card.Name, player, public, p, board) {
			return
		}
		forceCombat := playCard(m, card.Name, player, public, p, board)
		m.Log = append(m.Log, player+" played "+card.Name)
		private.RemoveCard(private.SelectedCard)
		m.PlayableCards(board)
		if forceCombat {
			m.PassPrior = true
			m.EndTurn(true, player)
		} else {
			m.EndTurn(false, player)
		}
		newTurn = true
		notifyOpponent = true
	case kingPlacementPhase:
		if public.KingPlayed {
			break
		}
		// ignore clicks on occupied spaces
		if getPieceSafe(p, board) != nil {
			break
		}
		// square must be on player's side of board
		if player == white && p.Y >= nColumns/2 {
			break
		}
		if player == black && p.Y < nColumns/2 {
			break
		}
		public.KingPlayed = true
		m.Log = append(m.Log, player+" played King")
		private.KingPos = &p
		newTurn = m.EndKingPlacement()
		notifyOpponent = true
	}
	return
}

func piecePositiveStatus(p *Piece) *PiecePositiveStatus {
	status := p.Status
	if status == nil {
		status = &PieceStatus{}
		p.Status = status
	}
	if status.Positive == nil {
		status.Positive = &PiecePositiveStatus{}
	}
	return status.Positive
}

func pieceNegativeStatus(p *Piece) *PieceNegativeStatus {
	status := p.Status
	if status == nil {
		status = &PieceStatus{}
		p.Status = status
	}

	if status.Negative == nil {
		status.Negative = &PieceNegativeStatus{}
	}
	return status.Negative
}

// inflict damage on piece at index
// checks for win condition if piece is killed
// does nothing if no piece at index
// does nothing if index is out of bounds
func inflictDamage(idx int, dmg int, board *Board, public *PublicState, m *Match) {
	if idx < 0 || idx >= (nColumns*nRows) {
		return
	}
	p := board.Pieces[idx]
	if p == nil {
		return
	}
	p.HP -= dmg
	switch p.Name {
	case king:
		public.King.HP -= dmg
	case rook:
		public.Rook.HP -= dmg
	case bishop:
		public.Bishop.HP -= dmg
	case knight:
		public.Knight.HP -= dmg
	}
	if p.HP < 0 {
		board.Pieces[idx] = nil
		switch p.Name {
		case king:
			public.KingPlayed = false
		case rook:
			public.RookPlayed = false
		case bishop:
			public.BishopPlayed = false
		case knight:
			public.KnightPlayed = false
		case pawn:
			public.NumPawns--
			if public.NumPawns < 0 {
				public.NumPawns = 0
			}
		}
		m.checkWinCondition()
	}
}

// inflict damage on piece at index
// checks for win condition if piece is killed
// does nothing if no piece at index
// does nothing if index is out of bounds
func inflictTempDamage(idx int, dmg int, board *Board) {
	if idx < 0 || idx >= (nColumns*nRows) {
		return
	}
	p := board.Pieces[idx]
	if p == nil {
		return
	}
	p.HP -= dmg

	if p.HP < 0 {
		board.Pieces[idx] = nil
	}
}

// sets match state to gameover if winner or draw
func (m *Match) checkWinCondition() bool {
	b, w := m.BlackPublic, m.WhitePublic
	whiteDeadVassals := 0
	if w.Knight.HP <= 0 {
		whiteDeadVassals++
	}
	if w.Bishop.HP <= 0 {
		whiteDeadVassals++
	}
	if w.Rook.HP <= 0 {
		whiteDeadVassals++
	}
	blackDeadVassals := 0
	if b.Knight.HP <= 0 {
		blackDeadVassals++
	}
	if b.Bishop.HP <= 0 {
		blackDeadVassals++
	}
	if b.Rook.HP <= 0 {
		blackDeadVassals++
	}
	whiteLose := w.King.HP <= 0 || whiteDeadVassals >= 2
	blackLose := b.King.HP <= 0 || blackDeadVassals >= 2
	if whiteLose && blackLose {
		m.Winner = draw
		m.Phase = gameoverPhase
		return true
	} else if whiteLose {
		m.Winner = black
		m.Phase = gameoverPhase
		return true
	} else if blackLose {
		m.Winner = white
		m.Phase = gameoverPhase
		return true
	}
	return false
}

// panics if i or j are out of bounds
func swapBoardIndex(i, j int, b *Board) {
	if i == j {
		return
	}
	b.PiecesActual[i], b.PiecesActual[j] = b.PiecesActual[j], b.PiecesActual[i]
	if b.Pieces[i] == nil && b.Pieces[j] != nil {
		b.Pieces[i] = &b.PiecesActual[i]
		b.Pieces[j] = nil
	} else if b.Pieces[i] != nil && b.Pieces[j] == nil {
		b.Pieces[i] = nil
		b.Pieces[j] = &b.PiecesActual[j]
	}
}

func ReclaimPieces(board *Board, whitePublic *PublicState, blackPublic *PublicState) {
	for i, piece := range board.Pieces {
		if piece != nil {
			public := whitePublic
			if piece.Color == black {
				public = blackPublic
			}
			switch piece.Name {
			case king:
				*public.King = board.PiecesActual[i]
				removePieceAt(positions[i], board)
				public.KingPlayed = false
			case bishop:
				*public.Bishop = board.PiecesActual[i]
				removePieceAt(positions[i], board)
				public.BishopPlayed = false
			case knight:
				*public.Knight = board.PiecesActual[i]
				removePieceAt(positions[i], board)
				public.KnightPlayed = false
			case rook:
				*public.Rook = board.PiecesActual[i]
				removePieceAt(positions[i], board)
				public.RookPlayed = false
				public.Rook.HP += reclaimHealRook
				if public.Rook.HP > rookHP {
					public.Rook.HP = rookHP
				}
			}
		}
	}
}

func (m *Match) EndRound() {
	m.LastMoveTime = time.Now().UnixNano()
	m.Round++
	m.Log = append(m.Log, "Round "+strconv.Itoa(m.Round))

	ReclaimPieces(&m.Board, &m.WhitePublic, &m.BlackPublic)

	if m.FirstTurnColor == black {
		m.Turn = white
		m.FirstTurnColor = white
	} else {
		m.Turn = black
		m.FirstTurnColor = black
	}

	m.MaxRank++

	m.PassPrior = false

	m.WhitePrivate.Cards = drawCards(m.WhitePrivate.Cards, &m.WhitePublic, m.DevMode, m.MaxRank)
	m.BlackPrivate.Cards = drawCards(m.BlackPrivate.Cards, &m.BlackPublic, m.DevMode, m.MaxRank)
	m.WhitePrivate.SelectedCard = -1
	m.BlackPrivate.SelectedCard = -1

	SpawnPawns(false, &m.Board, &m.WhitePublic, &m.BlackPublic, &m.Log)
	tickdownStatusEffects(true, &m.Board)
	m.UpdateStatusAndDamage()
	m.PlayableCards(&m.Board)

	m.Phase = kingPlacementPhase
	if m.WhiteAI {
		public, private := m.states(white)
		pos := kingPlacementAI(white, &m.Board)
		private.KingPos = &pos
		public.KingPlayed = true
		m.Log = append(m.Log, "white played King")
		highlightsOff(m.WhitePrivate.Highlights[:])
	} else {
		dimAllButFree(white, &m.Board, m.WhitePrivate.Highlights[:])
	}
	if m.BlackAI {
		public, private := m.states(black)
		pos := kingPlacementAI(black, &m.Board)
		private.KingPos = &pos
		public.KingPlayed = true
		m.Log = append(m.Log, "black played King")
		highlightsOff(m.BlackPrivate.Highlights[:])
	} else {
		dimAllButFree(black, &m.Board, m.BlackPrivate.Highlights[:])
	}
}

func tickdownStatusEffects(postReclaim bool, board *Board) {
	for _, p := range board.Pieces {
		if p != nil {
			status := p.Status
			if status != nil {
				if postReclaim {
					if status.Negative != nil {
						neg := status.Negative
						if neg.Unreclaimable > 0 {
							neg.Unreclaimable--
						}
						if *neg == (PieceNegativeStatus{}) {
							status.Negative = nil
						}
					}
				} else {
					if status.Negative != nil {
						neg := status.Negative
						if neg.Vulnerability > 0 {
							neg.Vulnerability--
						}
						if neg.Enraged > 0 {
							neg.Enraged--
						}
						if neg.Distracted > 0 {
							neg.Distracted--
						}
						if neg.Transparent > 0 {
							neg.Transparent--
						}
						if *neg == (PieceNegativeStatus{}) {
							status.Negative = nil
						}
					}
					if status.Positive != nil {
						pos := status.Positive
						if pos.Amplify > 0 {
							pos.Amplify--
						}
						if pos.DamageImmune > 0 {
							pos.DamageImmune--
						}
						if *pos == (PiecePositiveStatus{}) {
							status.Positive = nil
						}
					}
				}
				if p.Status.Negative == nil && p.Status.Positive == nil {
					p.Status = nil
				}
			}
		}
	}
}

func dimAllButFree(color string, board *Board, highlights []int) {
	halfIdx := len(board.Pieces) / 2
	if color == black {
		for i, piece := range board.Pieces {
			if i < halfIdx {
				highlights[i] = highlightDim
			} else if piece == nil {
				highlights[i] = highlightOff
			} else {
				highlights[i] = highlightDim
			}
		}
	} else {
		halfIdx := len(board.Pieces) / 2
		for i, piece := range board.Pieces {
			if i >= halfIdx {
				highlights[i] = highlightDim
			} else if piece == nil {
				highlights[i] = highlightOff
			} else {
				highlights[i] = highlightDim
			}
		}
	}
}

// color none leaves pieces of both colors undimmed
func (p *PrivateState) dimAllButPieces(color string, board *Board) {
	for i, piece := range board.Pieces {
		if piece != nil && (piece.Color == color || color == none) {
			p.Highlights[i] = highlightOff
		} else {
			p.Highlights[i] = highlightDim
		}
	}
}

func highlightsOff(highlights []int) {
	for i := range highlights {
		highlights[i] = highlightOff
	}
}

func highlightPosOn(pos Pos, highlights []int) {
	idx := pos.getBoardIdx()
	highlights[idx] = highlightOn
}

func highlightPosOff(pos Pos, highlights []int) {
	idx := pos.getBoardIdx()
	highlights[idx] = highlightOff
}

// dims all squares but for specified indexes
// (doesn't assume indexes are in ascending order)
func dimAllBut(indexes []int, highlights []int) {
	for i := range highlights {
		highlights[i] = highlightDim
		for _, idx := range indexes {
			if i == idx {
				highlights[i] = highlightOff
			}
		}
	}
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func intInSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func (m *Match) UpdateStatusAndDamage() {
	CalculateSquareStatus(&m.Board, m.SquareStatuses[:], m.SquareStatusesDirect[:])
	CalculateDamage(&m.Board, m.SquareStatuses[:])
}

func (m *Match) UpdateStatusAndDamageTemp() {
	CalculateSquareStatus(&m.BoardTemp, m.tempSquareStatuses[:], m.SquareStatusesDirect[:])
	CalculateDamage(&m.BoardTemp, m.tempSquareStatuses[:])
}

// generate m.Combined from (m.Direct + square status effects from the pieces)
func CalculateSquareStatus(board *Board, squareStatuses []SquareStatus, squareStatusesDirect []SquareStatus) {
	copy(squareStatuses, squareStatusesDirect)
	// get status effects from pieces
	for i, piece := range board.Pieces {
		if piece != nil {
			switch piece.Name {
			case jester:
				pos := positions[i]
				y := pos.Y + 1
				if piece.Color == black {
					y = pos.Y - 1
				}
				indexes := [5]int{
					getBoardIdx(pos.X-1, pos.Y), // left of jester
					getBoardIdx(pos.X+1, pos.Y), // right
					getBoardIdx(pos.X-1, y),     // front left
					getBoardIdx(pos.X, y),       // front
					getBoardIdx(pos.X+1, y),     // front right
				}
				for _, idx := range indexes {
					if idx != -1 {
						status := &squareStatuses[idx]
						if status.Negative == nil {
							status.Negative = &SquareNegativeStatus{Distracted: true}
						}
					}
				}
			}
		}
	}
}

// returns true if both kings are now down
func (m *Match) EndKingPlacement() bool {
	if m.WhitePublic.KingPlayed && m.BlackPublic.KingPlayed {
		m.LastMoveTime = time.Now().UnixNano()
		highlightsOff(m.WhitePrivate.Highlights[:])
		highlightsOff(m.BlackPrivate.Highlights[:])
		pos := m.WhitePrivate.KingPos
		if pos != nil {
			setPiece(*pos, *m.WhitePublic.King, &m.Board)
			m.WhitePrivate.KingPos = nil
		}
		pos = m.BlackPrivate.KingPos
		if pos != nil {
			setPiece(*pos, *m.BlackPublic.King, &m.Board)
			m.BlackPrivate.KingPos = nil
		}
		m.UpdateStatusAndDamage()
		m.Phase = mainPhase
		m.PlayableCards(&m.Board)
		if m.BlackAI && m.Turn == black {
			playTurnAI(black, m)
		} else if m.WhiteAI && m.Turn == white {
			playTurnAI(white, m)
		}
		return true
	}
	return false
}

func (p *PrivateState) dimUnreclaimable(board *Board) {
	for i, piece := range board.Pieces {
		if piece != nil {
			if piece.isUnreclaimable() {
				p.Highlights[i] = highlightDim
			}
		}
	}
}

// pass = if turn is ending by passing; player = color whose turn is ending
func (m *Match) EndTurn(pass bool, player string) {
	m.LastMoveTime = time.Now().UnixNano()
	m.UpdateStatusAndDamage()

	if pass && m.PassPrior { // if players both pass in succession, do combat
		board := &m.Board
		InflictDamage(board, &m.WhitePublic, &m.BlackPublic)

		if !m.checkWinCondition() {
			tickdownStatusEffects(false, board)
			m.EndRound()
		}
	} else {
		if m.Turn == black {
			m.Turn = white
		} else {
			m.Turn = black
		}
		m.PassPrior = pass

		m.WhitePrivate.SelectedCard = -1
		m.BlackPrivate.SelectedCard = -1
		highlightsOff(m.WhitePrivate.Highlights[:])
		highlightsOff(m.BlackPrivate.Highlights[:])

		if m.BlackAI && m.Turn == black {
			playTurnAI(black, m)
		} else if m.WhiteAI && m.Turn == white {
			playTurnAI(white, m)
		}
	}
}

func drawCards(existing []Card, public *PublicState, devMode bool, maxRank int) []Card {
	// remove vassal cards from existing hand
	i := 0
loop:
	for ; i < len(existing); i++ {
		switch existing[i].Name {
		case king, bishop, knight, rook:
		default:
			break loop
		}
	}
	existing = existing[i:]

	stock := []Card{}
	if public.Bishop.HP > 0 && !public.BishopPlayed {
		stock = append(stock, Card{bishop, bishopMana})
	}
	if public.Knight.HP > 0 && !public.KnightPlayed {
		stock = append(stock, Card{knight, knightMana})
	}
	if public.Rook.HP > 0 && !public.RookPlayed {
		stock = append(stock, Card{rook, rookMana})
	}

	additional := []Card{}
	if !devMode {
		diff := nCardsCap - len(stock) - len(existing)
		if diff >= nCardsPerRound {
			additional = randomCards(nCardsPerRound, maxRank)
		} else if diff > 0 {
			additional = randomCards(diff, maxRank)
		}
	}

	return append(append(stock, existing...), additional...)
}

func randomCards(n int, maxRank int) []Card {
	idx := maxRank
	if idx >= len(cardRankCount) {
		idx = len(cardRankCount) - 1
	}
	cardPoolSize := cardRankCount[idx]
	cards := make([]Card, n)
	for i := range cards {
		cards[i] = allCards[rand.Intn(cardPoolSize)]
	}
	return cards
}

func drawCommunalCards() []Card {
	return []Card{}
}

func (m *Match) IsOpen() bool {
	return (m.WhiteConn == nil || m.BlackConn == nil) && m.Winner == none
}

func (m *Match) IsBlackOpen() bool {
	return m.BlackPlayerID == "" && m.Winner == none && m.BlackAI == false
}

func (m *Match) IsWhiteOpen() bool {
	return m.WhiteConn == nil && m.Winner == none
}

func (m *Match) IsFull() bool {
	return (m.WhiteConn != nil && m.BlackConn != nil) && m.Winner == none
}

func (m *Match) IsFinished() bool {
	return m.Winner != none
}

// panics if out of bounds
func (p *PrivateState) RemoveCard(idx int) {
	p.Cards = append(p.Cards[:idx], p.Cards[idx+1:]...)
}

func (p *PrivateState) RemoveMatchingCard(card Card) {
	matchFound := false
	for i, c := range p.Cards {
		if matchFound {
			p.Cards[i-1] = p.Cards[i]
		} else if card == c {
			matchFound = true
		}
	}
	p.Cards = p.Cards[:len(p.Cards)-1]
}

func (p *Piece) isDamageImmune() bool {
	if p.Status == nil || p.Status.Positive == nil {
		return false
	}
	return p.Status.Positive.DamageImmune > 0
}

func (p *Piece) armorMitigation(attack int) int {
	if p.Status == nil || p.Status.Positive == nil {
		return attack
	}
	attack -= p.Status.Positive.Armor
	if attack < 0 {
		return 0
	}
	return attack
}

func (p *Piece) isTransparent() bool {
	if p.Status == nil || p.Status.Negative == nil {
		return false
	}
	return p.Status.Negative.Transparent > 0
}

func (p *Piece) getAmplifiedDamage() int {
	if p.Status == nil || p.Status.Positive == nil {
		return p.Attack
	}
	if p.Status.Positive.Amplify > 0 {
		return p.Attack * amplifyFactor
	}
	return p.Attack
}

func (p *Piece) isUnreclaimable() bool {
	if p.Status == nil || p.Status.Negative == nil {
		return false
	}
	return p.Status.Negative.Unreclaimable > 0
}

func (p *Piece) isDistracted() bool {
	if p.Status == nil || p.Status.Negative == nil {
		return false
	}
	return p.Status.Negative.Distracted > 0
}

func (p *Piece) isEnraged() bool {
	if p.Status == nil || p.Status.Negative == nil {
		return false
	}
	return p.Status.Negative.Enraged > 0
}

func (m *Match) processEvent(event string, player string, msg []byte) (notifyOpponent bool, newTurn bool) {
	if m.Phase == gameoverPhase {
		return
	}
	public, private := m.states(player)
	switch event {
	case "get_state":
		// doesn't change anything, just fetches current state
	case "ready":
		switch m.Phase {
		case readyUpPhase:
			public.Ready = true
			if m.BlackPublic.Ready && m.WhitePublic.Ready {
				m.Phase = kingPlacementPhase
				m.Round = 1 // by incrementing from 0, will sound new round fanfare
				m.LastMoveTime = time.Now().UnixNano()
			}
			notifyOpponent = true
		}
	case "time_expired":
		switch m.Phase {
		case mainPhase:
			turnElapsed := time.Now().UnixNano() - m.LastMoveTime
			remainingTurnTime := m.TurnTimer - turnElapsed
			if remainingTurnTime > 0 {
				break // ignore if time hasn't actually expired
			}
			// actual elapsed time is checked on server, but we rely upon clients to notify
			// (not ideal because both clients might fail, but then we have bigger problem)
			// Cheater could supress sending time_expired event from their client, but
			// opponent also sends the event (and has interest to do so).
			m.Log = append(m.Log, m.Turn+" passed")
			m.EndTurn(true, m.Turn)
			newTurn = true
			notifyOpponent = true
		case kingPlacementPhase:
			turnElapsed := time.Now().UnixNano() - m.LastMoveTime
			remainingTurnTime := m.TurnTimer - turnElapsed
			if remainingTurnTime > 0 {
				break // ignore if time hasn't actually expired
			}
			for _, color := range []string{black, white} {
				public, _ := m.states(color)
				if !public.KingPlayed {
					// randomly place king in free square
					// Because we must have reclaimed the King, there will always be a free square at this point
					pos, _ := RandomFreeSquare(color, &m.Board)
					setPiece(pos, *public.King, &m.Board)
					public.KingPlayed = true
					m.Log = append(m.Log, color+" played King")
				}
			}
			newTurn = m.EndKingPlacement()
			notifyOpponent = true
		}
	case "click_card":
		type ClickCardEvent struct {
			SelectedCard int
		}
		var event ClickCardEvent
		err := json.Unmarshal(msg, &event)
		if err != nil {
			fmt.Println("unmarshalling click_card error", err)
			break // todo: send error response
		}
		m.clickCard(player, public, private, event.SelectedCard)
	case "click_board":
		var pos Pos
		err := json.Unmarshal(msg, &pos)
		if err != nil {
			break // todo: send error response
		}
		newTurn, notifyOpponent = m.clickBoard(player, public, private, pos, &m.Board)
	case "pass":
		switch m.Phase {
		case mainPhase:
			if player != m.Turn {
				break // ignore if not the player's turn
			}
			if !public.KingPlayed {
				break // cannot pass when king has not been played
			}
			m.Log = append(m.Log, player+" passed")
			m.EndTurn(true, player)
			newTurn = true
			notifyOpponent = true
		}
	default:
		fmt.Println("bad event: ", event, msg) // todo: better error reporting
	}
	return
}
