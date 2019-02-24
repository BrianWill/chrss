package main

import (
	"math/rand"
	"sort"
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

// return score of a board position's exposure to attack
// based on current threats and potential (open lines of sight)
func exposureScore(color string, idx int, board []*Piece) int {
	return 0
}

// return score of a board position's suitability for attack
// based on current targets and potential (open lines of sight)
func offenseScore(color string, pieceType string, idx int, board []*Piece) int {
	return 0
}

func playTurnAI(color string, m *Match) {
	public, private := m.states(color)

	scores := make([]int, len(private.Cards))
	pos := make([]Pos, len(private.Cards)) // for the scored card, the chosen Pos to 'click'
	for i, c := range private.Cards {
		if private.PlayableCards[i] {
			scores[i], pos[i] = scoreCardAI(c.Name, color, m)
		}
	}

	// determine highest score, pick random from tie for first
	highestIdxs := []int{}
	highestScore := -1000
	for i, score := range scores {
		if score > highestScore {
			highestScore = score
			highestIdxs = []int{i}
		} else if score == highestScore {
			highestIdxs = append(highestIdxs, i)
		}
	}
	if len(highestIdxs) == 0 {
		m.EndTurn(true, color)
		return
	}
	selectedIdx := highestIdxs[rand.Intn(len(highestIdxs))]
	if scores[selectedIdx] < 0 {
		m.EndTurn(true, color)
		return
	}
	private.SelectedCard = selectedIdx
	m.clickBoard(color, public, private, pos[selectedIdx])
}

// assumes card/pos combo is a valid play
func scoreCardAIPos(cardName string, pos Pos, color string, m *Match) int {
	score := 0
	switch cardName {
	case castleCard:

	case reclaimVassalCard:

	case swapFrontLinesCard:

	case removePawnCard:

	case forceCombatCard:

	case dispellCard:

	case dodgeCard:

	case mirrorCard:

	case drainManaCard:

	case healCard:

	case poisonCard:

	case togglePawnCard:

	case nukeCard:

	case vulnerabilityCard:

	case amplifyCard:

	case transparencyCard:

	case stunVassalCard:

	case enrageCard:

	case armorCard:

	case shoveCard:

	case advanceCard:

	case restoreManaCard:

	case summonPawnCard:

	case resurrectVassalCard:

	case bishop, knight, rook, queen, jester:

	}
	return score
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

func validPositionsForCard(cardName string, color string, m *Match) []Pos {
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
		idxs = pieceIdxs(none, board)
	case reclaimVassalCard:
		idxs = vassalIdxs(color, board)
	case rook, bishop, knight, queen, jester:
		idxs = freeIdxs(color, board)
	}
	pos := make([]Pos, len(idxs))
	for i, idx := range idxs {
		pos[i] = positions[idx]
	}
	return pos
}

// return negative score and zero val Pos{} if no play has positive score
func scoreCardAI(cardName string, color string, m *Match) (int, Pos) {
	validPositions := validPositionsForCard(cardName, color, m)
	scores := make([]int, len(validPositions))
	for i, pos := range validPositions {
		scores[i] = scoreCardAIPos(cardName, pos, color, m)
	}
	const minScore = -1000
	winners := []int{}
	winningScore := minScore
	for i, score := range scores {
		if score > winningScore {
			winningScore = score
			winners = []int{i}
		} else if score == winningScore {
			winners = append(winners, i)
		}
	}
	if len(winners) == 0 {
		return -1, Pos{}
	}
	winnerIdx := winners[rand.Intn(len(winners))]
	return scores[winnerIdx], validPositions[winnerIdx]
}
