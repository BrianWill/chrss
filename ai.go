package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

func kingPlacementAI(color string, board []*Piece) Pos {
	free := freeSpaces(color, board)
	if len(free) == 0 {
		panic("Somehow no space for King! How did this happen?")
	}
	const backRowWeight = 2
	const middleRowWeight = 1
	highestScore := 0
	highestScoreIdxs := []int{free[0]}
	if color == black {
		const forwardOffset = -nColumns
		const backRowFirstIdx = nColumns*nRows - nColumns
		const middleRowFirstIdx = backRowFirstIdx - nColumns
		for _, idx := range free {
			score := 0
			// todo: calc damage on king in this position and subtract points accordingly
			if idx >= backRowFirstIdx {
				score += backRowWeight
			} else if idx >= middleRowFirstIdx {
				score += middleRowWeight
			} else {

			}
			if score > highestScore {
				highestScore = score
				highestScoreIdxs = []int{idx}
			} else if score == highestScore {
				highestScoreIdxs = append(highestScoreIdxs, idx)
			}
		}
	} else {
		const forwardOffset = nColumns
		const backRowLastIdx = nColumns - 1
		const middleRowLastIdx = nColumns + nColumns - 1
		for _, idx := range free {
			score := 0
			// todo: calc damage on king in this position and subtract points accordingly
			if idx <= backRowLastIdx {
				score += backRowWeight
			} else if idx >= middleRowLastIdx {
				score += middleRowWeight
			} else {

			}
			if score > highestScore {
				highestScore = score
				highestScoreIdxs = []int{idx}
			} else if score == highestScore {
				highestScoreIdxs = append(highestScoreIdxs, idx)
			}
		}
	}
	randWinner := highestScoreIdxs[rand.Intn(len(highestScoreIdxs))]
	return positions[randWinner]
}

func freeSpaces(color string, board []*Piece) []int {
	free := []int{}
	const half = nColumns * nRows / 2
	start := 0
	end := half
	if color == black {
		start = half
		end = len(board)
	}
	for i := start; i < end; i++ {
		if board[i] == nil {
			free = append(free, i)
		}
	}
	return free
}

// return indexes of up to two pieces to reclaim
func pickReclaimAI(color string, board []*Piece) []Pos {
	type PieceScore struct {
		Idx   int
		Score int
	}
	pickTopTwoScorers := func(scores []PieceScore) []PieceScore {
		// pick top two scorers
		// if tie for first place, pick randomly among the top scorers
		// (or if highest score is not tie, score greater than others, randomly choose second selection among those tied for second place)
		if len(scores) > 2 {
			sort.Slice(scores, func(i, j int) bool {
				return scores[i].Score > scores[j].Score
			})
			if scores[0].Score > scores[1].Score {
				// pick one randomly among second place scorers
				lastSecondPlaceIdx := 1
				for i := 2; i < len(scores); i++ {
					if scores[1].Score == scores[i].Score {
						lastSecondPlaceIdx++
					}
				}
				// add one for offset
				scores[1] = scores[rand.Intn(lastSecondPlaceIdx)+1]
				// first place now in slot 0 and random second place is now in slot 1
			} else {
				// pick two randomly among first place scorers
				lastFirstPlaceIdx := 1
				for i := 2; i < len(scores); i++ {
					if scores[1].Score == scores[i].Score {
						lastFirstPlaceIdx++
					}
				}
				rand.Shuffle(lastFirstPlaceIdx+1, func(i, j int) {
					scores[i], scores[j] = scores[j], scores[i]
				})
				// random first place scorers now in slots 0 and 1
			}
			scores = scores[:2]
		}
		return scores
	}
	const kingRemovalBonus = 2
	scores := []PieceScore{}
	for i, p := range board {
		if p != nil && p.Color == color && !p.isUnreclaimable() {
			score := 0
			switch p.Name {
			case king:
				score += kingRemovalBonus
			case bishop:
			case knight:
			case rook:
			case pawn:
				if p.HP < pawnHP {
					score += 2
				}
			default:
			}
			scores = append(scores, PieceScore{i, score})
		}
	}
	scores = pickTopTwoScorers(scores)
	selections := make([]Pos, len(scores))
	for i, val := range scores {
		selections[i] = positions[val.Idx]
	}
	return selections
}

