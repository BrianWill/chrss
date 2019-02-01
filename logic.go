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

	m.WhitePublic.Color = white
	m.WhitePublic.Other = &m.BlackPublic
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

	m.BlackPublic.Color = black
	m.BlackPublic.Other = &m.WhitePublic
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

	m.Log = []string{"Round 1"}

	m.SpawnPawns(true)
	m.CalculateDamage()

	startingHand := []Card{
		Card{queen, queenMana},
		Card{castleCard, castleMana},
		Card{reclaimVassalCard, reclaimVassalMana},
		Card{swapFrontLinesCard, swapFrontLinesMana},
		Card{removePawnCard, removePawnMana},
		Card{forceCombatCard, forceCombatMana},
		Card{mirrorCard, mirrorMana},
		Card{healCard, healMana},
		Card{drainManaCard, drainManaMana},
		Card{togglePawnCard, togglePawnMana},
		Card{nukeCard, nukeMana},
		Card{shoveCard, shoveMana},
	}

	m.BlackPrivate = PrivateState{
		Cards:             drawCards(startingHand, &m.BlackPublic),
		SelectedCard:      -1,
		PlayerInstruction: defaultInstruction,
	}
	// white starts ready to play king
	m.WhitePrivate = PrivateState{
		Cards:             drawCards(startingHand, &m.WhitePublic),
		SelectedCard:      -1,
		PlayerInstruction: defaultInstruction,
	}

	m.BlackPrivate.Other = &m.WhitePrivate
	m.WhitePrivate.Other = &m.BlackPrivate

	m.BlackPrivate.dimAllButFree(black, m.Board[:])
	m.WhitePrivate.dimAllButFree(white, m.Board[:])

	m.playableCards()
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
func (m *Match) SpawnPawns(init bool) {
	public := &m.WhitePublic

	for i := 0; i < 2; i++ {
		n := 1
		if init {
			n = 4
		}
		var columns []int
		if !init && public.NumPawns == 0 {
			n = 2
		} else if public.NumPawns == 5 {
			return // keep max pawns at 5
		}
		front := 2
		mid := 1
		offset := 1
		if public == &m.BlackPublic {
			front = 3
			mid = 4
			offset = 3
		}
		for i := 0; i < nColumns; i++ {
			if m.getPiece(Pos{i, front}) == nil && m.getPiece(Pos{i, mid}) == nil {
				columns = append(columns, i)
			}
		}
		columns = randSelect(n, columns)
		n = len(columns)
		for _, v := range columns {
			m.setPiece(Pos{v, rand.Intn(2) + offset}, Piece{pawn, public.Color, pawnHP, pawnAttack, 0})
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

func (m *Match) playableCards() {
	// determine which cards are playable for each player given state of board
	public, private := m.states(white)
	for i := 0; i < 2; i++ {
		private.PlayableCards = make([]bool, len(private.Cards))
		for j, c := range private.Cards {
			private.PlayableCards[j] = false
			if public.ManaCurrent >= c.ManaCost {
				switch c.Name {
				case bishop, knight, rook, queen:
					// todo: check if a free square is available on player's side
					private.PlayableCards[j] = true
				case castleCard:
					if public.RookPlayed || public.Other.RookPlayed {
						private.PlayableCards[j] = true
					}
				case reclaimVassalCard:
					if public.RookPlayed || public.KnightPlayed || public.BishopPlayed {
						private.PlayableCards[j] = true
					}
				case forceCombatCard:
					private.PlayableCards[j] = true
				case mirrorCard:
					private.PlayableCards[j] = true
				case drainManaCard:
					if public.Other.ManaCurrent > 0 {
						private.PlayableCards[j] = true
					}
				case nukeCard:
					private.PlayableCards[j] = true
				case shoveCard:
					if len(m.shoveablePieces()) > 0 {
						private.PlayableCards[j] = true
					}
				case togglePawnCard:
					if len(m.toggleablePawns()) > 0 {
						private.PlayableCards[j] = true
					}
				case healCard:
					// todo: not playable if player has no pieces other than King on board
					private.PlayableCards[j] = true
				case removePawnCard:
					if public.NumPawns > 0 || public.Other.NumPawns > 0 {
						private.PlayableCards[j] = true
					}
				case swapFrontLinesCard:
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
	start := (nRows / 2) * nColumns // first look for black shoveable pieces
	for j := start; j < start+(nColumns*2); j++ {
		k := j + nColumns
		if m.Board[j] != nil && m.Board[k] == nil {
			indexes = append(indexes, j)
		}
	}
	// for white
	for j := 0; j < (nColumns * 2); j++ {
		k := j + nColumns
		if m.Board[j] == nil && m.Board[k] != nil {
			indexes = append(indexes, k)
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
				private.PlayerInstruction = defaultInstruction
				private.highlightsOff()
			} else {
				card := private.Cards[cardIdx]
				private.SelectedCard = cardIdx
				if public.ManaCurrent >= card.ManaCost {
					board := m.Board[:]
					switch card.Name {
					case castleCard:
						if m.WhitePublic.RookPlayed && m.BlackPublic.RookPlayed {
							private.dimAllButType(king, none, board)
						} else if m.WhitePublic.RookPlayed {
							private.dimAllButType(king, white, board)
						} else if m.BlackPublic.RookPlayed {
							private.dimAllButType(king, black, board)
						}
					case removePawnCard:
						if public.NumPawns > 0 || public.Other.NumPawns > 0 {
							private.dimAllButType(pawn, none, board)
						}
					case forceCombatCard:
						private.dimAllButType(king, player, board)
					case mirrorCard:
						private.dimAllButType(king, none, board)
					case drainManaCard:
						private.dimAllButType(king, otherColor(player), board)
					case togglePawnCard:
						private.dimAllBut(m.toggleablePawns())
					case nukeCard:
						private.dimAllButType(king, none, board)
					case shoveCard:
						private.dimAllBut(m.shoveablePieces())
					case healCard:
						private.dimAllButPieces(player, board)
						private.dimType(king, player, board)
					case swapFrontLinesCard:
						private.dimAllButType(king, none, board)
					case reclaimVassalCard:
						if public.RookPlayed || public.KnightPlayed || public.BishopPlayed {
							private.dimAllButTypes([]string{bishop, knight, rook}, player, board)
						}
					default:
						private.dimAllButFree(player, board)
					}
				}
			}
		}
	}
}

func (m *Match) clickBoard(player string, public *PublicState, private *PrivateState, p Pos) (newTurn bool, notifyOpponent bool) {
	switch m.Phase {
	case mainPhase:
		// ignore if not the player's turn and/or no card selected
		if player != m.Turn || private.SelectedCard == -1 {
			return
		}
		card := private.Cards[private.SelectedCard]
		switch card.Name {
		case castleCard:
			if !m.playCastle(p, player) {
				return
			}
		case reclaimVassalCard:
			if !m.playReclaimVassal(p, public, player) {
				return
			}
		case swapFrontLinesCard:
			if !m.playSwapFrontLines(p) {
				return
			}
		case removePawnCard:
			if !m.playRemovePawn(p) {
				return
			}
		case forceCombatCard:
			if !m.playForceCombat(p, player) {
				return
			}
		case mirrorCard:
			if !m.playMirror(p) {
				return
			}
		case drainManaCard:
			if !m.playDrainMana(p, public.Other) {
				return
			}
		case healCard:
			if !m.playHeal(p, player) {
				return
			}
		case togglePawnCard:
			if !m.playToggleablePawn(p) {
				return
			}
		case nukeCard:
			if !m.playNuke(p) {
				return
			}
		case shoveCard:
			if !m.playShove(p) {
				return
			}
		case bishop, knight, rook, queen:
			// ignore clicks on occupied spaces
			if m.getPieceSafe(p) != nil {
				return
			}
			// square must be on player's side of board
			if player == white && p.Y >= nColumns/2 {
				return
			}
			if player == black && p.Y < nColumns/2 {
				return
			}
			switch card.Name {
			case bishop:
				m.setPiece(p, Piece{bishop, player, public.BishopHP, public.BishopAttack, 0})
				public.BishopPlayed = true
			case knight:
				m.setPiece(p, Piece{knight, player, public.KnightHP, public.KnightAttack, 0})
				public.KnightPlayed = true
			case rook:
				m.setPiece(p, Piece{rook, player, public.RookHP, public.RookAttack, 0})
				public.RookPlayed = true
			case queen:
				m.setPiece(p, Piece{queen, player, queenHP, queenAttack, 0})
			}
		}
		public.ManaCurrent -= card.ManaCost
		m.Log = append(m.Log, player+" played "+card.Name)
		private.RemoveCard(private.SelectedCard)
		m.playableCards()
		m.EndTurn(false, player)
		newTurn = true
		notifyOpponent = true
	case reclaimPhase:
		piece := m.getPieceSafe(p)
		if piece != nil && piece.Color == player {
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
		if m.getPieceSafe(p) != nil {
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
		m.setPiece(p, Piece{king, player, public.KingHP, public.KingAttack, 0})
		newTurn = m.EndKingPlacement()
		notifyOpponent = true
	}
	return
}

// return true if play is valid
func (m *Match) playCastle(pos Pos, color string) bool {
	piece := m.getPieceSafe(pos)
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
	swap := *rookPiece
	*rookPiece = *piece
	*piece = swap
	return true
}

// return true if play is valid
func (m *Match) playReclaimVassal(pos Pos, public *PublicState, color string) bool {
	piece := m.getPieceSafe(pos)
	if piece == nil || piece.Color != color {
		return false
	}
	switch piece.Name {
	case bishop:
		public.BishopPlayed = false
	case knight:
		public.KnightPlayed = false
	case rook:
		public.RookPlayed = false
	default:
		return false
	}
	m.removePieceAt(pos)
	return true
}

// return true if play is valid
func (m *Match) playSwapFrontLines(pos Pos) bool {
	piece := m.getPieceSafe(pos)
	if piece == nil || piece.Name != king {
		return false
	}
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
	return true
}

// return true if play is valid
func (m *Match) playRemovePawn(pos Pos) bool {
	piece := m.getPieceSafe(pos)
	if piece == nil || piece.Name != pawn {
		return false
	}
	public, _ := m.states(piece.Color)
	m.removePieceAt(pos)
	public.NumPawns--
	return true
}

// return true if play is valid
func (m *Match) playForceCombat(pos Pos, player string) bool {
	piece := m.getPieceSafe(pos)
	if piece == nil || piece.Name != king || piece.Color != player {
		return false
	}
	m.PassPrior = true
	m.EndTurn(true, player)
	return true
}

// return true if play is valid
func (m *Match) playDrainMana(pos Pos, otherPublic *PublicState) bool {
	piece := m.getPieceSafe(pos)
	if piece == nil || piece.Name != king || piece.Color != otherPublic.Color {
		return false
	}
	otherPublic.ManaCurrent -= drainManaAmount
	if otherPublic.ManaCurrent < 0 {
		otherPublic.ManaCurrent = 0
	}
	return true
}

// return true if play is valid
// (assumes board has even number of rows)
func (m *Match) playMirror(pos Pos) bool {
	piece := m.getPieceSafe(pos)
	if piece == nil || piece.Name != king {
		return false
	}
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
	return true
}

// return true if play is valid
func (m *Match) playHeal(pos Pos, player string) bool {
	piece := m.getPieceSafe(pos)
	if piece == nil || piece.Name == king {
		return false
	}
	piece.HP += healCardAmount
	public, _ := m.states(player)
	switch piece.Name {
	case rook:
		public.RookHP += healCardAmount
	case knight:
		public.KnightHP += healCardAmount
	case bishop:
		public.BishopHP += healCardAmount
	}
	return true
}

// return true if play is valid
func (m *Match) playNuke(pos Pos) bool {
	piece := m.getPieceSafe(pos)
	if piece == nil || piece.Name != king {
		return false
	}
	// inflict lesser damage on all within 2 squares
	minX, maxX := pos.X-2, pos.X+2
	minY, maxY := pos.Y-2, pos.Y+2
	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			target := Pos{x, y}
			if target == pos {
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
			if target == pos {
				continue
			}
			m.inflictDamage(target.getBoardIdx(), nukeDamageFull-nukeDamageLesser)
		}
	}
	return true
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
	var public *PublicState
	switch p.Name {
	case king:
		public = m.getPublic(p.Color)
		public.KingHP -= dmg
	case rook:
		public = m.getPublic(p.Color)
		public.RookHP -= dmg
	case bishop:
		public = m.getPublic(p.Color)
		public.BishopHP -= dmg
	case knight:
		public = m.getPublic(p.Color)
		public.KnightHP -= dmg
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
		}
		m.checkWinCondition()
	}
}

// sets match state to gameover if winner or draw
func (m *Match) checkWinCondition() bool {
	b, w := m.BlackPublic, m.WhitePublic
	whiteLose := w.KingHP <= 0 || (w.KnightHP <= 0 && w.BishopHP <= 0 && w.RookHP <= 0)
	blackLose := b.KingHP <= 0 || (b.KnightHP <= 0 && b.BishopHP <= 0 && b.RookHP <= 0)
	if whiteLose && blackLose {
		m.Winner = draw
		m.Phase = gameoverPhase
		return true
	} else if whiteLose {
		m.Winner = white
		m.Phase = gameoverPhase
		return true
	} else if blackLose {
		m.Winner = black
		m.Phase = gameoverPhase
		return true
	}
	return false
}

func (m *Match) playToggleablePawn(pos Pos) bool {
	piece := m.getPieceSafe(pos)
	if piece == nil || piece.Name != pawn {
		return false
	}
	idx := pos.getBoardIdx()
	for _, val := range m.toggleablePawns() {
		if idx == val {
			const whiteMid = nRows/2 - 2
			const whiteFront = whiteMid + 1
			const blackFront = whiteMid + 2
			const blackMid = whiteMid + 3
			newPos := pos
			switch pos.Y {
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
			return true
		}
	}
	return false
}

func (m *Match) playShove(pos Pos) bool {
	piece := m.getPieceSafe(pos)
	if piece == nil {
		return false
	}
	idx := pos.getBoardIdx()
	for _, val := range m.shoveablePieces() {
		if idx == val {
			var newIdx int
			if idx < len(m.Board)/2 {
				newIdx = idx - nColumns
			} else {
				newIdx = idx + nColumns
			}
			m.swapBoardIndex(idx, newIdx)
			return true
		}
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

	m.SpawnPawns(false)
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
	m.playableCards()
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

// dims all squares but for specified type and color (or both colors if 'none')
// returns count of pieces matching type and color
func (p *PrivateState) dimAllButType(pieceType string, color string, board []*Piece) int {
	n := 0
	for i, piece := range board {
		if piece != nil && piece.Name == pieceType && (piece.Color == color || color == none) {
			p.Highlights[i] = highlightOff
			n++
		} else {
			p.Highlights[i] = highlightDim
		}
	}
	return n
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

// dims all pieces of specified type and color (or both colors if 'none')
// returns count of pieces matching type and color
func (p *PrivateState) dimType(pieceType string, color string, board []*Piece) int {
	n := 0
	for i, piece := range board {
		if piece != nil && piece.Name == pieceType && (piece.Color == color || color == none) {
			p.Highlights[i] = highlightDim
			n++
		}
	}
	return n
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

// highlights units of a specified type and color (or both colors if 'none')
func (p *PrivateState) dimAllButTypes(pieceTypes []string, color string, board []*Piece) {
	for i, piece := range board {
		if piece != nil && (piece.Color == color || color == none) && stringInSlice(piece.Name, pieceTypes) {
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
		m.playableCards()
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

		if !m.checkWinCondition() {
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
