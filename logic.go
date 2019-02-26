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

	public := &m.WhitePublic
	public.Color = white
	public.Other = &m.BlackPublic
	public.ManaCurrent = startingMana
	public.ManaMax = startingMana
	public.King = &Piece{king, white, kingHP, kingAttack, 0, nil}
	public.Bishop = &Piece{bishop, white, bishopHP, bishopAttack, 0, nil}
	public.Knight = &Piece{knight, white, knightHP, knightAttack, 0, nil}
	public.Rook = &Piece{rook, white, rookHP, rookAttack, 0, nil}

	public = &m.BlackPublic
	public.Color = black
	public.Other = &m.WhitePublic
	public.ManaCurrent = startingMana
	public.ManaMax = startingMana
	public.King = &Piece{king, black, kingHP, kingAttack, 0, nil}
	public.Bishop = &Piece{bishop, black, bishopHP, bishopAttack, 0, nil}
	public.Knight = &Piece{knight, black, knightHP, knightAttack, 0, nil}
	public.Rook = &Piece{rook, black, rookHP, rookAttack, 0, nil}

	m.Log = []string{"Round 1"}

	m.SpawnPawns(true)
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
			randomCards(nCardsFirstRound, m.BlackPublic.ManaMax)...)
		m.WhitePrivate.Cards = append(append([]Card{}, stock...),
			randomCards(nCardsFirstRound, m.WhitePublic.ManaMax)...)
	}

	m.BlackPrivate.Other = &m.WhitePrivate
	m.WhitePrivate.Other = &m.BlackPrivate

	m.BlackPrivate.dimAllButFree(black, m.Board[:])
	m.WhitePrivate.dimAllButFree(white, m.Board[:])

	m.PlayableCards()

	if m.BlackAI {
		public, private := m.states(black)
		pos := kingPlacementAI(black, m.Board[:])
		private.KingPos = &pos
		public.KingPlayed = true
		m.Log = append(m.Log, "black played King")
	}
	if m.WhiteAI {
		public, private := m.states(white)
		pos := kingPlacementAI(white, m.Board[:])
		private.KingPos = &pos
		public.KingPlayed = true
		m.Log = append(m.Log, "white played King")
	}
}

func getPiece(p Pos, board []*Piece) *Piece {
	return board[nColumns*p.Y+p.X]
}

// does not panic
func getPieceSafe(p Pos, board []*Piece) *Piece {
	if p.X < 0 || p.X >= nColumns || p.Y < 0 || p.Y >= nRows {
		return nil
	}
	return board[nColumns*p.Y+p.X]
}

func (m *Match) getTempPiece(p Pos) *Piece {
	return m.tempBoard[nColumns*p.Y+p.X]
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
func (m *Match) setPiece(p Pos, piece Piece) {
	idx := nColumns*p.Y + p.X
	m.pieces[idx] = piece
	m.Board[idx] = &m.pieces[idx]
}

func (m *Match) setTempPiece(p Pos, piece Piece) {
	idx := nColumns*p.Y + p.X
	m.tempPieces[idx] = piece
	m.tempBoard[idx] = &m.tempPieces[idx]
}

// panics if out of bounds
func (m *Match) removePieceAt(p Pos) {
	idx := nColumns*p.Y + p.X
	m.Board[idx] = nil
	m.pieces[idx] = Piece{}
}

// panics if out of bounds
func (m *Match) removeTempPieceAt(p Pos) {
	idx := nColumns*p.Y + p.X
	m.tempBoard[idx] = nil
	m.tempPieces[idx] = Piece{}
}

func (m *Match) RemoveNonPawns() {
	for i, p := range m.pieces {
		if p.Name != pawn {
			m.pieces[i] = Piece{}
			m.Board[i] = nil
		}
	}
}

func (m *Match) InflictDamage() {
	for i, p := range m.Board {
		if p != nil {
			p.HP -= p.Damage
			p.Damage = 0
			public := m.getPublic(p.Color)
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
				m.pieces[i] = Piece{}
				m.Board[i] = nil
			}
		}
	}
}

