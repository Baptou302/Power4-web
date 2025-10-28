package game

import (
	"math/rand"
	"time"
)

const (
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
)

type AI struct {
	Difficulty string
	Player     int
}

func NewAI(difficulty string, player int) *AI {
	return &AI{
		Difficulty: difficulty,
		Player:     player,
	}
}

func (ai *AI) PlayMove(game *Game) int {
	switch ai.Difficulty {
	case DifficultyEasy:
		return ai.playEasy(game)
	case DifficultyMedium:
		return ai.playMedium(game)
	case DifficultyHard:
		return ai.playHard(game)
	default:
		return ai.playEasy(game)
	}
}

func (ai *AI) playEasy(game *Game) int {
	validCols := ai.getValidColumns(game)
	if len(validCols) == 0 {
		return -1
	}

	rand.Seed(time.Now().UnixNano())
	return validCols[rand.Intn(len(validCols))]
}

func (ai *AI) playMedium(game *Game) int {
	winCol := ai.findWinningMove(game, ai.Player)
	if winCol != -1 {
		return winCol
	}

	opponent := 3 - ai.Player
	blockCol := ai.findWinningMove(game, opponent)
	if blockCol != -1 {
		return blockCol
	}

	centerCols := []int{3, 2, 4, 1, 5, 0, 6}
	for _, col := range centerCols {
		if ai.isValidMove(game, col) {
			return col
		}
	}

	return ai.playEasy(game)
}

func (ai *AI) playHard(game *Game) int {
	winCol := ai.findWinningMove(game, ai.Player)
	if winCol != -1 {
		return winCol
	}

	opponent := 3 - ai.Player
	blockCol := ai.findWinningMove(game, opponent)
	if blockCol != -1 {
		return blockCol
	}

	bestCol := ai.minimax(game, 4, true)
	if bestCol != -1 && ai.isValidMove(game, bestCol) {
		return bestCol
	}

	centerCols := []int{3, 2, 4, 1, 5, 0, 6}
	for _, col := range centerCols {
		if ai.isValidMove(game, col) {
			return col
		}
	}

	return ai.playEasy(game)
}

func (ai *AI) findWinningMove(game *Game, player int) int {
	for col := 0; col < game.Cols; col++ {
		if !ai.isValidMove(game, col) {
			continue
		}

		row := ai.getNextEmptyRow(game, col)
		game.Board[row][col] = player

		if checkWin(game.Board, player, 4) {
			game.Board[row][col] = 0
			return col
		}

		game.Board[row][col] = 0
	}
	return -1
}

func (ai *AI) isValidMove(game *Game, col int) bool {
	if col < 0 || col >= game.Cols {
		return false
	}
	return game.Board[0][col] == 0
}

func (ai *AI) getNextEmptyRow(game *Game, col int) int {
	for row := game.Rows - 1; row >= 0; row-- {
		if game.Board[row][col] == 0 {
			return row
		}
	}
	return -1
}

func (ai *AI) getValidColumns(game *Game) []int {
	var validCols []int
	for col := 0; col < game.Cols; col++ {
		if ai.isValidMove(game, col) {
			validCols = append(validCols, col)
		}
	}
	return validCols
}

func (ai *AI) minimax(game *Game, depth int, maximizingPlayer bool) int {
	if depth == 0 || game.Over {
		return ai.evaluateBoard(game)
	}

	validCols := ai.getValidColumns(game)
	if len(validCols) == 0 {
		return 0
	}

	bestCol := validCols[0]

	if maximizingPlayer {
		maxEval := -1000
		for _, col := range validCols {
			row := ai.getNextEmptyRow(game, col)
			game.Board[row][col] = ai.Player

			eval := ai.minimax(game, depth-1, false)
			game.Board[row][col] = 0

			if eval > maxEval {
				maxEval = eval
				bestCol = col
			}
		}
		return bestCol
	} else {
		minEval := 1000
		opponent := 3 - ai.Player
		for _, col := range validCols {
			row := ai.getNextEmptyRow(game, col)
			game.Board[row][col] = opponent

			eval := ai.minimax(game, depth-1, true)
			game.Board[row][col] = 0

			if eval < minEval {
				minEval = eval
				bestCol = col
			}
		}
		return bestCol
	}
}

func (ai *AI) evaluateBoard(game *Game) int {
	score := 0

	if checkWin(game.Board, ai.Player, 4) {
		return 1000
	}
	if checkWin(game.Board, 3-ai.Player, 4) {
		return -1000
	}

	score += ai.evaluatePosition(game.Board, ai.Player)
	score -= ai.evaluatePosition(game.Board, 3-ai.Player)

	return score
}

func (ai *AI) evaluatePosition(board [][]int, player int) int {
	score := 0

	score += ai.countConsecutive(board, player, 3) * 100
	score += ai.countConsecutive(board, player, 2) * 10
	score += ai.countConsecutive(board, player, 1) * 1

	return score
}

func (ai *AI) countConsecutive(board [][]int, player int, length int) int {
	count := 0
	rows := len(board)
	cols := len(board[0])

	for r := 0; r < rows; r++ {
		for c := 0; c <= cols-length; c++ {
			consecutive := 0
			for i := 0; i < length; i++ {
				if board[r][c+i] == player {
					consecutive++
				}
			}
			if consecutive == length {
				count++
			}
		}
	}

	for r := 0; r <= rows-length; r++ {
		for c := 0; c < cols; c++ {
			consecutive := 0
			for i := 0; i < length; i++ {
				if board[r+i][c] == player {
					consecutive++
				}
			}
			if consecutive == length {
				count++
			}
		}
	}

	for r := 0; r <= rows-length; r++ {
		for c := 0; c <= cols-length; c++ {
			consecutive := 0
			for i := 0; i < length; i++ {
				if board[r+i][c+i] == player {
					consecutive++
				}
			}
			if consecutive == length {
				count++
			}
		}
	}

	for r := 0; r <= rows-length; r++ {
		for c := length - 1; c < cols; c++ {
			consecutive := 0
			for i := 0; i < length; i++ {
				if board[r+i][c-i] == player {
					consecutive++
				}
			}
			if consecutive == length {
				count++
			}
		}
	}

	return count
}
