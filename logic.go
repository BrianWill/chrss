package main

import (
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
	m.Round = 0 // will get incremented to 1 once both players ready up
	m.Phase = readyUpPhase

	m.WhitePublic.ManaCurrent = 3
	m.WhitePublic.ManaMax = 3
	m.WhitePublic.KingHP = kingHP
	m.WhitePublic.KingAttack = kingAttack
	m.WhitePublic.BishopHP = bishopHP
	m.WhitePublic.BishopAttack = bishopAttack
	m.WhitePublic.KnightHP = knightHP
	m.WhitePublic.KnightAttack = knightAttack
	m.WhitePublic.RookHP = rookHP
	m.WhitePublic.RookAttack = rookAttack

	m.BlackPublic.ManaCurrent = 3
	m.BlackPublic.ManaMax = 3
	m.BlackPublic.KingHP = kingHP
	m.BlackPublic.KingAttack = kingAttack
	m.BlackPublic.BishopHP = bishopHP
	m.BlackPublic.BishopAttack = bishopAttack
	m.BlackPublic.KnightHP = knightHP
	m.BlackPublic.KnightAttack = knightAttack
	m.BlackPublic.RookHP = rookHP
	m.BlackPublic.RookAttack = rookAttack

	m.SpawnPawns(white, true)
	m.SpawnPawns(black, true)
	m.CalculateDamage()

	startingHand := []Card{
		Card{queen, queenMana},
		Card{castleCard, castleMana},
	}

	m.BlackPrivate = PrivateState{
		Cards:             drawCards(startingHand, &m.BlackPublic),
		SelectedCard:      -1,
		SelectedPos:       Pos{-1, -1},
		PlayerInstruction: defaultInstruction,
	}
	// white starts ready to play king
	m.WhitePrivate = PrivateState{
		Cards:             drawCards(startingHand, &m.WhitePublic),
		SelectedCard:      -1,
		SelectedPos:       Pos{-1, -1},
		PlayerInstruction: defaultInstruction,
	}

	m.BlackPrivate.dimAllButFree(black, m.Board[:])
	m.WhitePrivate.dimAllButFree(white, m.Board[:])
}

// returns nil for empty square
func (m *Match) getPiece(p Pos) *Piece {
	return m.Board[nColumns*p.Y+p.X]
}

// returns -1 if invalid
func (p *Pos) getBoardIdx() int {
	idx := nColumns*p.Y + p.X
	if idx < 0 || idx >= (nColumns*nRows) {
		return -1
	}
	return idx
}

// does not panic
func (m *Match) getPieceSafe(p Pos) *Piece {
	if p.X < 0 || p.X >= nColumns || p.Y < 0 || p.Y >= nRows {
		return nil
	}
	return m.Board[nColumns*p.Y+p.X]
}

// panics if out of bounds
func (m *Match) setPiece(p Pos, piece Piece) {
	idx := nColumns*p.Y + p.X
	m.pieces[idx] = piece
	m.Board[idx] = &m.pieces[idx]
}

// panics if out of bounds
func (m *Match) removePieceAt(p Pos) {
	idx := nColumns*p.Y + p.X
	m.Board[idx] = nil
	m.pieces[idx] = Piece{}
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
			var public *PublicState
			if p.Color == black {
				public = &m.BlackPublic
			} else {
				public = &m.WhitePublic
			}
			switch p.Name {
			case king:
				public.KingHP = p.HP
			case bishop:
				public.BishopHP = p.HP
			case knight:
				public.KnightHP = p.HP
			case rook:
				public.RookHP = p.HP
			}
			if p.HP <= 0 {
				if p.Name == pawn {
					public.NumPawns--
				}
				m.pieces[i] = Piece{}
				m.Board[i] = nil
			}
		}
	}
}