func playTurnAI(color string, m *Match) {
	defer timeTrack(time.Now(), "playTurnAI")

	public, private := m.states(color)
	boardScore := scoreBoard(color, m.Board[:])
	fmt.Printf("current board state score: %v\n", boardScore)

	// positive score = better than passing
	// negative score = worse than passing
	scores := make([]int, len(private.Cards))
	pos := make([]Pos, len(private.Cards)) // for the scored card, the chosen Pos to 'click'
	fmt.Println("AI thinking...")
	for i, c := range private.Cards {
		if private.PlayableCards[i] {
			scores[i], pos[i] = scoreCardAI(c.Name, color, boardScore, m)
			fmt.Printf("card %s: score %v, pos %v\n", c.Name, scores[i], pos[i])
		}
	}

	// determine highest score, pick random from tie for first
	highestIdxs := []int{}
	highestScore := -10000 // arbitrarily low value
	for i, score := range scores {
		if score > highestScore {
			highestScore = score
			highestIdxs = []int{i}
		} else if score == highestScore {
			highestIdxs = append(highestIdxs, i)
		}
	}
	if len(highestIdxs) == 0 || highestScore <= 0 {
		m.EndTurn(true, color) // pass
		m.Log = append(m.Log, color+" passed")
		return
	}
	selectedIdx := highestIdxs[rand.Intn(len(highestIdxs))]
	private.SelectedCard = selectedIdx
	m.clickBoard(color, public, private, pos[selectedIdx])
}

// return score for entire board state from perspective of 'color' player
// a board score is only relative to other board scores
// (no special significance for positive or negative scores; simply, higher is better)
func scoreBoard(color string, board []*Piece) int {
	const allyVassalKilledBonus = -60
	const allySecondVassalKilledBonus = -1000
	const allyKingKilledBonus = -1000
	const allyDmgBonus = -3 // per point of dmg incurred
	const allyVassalDmgBonus = -6
	const allyKingDmgBonus = -12

	const enemyVassalKilledBonus = 60
	const enemySecondVassalKilledBonus = 500
	const enemyKingKilledBonus = 500
	const enemyDmgBonus = 3
	const enemyVassalDmgBonus = 6
	const enemyKingDmgBonus = 12

	// give points for presence of pieces on board
	const allyPawnBonus = 2
	const allyRookBonus = 4
	const allyKnightBonus = 4
	const allyBishopBonus = 4
	const allyOtherBonus = 3

	const enemyPawnBonus = 2
	const enemyRookBonus = 4
	const enemyKnightBonus = 4
	const enemyBishopBonus = 4
	const enemyOtherBonus = 3

	var score float64

	for _, p := range board {
		if p != nil {
			if p.Color == color {
				// ally
				switch p.Name {
				case pawn:
					score += allyPawnBonus
					score += float64(p.Damage) * allyDmgBonus
				case king:
					score += float64(p.Damage) * allyKingDmgBonus
				case bishop:
					score += allyBishopBonus
					score += float64(p.Damage) * allyVassalDmgBonus
				case knight:
					score += allyKnightBonus
					score += float64(p.Damage) * allyVassalDmgBonus
				case rook:
					score += allyRookBonus
					score += float64(p.Damage) * allyVassalDmgBonus
				default:
					score += allyOtherBonus
					score += float64(p.Damage) * allyDmgBonus
				}
			} else {
				// enemy
				switch p.Name {
				case pawn:
					score += enemyPawnBonus
					score += float64(p.Damage) * enemyDmgBonus
				case king:
					score += float64(p.Damage) * enemyKingDmgBonus
				case bishop:
					score += enemyBishopBonus
					score += float64(p.Damage) * enemyVassalDmgBonus
				case knight:
					score += enemyKnightBonus
					score += float64(p.Damage) * enemyVassalDmgBonus
				case rook:
					score += enemyRookBonus
					score += float64(p.Damage) * enemyVassalDmgBonus
				default:
					score += enemyOtherBonus
					score += float64(p.Damage) * enemyDmgBonus
				}
			}
		}
	}
	// todo: subtract points for exposed positions, esp. for exposed king / vassals
	return int(score)
}

