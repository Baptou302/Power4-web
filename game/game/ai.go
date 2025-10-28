package game

import (
	"math"
	"math/rand"
	"time"
)

const (
	DifficultyEasy       = "easy"
	DifficultyMedium     = "medium"
	DifficultyHard       = "hard"
	DifficultyImpossible = "impossible"
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
	case DifficultyImpossible:
		return ai.playImpossible(game)
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

	bestCol := ai.minimaxBestMove(game, 4)
	if bestCol != -1 && ai.isValidMove(game, bestCol) {
		return bestCol
	}

	order := ai.centerOrder(game.Cols)
	for _, col := range order {
		if ai.isValidMove(game, col) {
			return col
		}
	}
	return ai.playEasy(game)
}

// playImpossible tries to be unbeatable by combining tactical checks with
// a deeper minimax search and better move ordering.
func (ai *AI) playImpossible(game *Game) int {
	// 1) Win immediately if possible
	winCol := ai.findWinningMove(game, ai.Player)
	if winCol != -1 {
		return winCol
	}

	// 2) Block opponent immediate win
	opponent := 3 - ai.Player
	blockCol := ai.findWinningMove(game, opponent)
	if blockCol != -1 {
		return blockCol
	}

	// 3) Deep search with improved ordering and pruning (fast + strong)
	bestCol := ai.minimaxBestMoveAlphaBeta(game, 6) // depth 6 with pruning
	if bestCol != -1 && ai.isValidMove(game, bestCol) {
		return bestCol
	}

	// 4) Prefer center columns
	order := ai.centerOrder(game.Cols)
	for _, col := range order {
		if ai.isValidMove(game, col) {
			return col
		}
	}

	return ai.playEasy(game)
}

func (ai *AI) centerOrder(cols int) []int {
	// Returns column indices ordered from center outward
	order := []int{}
	centerLeft := (cols - 1) / 2
	centerRight := cols / 2
	used := map[int]bool{}
	for offset := 0; offset < cols; offset++ {
		for _, c := range []int{centerLeft - offset, centerRight + offset} {
			if c >= 0 && c < cols && !used[c] {
				order = append(order, c)
				used[c] = true
			}
		}
	}
	return order
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

func (ai *AI) minimaxBestMove(game *Game, depth int) int {
	bestScore := math.MinInt
	bestCol := -1
	// Order columns from center outward for better pruning
	order := ai.centerOrder(game.Cols)
	for _, col := range order {
		if !ai.isValidMove(game, col) {
			continue
		}
		row := ai.getNextEmptyRow(game, col)
		game.Board[row][col] = ai.Player
		score := ai.minimaxScore(game, depth-1, false)
		game.Board[row][col] = 0
		if score > bestScore {
			bestScore = score
			bestCol = col
		}
	}
	return bestCol
}

func (ai *AI) minimaxBestMoveAlphaBeta(game *Game, depth int) int {
	bestScore := math.MinInt
	bestCol := -1
	alpha := math.MinInt
	beta := math.MaxInt
	for _, col := range ai.centerOrder(game.Cols) {
		if !ai.isValidMove(game, col) {
			continue
		}
		row := ai.getNextEmptyRow(game, col)
		game.Board[row][col] = ai.Player
		score := ai.minimaxScoreAlphaBeta(game, depth-1, false, alpha, beta)
		game.Board[row][col] = 0
		if score > bestScore {
			bestScore = score
			bestCol = col
		}
		if bestScore > alpha {
			alpha = bestScore
		}
	}
	return bestCol
}

func (ai *AI) minimaxScore(game *Game, depth int, maximizing bool) int {
	// Terminal states
	if checkWin(game.Board, ai.Player, 4) {
		return 100000 - (4 - depth)
	}
	if checkWin(game.Board, 3-ai.Player, 4) {
		return -100000 + (4 - depth)
	}
	if depth == 0 {
		return ai.evaluateBoard(game)
	}

	valid := ai.getValidColumns(game)
	if len(valid) == 0 {
		return 0
	}

	if maximizing {
		best := math.MinInt
		for _, col := range valid {
			row := ai.getNextEmptyRow(game, col)
			game.Board[row][col] = ai.Player
			score := ai.minimaxScore(game, depth-1, false)
			game.Board[row][col] = 0
			if score > best {
				best = score
			}
		}
		return best
	} else {
		best := math.MaxInt
		opp := 3 - ai.Player
		for _, col := range valid {
			row := ai.getNextEmptyRow(game, col)
			game.Board[row][col] = opp
			score := ai.minimaxScore(game, depth-1, true)
			game.Board[row][col] = 0
			if score < best {
				best = score
			}
		}
		return best
	}
}

func (ai *AI) minimaxScoreAlphaBeta(game *Game, depth int, maximizing bool, alpha, beta int) int {
	// Terminal states
	if checkWin(game.Board, ai.Player, 4) {
		return 100000 - (4 - depth)
	}
	if checkWin(game.Board, 3-ai.Player, 4) {
		return -100000 + (4 - depth)
	}
	if depth == 0 {
		return ai.evaluateBoard(game)
	}

	// Order columns for better pruning
	ordered := ai.centerOrder(game.Cols)

	if maximizing {
		best := math.MinInt
		for _, col := range ordered {
			if !ai.isValidMove(game, col) {
				continue
			}
			row := ai.getNextEmptyRow(game, col)
			game.Board[row][col] = ai.Player
			score := ai.minimaxScoreAlphaBeta(game, depth-1, false, alpha, beta)
			game.Board[row][col] = 0
			if score > best {
				best = score
			}
			if best > alpha {
				alpha = best
			}
			if beta <= alpha {
				break
			}
		}
		return best
	} else {
		best := math.MaxInt
		opp := 3 - ai.Player
		for _, col := range ordered {
			if !ai.isValidMove(game, col) {
				continue
			}
			row := ai.getNextEmptyRow(game, col)
			game.Board[row][col] = opp
			score := ai.minimaxScoreAlphaBeta(game, depth-1, true, alpha, beta)
			game.Board[row][col] = 0
			if score < best {
				best = score
			}
			if best < beta {
				beta = best
			}
			if beta <= alpha {
				break
			}
		}
		return best
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