func (m *Match) CalculateDamage(board []*Piece, pieces []Piece, squareStatuses []SquareStatus) {
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
			hit := getPieceSafe(other, m.Board[:])
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
			hit := getPieceSafe(other, m.Board[:])
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
			hit := getPieceSafe(other, m.Board[:])
			if hit != nil && !hit.isDamageImmune() && (hit.Color != color || enraged) {
				hit.Damage += hit.armorMitigation(attack)
			}
		}
	}

	// reset all to 0
	for i := range pieces {
		pieces[i].Damage = 0
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
	for i, p := range board {
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

	for _, p := range board {
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

func (m *Match) SpawnSinglePawn(color string, public *PublicState, test bool) bool {
	if public.NumPawns == maxPawns {
		return false
	}
	n := 1
	offset := 1
	if color == black {
		offset = 3
	}
	columns := m.freePawnColumns(color)
	columns = randSelect(n, columns)
	n = len(columns)
	if n < 1 {
		return false
	}
	if test {
		return true
	}
	for _, v := range columns {
		m.setPiece(Pos{v, rand.Intn(2) + offset}, Piece{pawn, public.Color, pawnHP, pawnAttack, 0, nil})
	}
	public.NumPawns += n
	return true
}

func (m *Match) SpawnSinglePawnTemp(color string, public *PublicState) bool {
	if public.NumPawns == maxPawns {
		return false
	}
	n := 1
	offset := 1
	if color == black {
		offset = 3
	}
	columns := m.freePawnColumns(color)
	columns = randSelect(n, columns)
	n = len(columns)
	if n < 1 {
		return false
	}
	for _, v := range columns {
		m.setTempPiece(Pos{v, rand.Intn(2) + offset}, Piece{pawn, public.Color, pawnHP, pawnAttack, 0, nil})
	}
	return true
}

// returns indexes of the columns in which a new pawn can be placed
func (m *Match) freePawnColumns(color string) []int {
	var columns []int
	front := 2
	mid := 1
	if color == black {
		front = 3
		mid = 4
	}
	for i := 0; i < nColumns; i++ {
		if getPiece(Pos{i, front}, m.Board[:]) == nil && getPiece(Pos{i, mid}, m.Board[:]) == nil {
			columns = append(columns, i)
		}
	}
	return columns
}

// spawn n random pawns in free columns
func (m *Match) SpawnPawns(init bool) {
	public := &m.WhitePublic
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
		columns := m.freePawnColumns(public.Color)
		columns = randSelect(n, columns)
		n = len(columns)
		for _, v := range columns {
			m.setPiece(Pos{v, rand.Intn(2) + offset}, Piece{pawn, public.Color, pawnHP, pawnAttack, 0, nil})
		}
		public.NumPawns += n
		switch n {
		case 0:
			m.Log = append(m.Log, public.Color+" gained no pawns")
		case 1:
			m.Log = append(m.Log, public.Color+" gained 1 pawn")
		default:
			m.Log = append(m.Log, public.Color+" gained "+strconv.Itoa(n)+" pawns")
		}
		public = &m.BlackPublic
	}
}

// returns boolean true when no free slot
func (m *Match) RandomFreeSquare(player string) (Pos, bool) {
	// collect Pos of all free squares on player's side
	freeSquares := []Pos{}
	x := 0
	y := 0
	i := 0
	end := len(m.Board) / 2
	if player == black {
		y = nRows / 2
		i = end
		end = len(m.Board)
	}
	for ; i < end; i++ {
		if m.Board[i] == nil {
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

func (m *Match) PlayableCards() {
	// determine which cards are playable for each player given state of board
	public, private := m.states(white)
	for i := 0; i < 2; i++ {
		private.PlayableCards = make([]bool, len(private.Cards))
		for j, c := range private.Cards {
			private.PlayableCards[j] = false
			if public.ManaCurrent >= c.ManaCost {
				switch c.Name {
				case bishop, knight, rook, queen, jester:
					if hasFreeSpace(public.Color, m.Board[:]) {
						private.PlayableCards[j] = true
					}
				case forceCombatCard, mirrorCard, nukeCard, vulnerabilityCard, transparencyCard, amplifyCard, enrageCard, swapFrontLinesCard:
					private.PlayableCards[j] = true
				case dispellCard:
					if len(m.statusEffectedPieces(none)) > 0 {
						private.PlayableCards[j] = true
					}
				case stunVassalCard:
					if public.Other.KnightPlayed || public.Other.RookPlayed || public.Other.BishopPlayed {
						private.PlayableCards[j] = true
					}
				case dodgeCard:
					if len(m.dodgeablePieces(public.Color)) > 0 {
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
				case drainManaCard:
					if public.Other.ManaCurrent > 0 {
						private.PlayableCards[j] = true
					}
				case shoveCard:
					if len(m.shoveablePieces()) > 0 {
						private.PlayableCards[j] = true
					}
				case advanceCard:
					if len(m.advanceablePieces()) > 0 {
						private.PlayableCards[j] = true
					}
				case restoreManaCard:
					if public.ManaCurrent < public.ManaMax {
						private.PlayableCards[j] = true
					}
				case summonPawnCard:
					if public.NumPawns < maxPawns && len(m.freePawnColumns(public.Color)) > 0 {
						private.PlayableCards[j] = true
					}
				case resurrectVassalCard:
					if public.Bishop.HP <= 0 || public.Rook.HP <= 0 || public.Knight.HP <= 0 {
						private.PlayableCards[j] = true
					}
				case togglePawnCard:
					if len(m.toggleablePawns()) > 0 {
						private.PlayableCards[j] = true
					}
				case poisonCard:
					// only playable on enemy piece other than king
					if m.pieceCount(public.Other.Color) > 1 {
						private.PlayableCards[j] = true
					}
				case healCard, armorCard:
					// only playable on piece other than king
					if m.pieceCount(public.Color) > 1 {
						private.PlayableCards[j] = true
					}
				case removePawnCard:
					if public.NumPawns > 0 || public.Other.NumPawns > 0 {
						private.PlayableCards[j] = true
					}
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
func (m *Match) toggleablePawns() []int {
	indexes := []int{}
	start := (nRows / 2) * nColumns // first look for black toggleable pawns
	for i := 0; i < 2; i++ {
		for j := start; j < start+nColumns; j++ {
			k := j + nColumns
			a, b := m.Board[j], m.Board[k]
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
func (m *Match) shoveablePieces() []int {
	indexes := []int{}
	for i, p := range m.Board {
		if p != nil {
			if p.Color == black {
				j := i + nColumns
				if j < len(m.Board) && m.Board[j] == nil {
					indexes = append(indexes, i)
				}
			} else {
				j := i - nColumns
				if j >= 0 && m.Board[j] == nil {
					indexes = append(indexes, i)
				}
			}
		}
	}
	return indexes
}

func (m *Match) freeAdjacentSpaces(idx int) []int {
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
			if m.Board[idx] == nil {
				free = append(free, idx)
			}
		}
	}
	return free
}

// returns indexes of all pieces which can dodge (under threat and has a free adjacent square)
func (m *Match) dodgeablePieces(color string) []int {
	indexes := []int{}
	for i, p := range m.Board {
		if p != nil && (color == none || p.Color == color) && p.Damage > 0 {
			// find free space
			if len(m.freeAdjacentSpaces(i)) > 0 {
				indexes = append(indexes, i)
			}
		}
	}
	return indexes
}

// returns index of all pieces which have status effects (positive or negative)
func (m *Match) statusEffectedPieces(color string) []int {
	indexes := []int{}
	for i, p := range m.Board {
		if p != nil && (color == none || p.Color == color) && p.Status != nil {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func hasFreeSpace(color string, board []*Piece) bool {
	var side []*Piece
	half := nColumns * nRows / 2
	if color == black {
		side = board[half:]
	} else {
		side = board[:half]
	}
	for _, p := range side {
		if p == nil {
			return true
		}
	}
	return false
}

func (m *Match) pieceCount(color string) int {
	count := 0
	for _, p := range m.Board {
		if p != nil && p.Color == color {
			count++
		}
	}
	return count
}

// returns indexes of all pawns which can be toggled
func (m *Match) advanceablePieces() []int {
	indexes := []int{}
	for i, p := range m.Board {
		if p != nil {
			if p.Color == white {
				j := i + nColumns
				if j < len(m.Board) && m.Board[j] == nil {
					indexes = append(indexes, i)
				}
			} else {
				j := i - nColumns
				if j >= 0 && m.Board[j] == nil {
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
				private.highlightsOff()
			} else {
				card := private.Cards[cardIdx]
				private.SelectedCard = cardIdx
				if public.ManaCurrent >= card.ManaCost {
					idxs := validCardPositions(card.Name, player, m)
					private.dimAllBut(idxs)
				}
			}
		}
	}
}

// assumes we have already checked that the move is playable
func playCard(m *Match, card string, player string, public *PublicState, p Pos) {
	piece := getPieceSafe(p, m.Board[:])
	switch card {
	case castleCard:
		// find rook of same color as clicked king
		var rookPiece *Piece
		for _, p := range m.Board {
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
		m.removePieceAt(p)
	case swapFrontLinesCard:
		frontIdx := (nRows/2 - 1) * nColumns
		midIdx := (nRows/2 - 2) * nColumns
		if piece.Color == black {
			frontIdx = (nRows / 2) * nColumns
			midIdx = (nRows/2 + 1) * nColumns
		}
		for i := 0; i < nColumns; i++ {
			m.swapBoardIndex(frontIdx, midIdx)
			frontIdx++
			midIdx++
		}
	case removePawnCard:
		public, _ := m.states(piece.Color)
		m.removePieceAt(p)
		public.NumPawns--
	case forceCombatCard:
		m.PassPrior = true
		m.EndTurn(true, player)
	case dispellCard:
		piece.Status = nil
	case dodgeCard:
		idx := p.getBoardIdx()
		for _, val := range m.dodgeablePieces(player) {
			if val == idx {
				free := m.freeAdjacentSpaces(idx)
				newIdx := free[rand.Intn(len(free))]
				m.swapBoardIndex(idx, newIdx)
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
				m.swapBoardIndex(idx, other)
				idx++
				other--
			}
			row++
		}
	case drainManaCard:
		otherPublic := public.Other
		otherPublic.ManaCurrent -= drainManaAmount
		if otherPublic.ManaCurrent < 0 {
			otherPublic.ManaCurrent = 0
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
		neg := m.pieceNegativeStatus(piece)
		neg.Poison += poisonAmount
	case togglePawnCard:
		idx := p.getBoardIdx()
		for _, val := range m.toggleablePawns() {
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
				m.swapBoardIndex(idx, newPos.getBoardIdx())
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
				m.inflictDamage(target.getBoardIdx(), nukeDamageLesser)
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
				m.inflictDamage(target.getBoardIdx(), nukeDamageFull-nukeDamageLesser)
			}
		}
	case vulnerabilityCard:
		neg := m.pieceNegativeStatus(piece)
		neg.Vulnerability += vulnerabilityDuration
	case amplifyCard:
		positive := m.piecePositiveStatus(piece)
		positive.Amplify += amplifyDuration
	case transparencyCard:
		neg := m.pieceNegativeStatus(piece)
		neg.Transparent += transparencyDuration
	case stunVassalCard:
		positive := m.piecePositiveStatus(piece)
		negative := m.pieceNegativeStatus(piece)
		positive.DamageImmune += stunVassalDuration
		negative.Distracted += stunVassalDuration
		negative.Unreclaimable += stunVassalDuration
	case enrageCard:
		neg := m.pieceNegativeStatus(piece)
		neg.Enraged += enrageDuration
	case armorCard:
		positive := m.piecePositiveStatus(piece)
		positive.Armor += armorAmount
	case shoveCard:
		idx := p.getBoardIdx()
		for _, val := range m.shoveablePieces() {
			if idx == val {
				var newIdx int
				if piece.Color == black {
					newIdx = idx + nColumns
				} else {
					newIdx = idx - nColumns
				}
				m.swapBoardIndex(idx, newIdx)
				break
			}
		}
	case advanceCard:
		idx := p.getBoardIdx()
		for _, val := range m.advanceablePieces() {
			if idx == val {
				var newIdx int
				if piece.Color == white {
					newIdx = idx + nColumns
				} else {
					newIdx = idx - nColumns
				}
				m.swapBoardIndex(idx, newIdx)
				break
			}
		}
	case restoreManaCard:
		public.ManaCurrent = public.ManaMax + restoreManaMana // add cost of card because it gets subtracted later
	case summonPawnCard:
		m.SpawnSinglePawn(player, public, false)
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
			m.setPiece(p, *public.Bishop)
			public.BishopPlayed = true
		case knight:
			m.setPiece(p, *public.Knight)
			public.KnightPlayed = true
		case rook:
			m.setPiece(p, *public.Rook)
			public.RookPlayed = true
		case queen:
			m.setPiece(p, Piece{queen, player, queenHP, queenAttack, 0, nil})
		case jester:
			m.setPiece(p, Piece{jester, player, jesterHP, jesterAttack, 0, nil})
		}
	}
}

func validCardPositions(cardName string, color string, m *Match) []int {
	idxs := []int{}
	board := m.Board[:]
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
		idxs = m.dodgeablePieces(color)
	case forceCombatCard:
		idxs = kingIdxs(color, board)
	case dispellCard:
		idxs = m.statusEffectedPieces(none)
	case mirrorCard:
		idxs = kingIdxs(none, board)
	case drainManaCard:
		idxs = kingIdxs(otherColor(color), board)
	case togglePawnCard:
		idxs = m.toggleablePawns()
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
			if m.Board[idx].Name == king {
				idxs = append(idxs[:i], idxs[i+1:]...)
				break
			}
		}
	case enrageCard:
		idxs = pieceIdxs(otherColor(color), board)
	case shoveCard:
		idxs = m.shoveablePieces()
	case advanceCard:
		idxs = m.advanceablePieces()
	case restoreManaCard:
		idxs = kingIdxs(color, board)
	case summonPawnCard:
		idxs = kingIdxs(color, board)
	case resurrectVassalCard:
		idxs = kingIdxs(color, board)
	case healCard:
		idxs = pieceIdxs(color, board)
		// remove king's idx
		for i, idx := range idxs {
			if m.Board[idx].Name == king {
				idxs = append(idxs[:i], idxs[i+1:]...)
				break
			}
		}
	case poisonCard:
		idxs = pieceIdxs(otherColor(color), board)
		// remove king's idx
		for i, idx := range idxs {
			if m.Board[idx].Name == king {
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

func canPlayCard(m *Match, card string, player string, public *PublicState, p Pos) bool {
	piece := getPieceSafe(p, m.Board[:])
	switch card {
	case castleCard:
		if piece == nil || piece.Name != king {
			return false
		}
		// find rook of same color as clicked king
		var rookPiece *Piece
		for _, p := range m.Board {
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
		for _, val := range m.dodgeablePieces(player) {
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
	case drainManaCard:
		otherPublic := public.Other
		if piece == nil || piece.Name != king || piece.Color != otherPublic.Color {
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
		for _, val := range m.toggleablePawns() {
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
		for _, val := range m.shoveablePieces() {
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
		for _, val := range m.advanceablePieces() {
			if idx == val {
				return true
			}
		}
		return false
	case restoreManaCard:
		if piece == nil || piece.Name != king || piece.Color != player {
			return false
		}
	case summonPawnCard:
		if piece == nil || piece.Name != king || piece.Color != player {
			return false
		}
		return m.SpawnSinglePawn(player, public, true)
	case resurrectVassalCard:
		if piece == nil || piece.Name != king || piece.Color != public.Color {
			return false
		}
	case bishop, knight, rook, queen, jester:
		// ignore clicks on occupied spaces
		if getPieceSafe(p, m.Board[:]) != nil {
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

func (m *Match) clickBoard(player string, public *PublicState, private *PrivateState, p Pos) (newTurn bool, notifyOpponent bool) {
	switch m.Phase {
	case mainPhase:
		// ignore if not the player's turn and/or no card selected
		if player != m.Turn || private.SelectedCard == -1 {
			return
		}
		card := private.Cards[private.SelectedCard]
		if !canPlayCard(m, card.Name, player, public, p) {
			return
		}
		playCard(m, card.Name, player, public, p)
		public.ManaCurrent -= card.ManaCost
		m.Log = append(m.Log, player+" played "+card.Name)
		private.RemoveCard(private.SelectedCard)
		m.PlayableCards()
		m.EndTurn(false, player)
		newTurn = true
		notifyOpponent = true
	case reclaimPhase:
		piece := getPieceSafe(p, m.Board[:])
		if piece != nil && piece.Color == player && !piece.isUnreclaimable() {
			found := false
			selections := private.ReclaimSelections

			// unselect if already selected
			for i, selection := range selections {
				if selection == p {
					selections = append(selections[:i], selections[i+1:]...)
					private.highlightPosOff(p)
					found = true
				}
			}

			// select if not already selected
			if !found && len(selections) < maxReclaim {
				private.highlightPosOn(p)
				selections = append(selections, p)
			}

			private.ReclaimSelections = selections
		}
	case kingPlacementPhase:
		if public.KingPlayed {
			break
		}
		// ignore clicks on occupied spaces
		if getPieceSafe(p, m.Board[:]) != nil {
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

func (m *Match) piecePositiveStatus(p *Piece) *PiecePositiveStatus {
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

func (m *Match) pieceNegativeStatus(p *Piece) *PieceNegativeStatus {
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

func (m *Match) getPublic(color string) *PublicState {
	if color == black {
		return &m.BlackPublic
	} else {
		return &m.WhitePublic
	}
}

// inflict damage on piece at index
// checks for win condition if piece is killed
// does nothing if no piece at index
// does nothing if index is out of bounds
func (m *Match) inflictDamage(idx int, dmg int) {
	if idx < 0 || idx >= (nColumns*nRows) {
		return
	}
	p := m.Board[idx]
	if p == nil {
		return
	}
	p.HP -= dmg
	public := m.getPublic(p.Color)
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
		m.Board[idx] = nil
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
func (m *Match) inflictTempDamage(idx int, dmg int) {
	if idx < 0 || idx >= (nColumns*nRows) {
		return
	}
	p := m.tempBoard[idx]
	if p == nil {
		return
	}
	p.HP -= dmg

	if p.HP < 0 {
		m.tempBoard[idx] = nil
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
func (m *Match) swapBoardIndex(i, j int) {
	if i == j {
		return
	}
	m.pieces[i], m.pieces[j] = m.pieces[j], m.pieces[i]
	if m.Board[i] == nil && m.Board[j] != nil {
		m.Board[i] = &m.pieces[i]
		m.Board[j] = nil
	} else if m.Board[i] != nil && m.Board[j] == nil {
		m.Board[i] = nil
		m.Board[j] = &m.pieces[j]
	}
}

// panics if i or j are out of bounds
func (m *Match) swapTempBoardIndex(i, j int) {
	if i == j {
		return
	}
	m.tempPieces[i], m.tempPieces[j] = m.tempPieces[j], m.tempPieces[i]
	if m.tempBoard[i] == nil && m.tempBoard[j] != nil {
		m.tempBoard[i] = &m.tempPieces[i]
		m.tempBoard[j] = nil
	} else if m.Board[i] != nil && m.tempBoard[j] == nil {
		m.tempBoard[i] = nil
		m.tempBoard[j] = &m.tempPieces[j]
	}
}

func (m *Match) ReclaimPieces() {
	for _, color := range []string{white, black} {
		public, private := m.states(color)
		if public.ReclaimSelectionMade {
			for i, pos := range private.ReclaimSelections {
				if i >= 2 {
					break // in case more than two selections were sent, we ignore all but first two
				}
				p := getPieceSafe(pos, m.Board[:])
				if p != nil { // selection might be off the board or a square without a piece (todo: send error to client)
					// if reclaiming a king or vassal, add card back to hand
					// if reclaiming a rook, heal the rook
					switch p.Name {
					case king:
						*public.King = *m.Board[pos.getBoardIdx()]
						m.removePieceAt(pos)
						public.KingPlayed = false
					case bishop:
						*public.Bishop = *m.Board[pos.getBoardIdx()]
						m.removePieceAt(pos)
						public.BishopPlayed = false
					case knight:
						*public.Knight = *m.Board[pos.getBoardIdx()]
						m.removePieceAt(pos)
						public.KnightPlayed = false
					case rook:
						*public.Rook = *m.Board[pos.getBoardIdx()]
						m.removePieceAt(pos)
						public.RookPlayed = false
						public.Rook.HP += reclaimHealRook
						if public.Rook.HP > rookHP {
							public.Rook.HP = rookHP
						}
					case queen, jester:
						m.removePieceAt(pos)
					case pawn:
						m.removePieceAt(pos)
						public.NumPawns--
					default:

					}
				}
			}
		}
		private.ReclaimSelections = nil
		public.ReclaimSelectionMade = false
	}
}

func (m *Match) EndRound() {
	m.LastMoveTime = time.Now().UnixNano()
	m.Round++
	m.Log = append(m.Log, "Round "+strconv.Itoa(m.Round))

	if m.FirstTurnColor == black {
		m.Turn = white
		m.FirstTurnColor = white
	} else {
		m.Turn = black
		m.FirstTurnColor = black
	}

	m.BlackPublic.ManaMax++
	m.BlackPublic.ManaCurrent = m.BlackPublic.ManaMax

	m.WhitePublic.ManaMax++
	m.WhitePublic.ManaCurrent = m.WhitePublic.ManaMax

	m.PassPrior = false

	m.WhitePrivate.Cards = drawCards(m.WhitePrivate.Cards, &m.WhitePublic, m.DevMode)
	m.BlackPrivate.Cards = drawCards(m.BlackPrivate.Cards, &m.BlackPublic, m.DevMode)
	m.WhitePrivate.SelectedCard = -1
	m.BlackPrivate.SelectedCard = -1

	m.SpawnPawns(false)
	m.tickdownStatusEffects(true)
	m.UpdateStatusAndDamage()
	m.PlayableCards()

	if m.WhitePublic.KingPlayed && m.BlackPublic.KingPlayed {
		m.Phase = mainPhase
		m.WhitePrivate.highlightsOff()
		m.BlackPrivate.highlightsOff()
		if m.BlackAI && m.Turn == black {
			playTurnAI(black, m)
		} else if m.WhiteAI && m.Turn == white {
			playTurnAI(white, m)
		}
	} else {
		m.Phase = kingPlacementPhase
		if m.WhitePublic.KingPlayed {
			m.WhitePrivate.highlightsOff()
		} else {
			if m.WhiteAI {
				public, private := m.states(white)
				pos := kingPlacementAI(white, m.Board[:])
				private.KingPos = &pos
				public.KingPlayed = true
				m.Log = append(m.Log, "white played King")
				m.WhitePrivate.highlightsOff()
				m.EndKingPlacement()
			} else {
				m.WhitePrivate.dimAllButFree(white, m.Board[:])
			}
		}
		if m.BlackPublic.KingPlayed {
			m.BlackPrivate.highlightsOff()
		} else {
			if m.BlackAI {
				public, private := m.states(black)
				pos := kingPlacementAI(black, m.Board[:])
				private.KingPos = &pos
				public.KingPlayed = true
				m.Log = append(m.Log, "black played King")
				m.BlackPrivate.highlightsOff()
				m.EndKingPlacement()
			} else {
				m.BlackPrivate.dimAllButFree(black, m.Board[:])
			}
		}
	}
}

func (m *Match) tickdownStatusEffects(postReclaim bool) {
	for _, p := range m.Board {
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

func (p *PrivateState) dimAllButFree(color string, board []*Piece) {
	halfIdx := len(board) / 2
	if color == black {
		for i, piece := range board {
			if i < halfIdx {
				p.Highlights[i] = highlightDim
			} else if piece == nil {
				p.Highlights[i] = highlightOff
			} else {
				p.Highlights[i] = highlightDim
			}
		}
	} else {
		halfIdx := len(board) / 2
		for i, piece := range board {
			if i >= halfIdx {
				p.Highlights[i] = highlightDim
			} else if piece == nil {
				p.Highlights[i] = highlightOff
			} else {
				p.Highlights[i] = highlightDim
			}
		}
	}
}

// color none leaves pieces of both colors undimmed
func (p *PrivateState) dimAllButPieces(color string, board []*Piece) {
	for i, piece := range board {
		if piece != nil && (piece.Color == color || color == none) {
			p.Highlights[i] = highlightOff
		} else {
			p.Highlights[i] = highlightDim
		}
	}
}

func (p *PrivateState) highlightsOff() {
	for i := range p.Highlights {
		p.Highlights[i] = highlightOff
	}
}

func (p *PrivateState) highlightPosOn(pos Pos) {
	idx := pos.getBoardIdx()
	p.Highlights[idx] = highlightOn
}

func (p *PrivateState) highlightPosOff(pos Pos) {
	idx := pos.getBoardIdx()
	p.Highlights[idx] = highlightOff
}

// dims all squares but for specified indexes
// (doesn't assume indexes are in ascending order)
func (p *PrivateState) dimAllBut(indexes []int) {
	for i := range p.Highlights {
		p.Highlights[i] = highlightDim
		for _, idx := range indexes {
			if i == idx {
				p.Highlights[i] = highlightOff
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

// these things are always done together and in this order, hence this method
func (m *Match) UpdateStatusAndDamage() {
	m.CalculateSquareStatus(m.Board[:], m.SquareStatuses[:])
	m.CalculateDamage(m.Board[:], m.pieces[:], m.SquareStatuses[:])
}

func (m *Match) UpdateStatusAndDamageTemp() {
	m.CalculateSquareStatus(m.tempBoard[:], m.tempSquareStatuses[:])
	m.CalculateDamage(m.tempBoard[:], m.tempPieces[:], m.tempSquareStatuses[:])
}

// generate m.Combined from (m.Direct + square status effects from the pieces)
func (m *Match) CalculateSquareStatus(board []*Piece, squareStatuses []SquareStatus) {
	copy(squareStatuses, m.SquareStatusesDirect[:])
	// get status effects from pieces
	for i, piece := range board {
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
		m.WhitePrivate.highlightsOff()
		m.BlackPrivate.highlightsOff()
		pos := m.WhitePrivate.KingPos
		if pos != nil {
			m.setPiece(*pos, *m.WhitePublic.King)
			m.WhitePrivate.KingPos = nil
		}
		pos = m.BlackPrivate.KingPos
		if pos != nil {
			m.setPiece(*pos, *m.BlackPublic.King)
			m.BlackPrivate.KingPos = nil
		}
		m.UpdateStatusAndDamage()
		m.Phase = mainPhase
		m.PlayableCards()
		if m.BlackAI && m.Turn == black {
			playTurnAI(black, m)
		} else if m.WhiteAI && m.Turn == white {
			playTurnAI(white, m)
		}
		return true
	}
	return false
}

func (p *PrivateState) dimUnreclaimable(board []*Piece) {
	for i, piece := range board {
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
		m.InflictDamage()

		if !m.checkWinCondition() {
			m.Phase = reclaimPhase
			m.tickdownStatusEffects(false)
			board := m.Board[:]
			m.BlackPrivate.dimAllButPieces(black, board)
			m.WhitePrivate.dimAllButPieces(white, board)
			m.BlackPrivate.dimUnreclaimable(board)
			m.WhitePrivate.dimUnreclaimable(board)

			if m.BlackAI {
				m.BlackPrivate.ReclaimSelections = pickReclaimAI(black, board)
				m.BlackPublic.ReclaimSelectionMade = true
			}
			if m.WhiteAI {
				m.WhitePrivate.ReclaimSelections = pickReclaimAI(white, board)
				m.WhitePublic.ReclaimSelectionMade = true
			}
		}

		// with some blocking pieces possibly removed, status and lines of attack may change
		m.UpdateStatusAndDamage()
	} else {
		if m.Turn == black {
			m.Turn = white
		} else {
			m.Turn = black
		}
		m.PassPrior = pass

		m.WhitePrivate.SelectedCard = -1
		m.WhitePrivate.highlightsOff()

		m.BlackPrivate.SelectedCard = -1
		m.BlackPrivate.highlightsOff()

		if m.BlackAI && m.Turn == black {
			playTurnAI(black, m)
		} else if m.WhiteAI && m.Turn == white {
			playTurnAI(white, m)
		}
	}
}

func drawCards(existing []Card, public *PublicState, devMode bool) []Card {
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
			additional = randomCards(nCardsPerRound, public.ManaMax)
		} else if diff > 0 {
			additional = randomCards(diff, public.ManaMax)
		}
	}

	return append(append(stock, existing...), additional...)
}

func randomCards(n int, manaMax int) []Card {
	idx := manaMax
	if idx >= len(cardManaCount) {
		idx = len(cardManaCount) - 1
	}
	cardPoolSize := cardManaCount[idx]
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
	if m.Phase != gameoverPhase {
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
		case "reclaim_time_expired":
			switch m.Phase {
			case reclaimPhase:
				turnElapsed := time.Now().UnixNano() - m.LastMoveTime
				remainingTurnTime := m.TurnTimer - turnElapsed
				if remainingTurnTime > 0 {
					break // ignore if time hasn't actually expired
				}
				m.ReclaimPieces()
				m.EndRound()
				notifyOpponent = true
			}
		case "reclaim_done":
			switch m.Phase {
			case reclaimPhase:
				public.ReclaimSelectionMade = true
				// if other player already has a reclaim selection, then we move on to next round
				if (player == black && m.WhitePublic.ReclaimSelectionMade) ||
					(player == white && m.BlackPublic.ReclaimSelectionMade) {
					m.ReclaimPieces()
					m.EndRound()
					notifyOpponent = true
				} else {
					notifyOpponent = true
				}
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
						pos, _ := m.RandomFreeSquare(color)
						m.setPiece(pos, *public.King)
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
			newTurn, notifyOpponent = m.clickBoard(player, public, private, pos)
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
	}
	return
}
