package main

import (
	"math/rand"
)

func kingPlacement(color string, board []*Piece) Pos {
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