func (m *Match) saveBoardToTemp() {
	m.tempPieces = m.pieces
	for i, p := range m.Board {
		if p == nil {
			m.tempBoard[i] = nil
		} else {
			piece := &m.tempPieces[i]
			m.tempBoard[i] = piece

			// deep copy of piece status
			if piece.Status != nil {
				temp := *piece.Status
				piece.Status = &temp
				if piece.Status.Negative != nil {
					temp := *piece.Status.Negative
					piece.Status.Negative = &temp
				}
				if piece.Status.Positive != nil {
					temp := *piece.Status.Positive
					piece.Status.Positive = &temp
				}
			}
		}
	}
}

// assumes card/pos combo is a valid play
func scoreCardAIPos(cardName string, pos Pos, color string, boardScore int, m *Match) int {
	score := 0
	switch cardName {
	// cards that affect the board
	case castleCard, reclaimVassalCard, swapFrontLinesCard, removePawnCard, dispellCard, dodgeCard, mirrorCard,
		healCard, poisonCard, togglePawnCard, nukeCard, vulnerabilityCard, amplifyCard, transparencyCard,
		stunVassalCard, enrageCard, armorCard, shoveCard, advanceCard, summonPawnCard,
		bishop, knight, rook, queen, jester:
		m.saveBoardToTemp()
		public, _ := m.states(color)
		playCardTemp(m, cardName, color, public, pos)
		m.UpdateStatusAndDamageTemp()
		score = scoreBoard(color, m.tempBoard[:]) - boardScore
	// cards that don't affect the board
	case forceCombatCard:
		// todo: high score if you have combat advantage (or no other good cards
		// to play and opponent has high mana / num cards)
		score = 1
	case drainManaCard:
		// todo: high score if enemy has low mana
		score = 1
	case restoreManaCard:
		// todo: high score if player has low mana && player has high cost cards in hand
		score = 1
	case resurrectVassalCard:
		// todo: high score in all scenarios
		score = 100
	}
	return score
}

func playCardTemp(m *Match, card string, player string, public *PublicState, p Pos) {
	piece := m.getTempPiece(p)
	switch card {
	case castleCard:
		// find rook of same color as clicked king
		var rookPiece *Piece
		for _, p := range m.tempBoard {
			if p != nil && p.Name == rook && p.Color == piece.Color {
				rookPiece = p
				break
			}
		}
		swap := *rookPiece
		*rookPiece = *piece
		*piece = swap
	case reclaimVassalCard:
		m.removeTempPieceAt(p)
	case swapFrontLinesCard:
		frontIdx := (nRows/2 - 1) * nColumns
		midIdx := (nRows/2 - 2) * nColumns
		if piece.Color == black {
			frontIdx = (nRows / 2) * nColumns
			midIdx = (nRows/2 + 1) * nColumns
		}
		for i := 0; i < nColumns; i++ {
			m.swapTempBoardIndex(frontIdx, midIdx)
			frontIdx++
			midIdx++
		}
	case removePawnCard:
		m.removeTempPieceAt(p)
	case forceCombatCard:
		//
	case dispellCard:
		piece.Status = nil
	case dodgeCard:
		idx := p.getBoardIdx()
		for _, val := range m.dodgeablePieces(player) {
			if val == idx {
				free := m.freeAdjacentSpaces(idx)
				newIdx := free[rand.Intn(len(free))]
				m.swapTempBoardIndex(idx, newIdx)
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
				m.swapTempBoardIndex(idx, other)
				idx++
				other--
			}
			row++
		}
	case drainManaCard:
		//
	case healCard:
		piece.HP += healCardAmount
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
				m.swapTempBoardIndex(idx, newPos.getBoardIdx())
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
				m.inflictTempDamage(target.getBoardIdx(), nukeDamageLesser)
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
				m.inflictTempDamage(target.getBoardIdx(), nukeDamageFull-nukeDamageLesser)
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
				m.swapTempBoardIndex(idx, newIdx)
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
				m.swapTempBoardIndex(idx, newIdx)
				break
			}
		}
	case restoreManaCard:
		//
	case summonPawnCard:
		m.SpawnSinglePawnTemp(player, public)
	case resurrectVassalCard:
		//
	case bishop, knight, rook, queen, jester:
		switch card {
		case bishop:
			m.setTempPiece(p, *public.Bishop)
		case knight:
			m.setTempPiece(p, *public.Knight)
		case rook:
			m.setTempPiece(p, *public.Rook)
		case queen:
			m.setTempPiece(p, Piece{queen, player, queenHP, queenAttack, 0, nil})
		case jester:
			m.setTempPiece(p, Piece{jester, player, jesterHP, jesterAttack, 0, nil})
		}
	}
}