func (m *Match) CalculateDamage() {
	// todo
	rookAttack := func(p Pos, color string, attack int) {
		x := p.X + 1
		y := p.Y
		for x < nColumns {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x++
		}

		x = p.X - 1
		y = p.Y
		for x >= 0 {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x--
		}

		x = p.X
		y = p.Y + 1
		for y < nRows {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			y++
		}

		x = p.X
		y = p.Y - 1
		for y >= 0 {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			y--
		}
	}

	bishopAttack := func(p Pos, color string, attack int) {
		x := p.X + 1
		y := p.Y + 1
		for x < nColumns && y < nRows {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x++
			y++
		}

		x = p.X - 1
		y = p.Y + 1
		for x >= 0 && y < nRows {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x--
			y++
		}

		x = p.X + 1
		y = p.Y - 1
		for x < nColumns && y >= 0 {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x++
			y--
		}

		x = p.X - 1
		y = p.Y - 1
		for x >= 0 && y >= 0 {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x--
			y--
		}
	}

	queenAttack := func(p Pos, color string, attack int) {
		x := p.X + 1
		y := p.Y + 1
		for x < nColumns && y < nRows {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x++
			y++
		}

		x = p.X - 1
		y = p.Y + 1
		for x >= 0 && y < nRows {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x--
			y++
		}

		x = p.X + 1
		y = p.Y - 1
		for x < nColumns && y >= 0 {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x++
			y--
		}

		x = p.X - 1
		y = p.Y - 1
		for x >= 0 && y >= 0 {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x--
			y--
		}

		// cardinal directions
		x = p.X + 1
		y = p.Y
		for x < nColumns {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x++
		}

		x = p.X - 1
		y = p.Y
		for x >= 0 {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			x--
		}

		x = p.X
		y = p.Y + 1
		for y < nRows {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			y++
		}

		x = p.X
		y = p.Y - 1
		for y >= 0 {
			hit := m.getPiece(Pos{x, y})
			if hit != nil {
				if hit.Color != color {
					hit.Damage += attack
				}
				break
			}
			y--
		}
	}

	kingAttack := func(p Pos, color string, attack int) {
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
			hit := m.getPieceSafe(other)
			if hit != nil && hit.Color != color {
				hit.Damage += attack
			}
		}
	}

	knightAttack := func(p Pos, color string, attack int) {
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
			hit := m.getPieceSafe(other)
			if hit != nil && hit.Color != color {
				hit.Damage += attack
			}
		}
	}

	pawnAttack := func(p Pos, color string, attack int) {
		yOffset := 1
		if color == black {
			yOffset = -1
		}
		ps := []Pos{
			Pos{p.X + 1, p.Y + yOffset},
			Pos{p.X - 1, p.Y + yOffset},
		}
		for _, other := range ps {
			hit := m.getPieceSafe(other)
			if hit != nil && hit.Color != color {
				hit.Damage += attack
			}
		}
	}

	// reset all to 0
	for i := range m.pieces {
		m.pieces[i].Damage = 0
	}

	attackMap := map[string]func(Pos, string, int){
		king:   kingAttack,
		bishop: bishopAttack,
		knight: knightAttack,
		rook:   rookAttack,
		pawn:   pawnAttack,
		queen:  queenAttack,
	}

	// visit each piece, adding the damage it inflicts on other pieces
	for i, p := range m.Board {
		if p != nil {
			attackMap[p.Name](positions[i], p.Color, p.Attack)
		}
	}
}

