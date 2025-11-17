package game

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

type Game struct {
	Rows, Cols int
	Board      [][]int
	Current    int
	Over       bool
	Message    string
	Turn       int
	GravityUp  bool
	Mutex      sync.Mutex
	WinLength  int
	// Mode IA
	IsAIMode     bool
	AIDifficulty string
	AI           *AI
}

var currentGame *Game

func newGame(rows, cols int) *Game {
	board := make([][]int, rows)
	for i := range board {
		board[i] = make([]int, cols)
	}
	return &Game{
		Rows:      rows,
		Cols:      cols,
		Board:     board,
		Current:   1,
		GravityUp: false,
		WinLength: 4,
		IsAIMode:  false,
	}
}

func newAIGame(rows, cols int, difficulty string) *Game {
	game := newGame(rows, cols)
	game.IsAIMode = true
	game.AIDifficulty = difficulty
	game.AI = &AI{
		Difficulty: difficulty,
		Player:     2, // L'IA est toujours le joueur 2
	}
	return game
}

// checkWin vérifie si un joueur a gagné en vérifiant les lignes horizontales, verticales et diagonales
func checkWin(board [][]int, player int, winLen int) bool {
	rows := len(board)
	if rows == 0 {
		return false
	}
	cols := len(board[0])

	// Horizontal
	for r := 0; r < rows; r++ {
		for c := 0; c <= cols-winLen; c++ {
			ok := true
			for i := 0; i < winLen; i++ {
				if board[r][c+i] != player {
					ok = false
					break
				}
			}
			if ok {
				return true
			}
		}
	}

	// Vertical
	for r := 0; r <= rows-winLen; r++ {
		for c := 0; c < cols; c++ {
			ok := true
			for i := 0; i < winLen; i++ {
				if board[r+i][c] != player {
					ok = false
					break
				}
			}
			if ok {
				return true
			}
		}
	}

	// Diagonal down-right
	for r := 0; r <= rows-winLen; r++ {
		for c := 0; c <= cols-winLen; c++ {
			ok := true
			for i := 0; i < winLen; i++ {
				if board[r+i][c+i] != player {
					ok = false
					break
				}
			}
			if ok {
				return true
			}
		}
	}

	// Diagonal up-right
	for r := 0; r <= rows-winLen; r++ {
		for c := winLen - 1; c < cols; c++ {
			ok := true
			for i := 0; i < winLen; i++ {
				if board[r+i][c-i] != player {
					ok = false
					break
				}
			}
			if ok {
				return true
			}
		}
	}

	return false
}

// isFull vérifie si le plateau est plein
func isFull(board [][]int) bool {
	for _, row := range board {
		for _, cell := range row {
			if cell == 0 {
				return false
			}
		}
	}
	return true
}

// placeToken place un jeton dans la colonne spécifiée
func placeToken(game *Game, col int) bool {
	if col < 0 || col >= game.Cols {
		return false
	}

	// Trouver la première case vide depuis le bas
	for row := game.Rows - 1; row >= 0; row-- {
		if game.Board[row][col] == 0 {
			game.Board[row][col] = game.Current
			game.Turn++

			// Vérifier la victoire
			if checkWin(game.Board, game.Current, game.WinLength) {
				game.Over = true
				if game.IsAIMode && game.Current == 2 {
					game.Message = "L'IA gagne ! 🤖"
				} else if game.IsAIMode && game.Current == 1 {
					game.Message = "Vous gagnez ! 🎉"
				} else {
					game.Message = fmt.Sprintf("Le joueur %d gagne !", game.Current)
				}
				return true
			}

			// Vérifier le match nul
			if isFull(game.Board) {
				game.Over = true
				game.Message = "Match nul !"
				return true
			}

			// Changer de joueur
			game.Current = 3 - game.Current
			return true
		}
	}
	return false
}