// assumes both kings are on the board
func kingIdxs(color string, board []*Piece) []int {
	idxs := []int{}
	for i, p := range board {
		if p != nil && p.Name == king && (p.Color == color || color == none) {
			idxs = append(idxs, i)
		}
	}
	return idxs
}

func pawnIdxs(color string, board []*Piece) []int {
	idxs := []int{}
	for i, p := range board {
		if p != nil && p.Name == pawn {
			if p.Color == color || color == none {
				idxs = append(idxs, i)
			}
		}
	}
	return idxs
}

func pieceIdxs(color string, board []*Piece) []int {
	idxs := []int{}
	for i, p := range board {
		if p != nil {
			if p.Color == color || color == none {
				idxs = append(idxs, i)
			}
		}
	}
	return idxs
}

func vassalIdxs(color string, board []*Piece) []int {
	idxs := []int{}
	for i, p := range board {
		if p != nil && (p.Name == knight || p.Name == bishop || p.Name == rook) {
			if p.Color == color || color == none {
				idxs = append(idxs, i)
			}
		}
	}
	return idxs
}

func freeIdxs(color string, board []*Piece) []int {
	start := 0
	end := nColumns * nRows
	switch color {
	case white:
		end = nColumns * nRows / 2
	case black:
		start = nColumns * nRows / 2
	}
	idxs := []int{}
	for i := start; i < end; i++ {
		if board[i] == nil {
			idxs = append(idxs, i)
		}
	}
	return idxs
}

func idxsToPos(idxs []int) []Pos {
	pos := make([]Pos, len(idxs))
	for i, idx := range idxs {
		pos[i] = positions[idx]
	}
	return pos
}

// return negative score and zero val Pos{} if no play has positive score
func scoreCardAI(cardName string, color string, boardScore int, m *Match) (int, Pos) {
	validPositions := idxsToPos(validCardPositions(cardName, color, m))
	scores := make([]int, len(validPositions))
	for i, pos := range validPositions {
		scores[i] = scoreCardAIPos(cardName, pos, color, boardScore, m)
		//fmt.Printf("card %s: score %v, pos %v\n", cardName, scores[i], pos)
	}
	winnerIdxs := []int{}
	winningScore := 0 // winning score must be greater than zero
	for i, score := range scores {
		if score > winningScore {
			winningScore = score
			winnerIdxs = []int{i}
		} else if score == winningScore {
			winnerIdxs = append(winnerIdxs, i)
		}
	}
	if len(winnerIdxs) == 0 {
		return -1, Pos{}
	}
	winnerIdx := winnerIdxs[rand.Intn(len(winnerIdxs))]
	return scores[winnerIdx], validPositions[winnerIdx]
}