// spawn n random pawns in free columns
func (m *Match) SpawnPawns(player string, init bool) {
	whiteNumPawns := m.WhitePublic.NumPawns
	blackNumPawns := m.BlackPublic.NumPawns
	n := 1
	if init {
		n = 4
	}
	var columns []int
	if player == white {
		if !init && whiteNumPawns == 0 {
			n = 2
		} else if whiteNumPawns == 5 {
			return // keep max pawns at 5
		}
		for i := 0; i < nColumns; i++ {
			if m.getPiece(Pos{i, 1}) == nil && m.getPiece(Pos{i, 2}) == nil {
				columns = append(columns, i)
			}
		}
		columns = randSelect(n, columns)
		for _, v := range columns {
			m.setPiece(Pos{v, rand.Intn(2) + 1}, Piece{pawn, white, pawnHP, pawnAttack, 0})
		}
		m.WhitePublic.NumPawns = whiteNumPawns + n
	} else {
		if !init && blackNumPawns == 0 {
			n = 2
		} else if blackNumPawns == 5 {
			return // keep max pawns at 5
		}
		for i := 0; i < nColumns; i++ {
			if m.getPiece(Pos{i, 3}) == nil && m.getPiece(Pos{i, 4}) == nil {
				columns = append(columns, i)
			}
		}
		columns = randSelect(n, columns)
		for _, v := range columns {
			m.setPiece(Pos{v, rand.Intn(2) + 3}, Piece{pawn, black, pawnHP, pawnAttack, 0})
		}
		m.BlackPublic.NumPawns = blackNumPawns + n
	}
	switch n {
	case 0:
		m.Log = append(m.Log, player+" gained no pawns")
	case 1:
		m.Log = append(m.Log, player+" gained 1 pawn")
	default:
		m.Log = append(m.Log, player+" gained "+strconv.Itoa(n)+" pawns")
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

func (m *Match) allStates(color string) (*PublicState, *PrivateState, *PublicState, *PrivateState) {
	if color == black {
		return &m.BlackPublic, &m.BlackPrivate, &m.WhitePublic, &m.WhitePrivate
	} else {
		return &m.WhitePublic, &m.WhitePrivate, &m.BlackPublic, &m.BlackPrivate
	}
}

func (m *Match) playCastle(pos Pos, color string) bool {
	piece := m.getPieceSafe(pos)
	if piece == nil {
		return false
	}
	if piece.Name != king {
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
	swap := *rookPiece
	*rookPiece = *piece
	*piece = swap
	return true
}

func (m *Match) ReclaimPieces() {
	for _, color := range []string{white, black} {
		public, private := m.states(color)
		if public.ReclaimSelectionMade {
			for i, pos := range private.ReclaimSelections {
				if i >= 2 {
					break // in case more than two selections were sent, we ignore all but first two
				}
				p := m.getPieceSafe(pos)
				if p != nil { // selection might be off the board or a square without a piece (todo: send error to client)
					// if reclaiming a king or vassal, add card back to hand
					// if reclaiming a rook, heal the rook
					switch p.Name {
					case king:
						m.removePieceAt(pos)
						public.KingPlayed = false
					case bishop:
						m.removePieceAt(pos)
						public.BishopPlayed = false
					case knight:
						m.removePieceAt(pos)
						public.KnightPlayed = false
					case rook:
						m.removePieceAt(pos)
						public.RookPlayed = false
						public.RookHP += reclaimHealRook
						if public.RookHP > rookHP {
							public.RookHP = rookHP
						}
					case queen:
						m.removePieceAt(pos)
					case pawn:
						m.removePieceAt(pos)
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

	m.WhitePrivate.Cards = drawCards(m.WhitePrivate.Cards, &m.WhitePublic)
	m.BlackPrivate.Cards = drawCards(m.BlackPrivate.Cards, &m.BlackPublic)
	m.WhitePrivate.SelectedCard = -1
	m.BlackPrivate.SelectedCard = -1

	m.SpawnPawns(black, false)
	m.SpawnPawns(white, false)
	m.CalculateDamage()

	if m.WhitePublic.KingPlayed && m.BlackPublic.KingPlayed {
		m.Phase = mainPhase
		m.WhitePrivate.highlightsOff()
		m.BlackPrivate.highlightsOff()
	} else {
		m.Phase = kingPlacementPhase
		if m.WhitePublic.KingPlayed {
			m.WhitePrivate.highlightsOff()
		} else {
			m.WhitePrivate.dimAllButFree(white, m.Board[:])
		}
		if m.BlackPublic.KingPlayed {
			m.BlackPrivate.highlightsOff()
		} else {
			m.BlackPrivate.dimAllButFree(black, m.Board[:])
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

func (p *PrivateState) highlightSelections(selections []Pos) {
	for _, pos := range selections {
		idx := pos.getBoardIdx()
		p.Highlights[idx] = highlightOn
	}
}

func (p *PrivateState) highlightsOff() {
	for i := range p.Highlights {
		p.Highlights[i] = highlightOff
	}
}

func (p *PrivateState) dimAll() {
	for i := range p.Highlights {
		p.Highlights[i] = highlightDim
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

// highlights units of a specified type and color (or both colors if 'none')
func (p *PrivateState) dimAllButType(pieceType string, color string, board []*Piece) {
	for i, piece := range board {
		if piece != nil && piece.Name == pieceType && (piece.Color == color || color == none) {
			p.Highlights[i] = highlightOff
		} else {
			p.Highlights[i] = highlightDim
		}
	}
}

func (m *Match) EndKingPlacement() bool {
	if m.WhitePublic.KingPlayed && m.BlackPublic.KingPlayed {
		m.CalculateDamage()
		m.LastMoveTime = time.Now().UnixNano()
		m.WhitePrivate.highlightsOff()
		m.BlackPrivate.highlightsOff()
		m.Phase = mainPhase
		return true
	}
	return false
}

// pass = if turn is ending by passing; player = color whose turn is ending
func (m *Match) EndTurn(pass bool, player string) {
	m.LastMoveTime = time.Now().UnixNano()
	m.CalculateDamage()

	if pass && m.PassPrior { // if players both pass in succession, do combat
		m.InflictDamage()

		blackVassalsDead := m.BlackPublic.BishopHP <= 0 && m.BlackPublic.KnightHP <= 0 && m.BlackPublic.RookHP <= 0
		whiteVassalsDead := m.WhitePublic.BishopHP <= 0 && m.WhitePublic.KnightHP <= 0 && m.WhitePublic.RookHP <= 0
		blackKingDead := m.BlackPublic.KingHP <= 0
		whiteKingDead := m.WhitePublic.KingHP <= 0

		if (blackKingDead || blackVassalsDead) && (whiteKingDead || whiteVassalsDead) {
			m.Winner = draw
			m.Phase = gameoverPhase
		} else if blackKingDead || blackVassalsDead {
			m.Winner = white
			m.Phase = gameoverPhase
		} else if whiteKingDead || whiteVassalsDead {
			m.Winner = black
			m.Phase = gameoverPhase
		} else {
			m.Phase = reclaimPhase
			m.BlackPrivate.dimAllButPieces(black, m.Board[:])
			m.WhitePrivate.dimAllButPieces(white, m.Board[:])
		}
	} else {
		if m.Turn == black {
			m.Turn = white
		} else {
			m.Turn = black
		}
		m.PassPrior = pass

		m.WhitePrivate.PlayerInstruction = defaultInstruction
		m.WhitePrivate.SelectedCard = -1
		m.WhitePrivate.highlightsOff()

		m.BlackPrivate.PlayerInstruction = defaultInstruction
		m.BlackPrivate.SelectedCard = -1
		m.BlackPrivate.highlightsOff()
	}
}

func drawCards(existing []Card, public *PublicState) []Card {
	// remove vassal cards from existing
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
	if public.BishopHP > 0 && !public.BishopPlayed {
		stock = append(stock, Card{bishop, 0})
	}
	if public.KnightHP > 0 && !public.KnightPlayed {
		stock = append(stock, Card{knight, 0})
	}
	if public.RookHP > 0 && !public.RookPlayed {
		stock = append(stock, Card{rook, 0})
	}

	return append(stock, existing...)
}

func drawCommunalCards() []Card {
	return []Card{}
}

func (m *Match) IsOpen() bool {
	return (m.WhiteConn == nil || m.BlackConn == nil) && m.Winner == none
}

func (m *Match) IsBlackOpen() bool {
	return m.BlackConn == nil && m.Winner == none
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

// panics if outo f bounds
func (p *PrivateState) RemoveCard(idx int) {
	p.Cards = append(p.Cards[:idx], p.Cards[idx+1:]...)
}