// HandlePlay gère un coup joué
func HandlePlay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	colStr := r.FormValue("col")
	if colStr == "" {
		http.Error(w, "Colonne manquante", http.StatusBadRequest)
		return
	}

	col, err := strconv.Atoi(colStr)
	if err != nil {
		http.Error(w, "Colonne invalide: "+err.Error(), http.StatusBadRequest)
		return
	}

	// S'assurer qu'un jeu existe
	// Si aucun jeu n'existe, ne pas créer de jeu par défaut
	// Le jeu doit être créé via /reset ou /new-ai-game
	if currentGame == nil {
		http.Error(w, "Aucune partie en cours. Veuillez démarrer une nouvelle partie.", http.StatusBadRequest)
		return
	}

	currentGame.Mutex.Lock()
	defer currentGame.Mutex.Unlock()

	if currentGame.Over {
		http.Error(w, "Partie terminée", http.StatusConflict)
		return
	}

	// Vérifier que la colonne est valide
	if col < 0 || col >= currentGame.Cols {
		http.Error(w, fmt.Sprintf("Colonne invalide: %d (doit être entre 0 et %d)", col, currentGame.Cols-1), http.StatusBadRequest)
		return
	}

	// Placer le jeton du joueur
	if !placeToken(currentGame, col) {
		http.Error(w, "Coup invalide - colonne pleine", http.StatusBadRequest)
		return
	}

	// Si mode IA et c'est le tour de l'IA
	if currentGame.IsAIMode && currentGame.Current == 2 && !currentGame.Over {
		aiCol := currentGame.AI.PlayMove(currentGame)
		if aiCol != -1 {
			placeToken(currentGame, aiCol)
		}
	}

	// Retourner l'état du jeu
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"board":         currentGame.Board,
		"currentPlayer": currentGame.Current,
		"message":       currentGame.Message,
		"over":          currentGame.Over,
		"isAIMode":      currentGame.IsAIMode,
		"aiDifficulty":  currentGame.AIDifficulty,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Erreur lors de l'encodage JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleReset réinitialise la partie
func HandleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	// Parser le formulaire
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Erreur lors du parsing du formulaire: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Essayer plusieurs méthodes pour récupérer hdifficulty
	hdifficulty := r.FormValue("hdifficulty")
	if hdifficulty == "" {
		// Essayer depuis l'URL
		hdifficulty = r.URL.Query().Get("hdifficulty")
	}
	if hdifficulty == "" {
		// Essayer depuis le body
		if r.PostForm != nil {
			hdifficulty = r.PostForm.Get("hdifficulty")
		}
	}

	// Si c'est une partie IA, redémarrer avec les mêmes paramètres
	if currentGame != nil && currentGame.IsAIMode {
		currentGame = newAIGame(6, 7, currentGame.AIDifficulty)
	} else {
		// TOUJOURS créer un nouveau jeu selon la difficulté choisie (même si un jeu existe)
		if hdifficulty == "easy" {
			g := newGame(6, 7)
			g.WinLength = 3
			currentGame = g
		} else if hdifficulty == "hard" {
			g := newGame(7, 8)
			g.WinLength = 7
			currentGame = g
		} else {
			// Normal par défaut
			g := newGame(6, 7)
			g.WinLength = 4
			currentGame = g
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"board":         currentGame.Board,
		"currentPlayer": currentGame.Current,
		"message":       currentGame.Message,
		"over":          currentGame.Over,
		"rows":          currentGame.Rows,
		"cols":          currentGame.Cols,
		"winLength":     currentGame.WinLength,
	}); err != nil {
		http.Error(w, "Erreur lors de l'encodage JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleNewAIGame crée une nouvelle partie contre l'IA
func HandleNewAIGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	difficulty := r.FormValue("difficulty")
	if difficulty == "" {
		difficulty = "easy"
	}

	// Valider la difficulté
	validDifficulties := map[string]bool{
		"easy":       true,
		"medium":     true,
		"hard":       true,
		"impossible": true,
	}
	if !validDifficulties[difficulty] {
		http.Error(w, "Difficulté invalide", http.StatusBadRequest)
		return
	}

	currentGame = newAIGame(6, 7, difficulty)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"board":         currentGame.Board,
		"currentPlayer": currentGame.Current,
		"message":       fmt.Sprintf("Nouvelle partie contre l'IA (%s) ! Vous commencez.", difficulty),
		"over":          currentGame.Over,
		"isAIMode":      currentGame.IsAIMode,
		"aiDifficulty":  currentGame.AIDifficulty,
	})
}
