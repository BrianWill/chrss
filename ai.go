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
			score := 0
			switch c.Name {

			}
			scores[i] = score
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
