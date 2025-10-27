package game

import (
	"math/rand"
	"time"
)

// Niveaux de difficulté de l'IA
const (
	DifficultyEasy   = "easy"
	DifficultyMedium = "medium"
	DifficultyHard   = "hard"
)

// Structure pour l'IA
type AI struct {
	Difficulty string
	Player     int // 1 ou 2
}

// Nouvelle instance d'IA
func NewAI(difficulty string, player int) *AI {
	return &AI{
		Difficulty: difficulty,
		Player:     player,
	}
}

// Fonction principale pour que l'IA joue
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

// IA Facile : Joue aléatoirement
func (ai *AI) playEasy(game *Game) int {
	validCols := ai.getValidColumns(game)
	if len(validCols) == 0 {
		return -1
	}

	rand.Seed(time.Now().UnixNano())
	return validCols[rand.Intn(len(validCols))]
}

// IA Intermédiaire : Joue intelligemment mais pas parfaitement
func (ai *AI) playMedium(game *Game) int {
	// 1. Vérifier si on peut gagner
	winCol := ai.findWinningMove(game, ai.Player)
	if winCol != -1 {
		return winCol
	}

	// 2. Vérifier si l'adversaire peut gagner (bloquer)
	opponent := 3 - ai.Player
	blockCol := ai.findWinningMove(game, opponent)
	if blockCol != -1 {
		return blockCol
	}

	// 3. Jouer au centre si possible
	centerCols := []int{3, 2, 4, 1, 5, 0, 6}
	for _, col := range centerCols {
		if ai.isValidMove(game, col) {
			return col
		}
	}

	// 4. Sinon jouer aléatoirement
	return ai.playEasy(game)
}

// IA Hardcore : Joue de manière optimale
func (ai *AI) playHard(game *Game) int {
	// 1. Vérifier si on peut gagner
	winCol := ai.findWinningMove(game, ai.Player)
	if winCol != -1 {
		return winCol
	}

	// 2. Vérifier si l'adversaire peut gagner (bloquer)
	opponent := 3 - ai.Player
	blockCol := ai.findWinningMove(game, opponent)
	if blockCol != -1 {
		return blockCol
	}

	// 3. Utiliser l'algorithme minimax pour le meilleur coup
	bestCol := ai.minimax(game, 4, true) // Profondeur de 4
	if bestCol != -1 && ai.isValidMove(game, bestCol) {
		return bestCol
	}

	// 4. Fallback : jouer au centre
	centerCols := []int{3, 2, 4, 1, 5, 0, 6}
	for _, col := range centerCols {
		if ai.isValidMove(game, col) {
			return col
		}
	}

	// 5. Dernier recours : aléatoire
	return ai.playEasy(game)
}

// Trouve un coup gagnant pour un joueur donné
func (ai *AI) findWinningMove(game *Game, player int) int {
	for col := 0; col < game.Cols; col++ {
		if !ai.isValidMove(game, col) {
			continue
		}

		// Simuler le coup
		row := ai.getNextEmptyRow(game, col)
		game.Board[row][col] = player

		// Vérifier si c'est gagnant
		if checkWin(game.Board, player) {
			game.Board[row][col] = 0 // Annuler le coup
			return col
		}

		game.Board[row][col] = 0 // Annuler le coup
	}
	return -1
}

// Vérifie si un coup est valide
func (ai *AI) isValidMove(game *Game, col int) bool {
	if col < 0 || col >= game.Cols {
		return false
	}
	return game.Board[0][col] == 0
}

// Trouve la prochaine ligne vide dans une colonne
func (ai *AI) getNextEmptyRow(game *Game, col int) int {
	for row := game.Rows - 1; row >= 0; row-- {
		if game.Board[row][col] == 0 {
			return row
		}
	}
	return -1
}

// Obtient toutes les colonnes valides
func (ai *AI) getValidColumns(game *Game) []int {
	var validCols []int
	for col := 0; col < game.Cols; col++ {
		if ai.isValidMove(game, col) {
			validCols = append(validCols, col)
		}
	}
	return validCols
}

// Algorithme Minimax pour l'IA Hardcore
func (ai *AI) minimax(game *Game, depth int, maximizingPlayer bool) int {
	if depth == 0 || game.Over {
		return ai.evaluateBoard(game)
	}

	validCols := ai.getValidColumns(game)
	if len(validCols) == 0 {
		return 0 // Match nul
	}

	bestCol := validCols[0]

	if maximizingPlayer {
		maxEval := -1000
		for _, col := range validCols {
			row := ai.getNextEmptyRow(game, col)
			game.Board[row][col] = ai.Player

			eval := ai.minimax(game, depth-1, false)
			game.Board[row][col] = 0 // Annuler le coup

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
			game.Board[row][col] = 0 // Annuler le coup

			if eval < minEval {
				minEval = eval
				bestCol = col
			}
		}
		return bestCol
	}
}

// Évalue la position du plateau
func (ai *AI) evaluateBoard(game *Game) int {
	score := 0

	// Vérifier les victoires
	if checkWin(game.Board, ai.Player) {
		return 1000
	}
	if checkWin(game.Board, 3-ai.Player) {
		return -1000
	}

	// Évaluer les positions
	score += ai.evaluatePosition(game.Board, ai.Player)
	score -= ai.evaluatePosition(game.Board, 3-ai.Player)

	return score
}

// Évalue la position d'un joueur
func (ai *AI) evaluatePosition(board [][]int, player int) int {
	score := 0

	// Évaluer les lignes de 3, 2, 1 pions alignés
	score += ai.countConsecutive(board, player, 3) * 100
	score += ai.countConsecutive(board, player, 2) * 10
	score += ai.countConsecutive(board, player, 1) * 1

	return score
}

// Compte les séquences consécutives d'un joueur
func (ai *AI) countConsecutive(board [][]int, player int, length int) int {
	count := 0
	rows := len(board)
	cols := len(board[0])

	// Horizontal
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

	// Vertical
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

	// Diagonales
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
