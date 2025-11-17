package game

import (
	"encoding/json"
	"fmt"
	"net/http"
	"power4/src/auth"
	"power4/src/logger"
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

	// Récupérer le nom d'utilisateur pour les logs
	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		username = "Inconnu"
	}

	// Placer le jeton du joueur
	wasOver := currentGame.Over
	if !placeToken(currentGame, col) {
		http.Error(w, "Coup invalide - colonne pleine", http.StatusBadRequest)
		return
	}

	// Logger si la partie vient de se terminer (après le coup du joueur)
	if !wasOver && currentGame.Over {
		mode := "2 Joueurs"
		difficulty := ""
		if currentGame.IsAIMode {
			mode = "IA"
			difficulty = currentGame.AIDifficulty
		}

		// Déterminer le résultat pour le joueur
		if currentGame.Message == "Match nul !" {
			logger.LogGameDraw(username, mode, difficulty, currentGame.IsAIMode)
		} else if currentGame.IsAIMode {
			// En mode IA, le joueur 1 est toujours l'utilisateur
			// Si currentGame.Current == 1 après placeToken, c'est que le joueur a gagné
			// (car placeToken ne change pas Current si la partie est terminée)
			if currentGame.Current == 1 {
				logger.LogGameWin(username, mode, difficulty, currentGame.IsAIMode)
			} else {
				// L'IA a gagné, donc le joueur a perdu
				logger.LogGameLoss(username, mode, difficulty, currentGame.IsAIMode)
			}
		} else {
			// Mode 2 joueurs - on suppose que le joueur connecté est le joueur 1
			if currentGame.Current == 1 {
				logger.LogGameWin(username, mode, difficulty, currentGame.IsAIMode)
			} else {
				logger.LogGameLoss(username, mode, difficulty, currentGame.IsAIMode)
			}
		}
	}

	// Si mode IA et c'est le tour de l'IA
	if currentGame.IsAIMode && currentGame.Current == 2 && !currentGame.Over {
		aiCol := currentGame.AI.PlayMove(currentGame)
		if aiCol != -1 {
			placeToken(currentGame, aiCol)
			// Logger si l'IA a gagné après son coup
			if currentGame.Over && currentGame.Current == 2 {
				mode := "IA"
				difficulty := currentGame.AIDifficulty
				logger.LogGameLoss(username, mode, difficulty, currentGame.IsAIMode)
			} else if currentGame.Over && currentGame.Message == "Match nul !" {
				// Match nul après le coup de l'IA
				mode := "IA"
				difficulty := currentGame.AIDifficulty
				logger.LogGameDraw(username, mode, difficulty, currentGame.IsAIMode)
			}
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

	// Récupérer le nom d'utilisateur pour les logs
	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		username = "Inconnu"
	}

	// Si c'est une partie IA, redémarrer avec les mêmes paramètres
	if currentGame != nil && currentGame.IsAIMode {
		currentGame = newAIGame(6, 7, currentGame.AIDifficulty)
		// Logger le démarrage de partie
		logger.LogGameStart(username, "IA", currentGame.AIDifficulty, true)
	} else {
		// TOUJOURS créer un nouveau jeu selon la difficulté choisie (même si un jeu existe)
		var difficulty string
		if hdifficulty == "easy" {
			g := newGame(6, 7)
			g.WinLength = 3
			currentGame = g
			difficulty = "easy"
		} else if hdifficulty == "hard" {
			g := newGame(7, 8)
			g.WinLength = 7
			currentGame = g
			difficulty = "hard"
		} else {
			// Normal par défaut
			g := newGame(6, 7)
			g.WinLength = 4
			currentGame = g
			difficulty = "normal"
		}
		// Logger le démarrage de partie
		logger.LogGameStart(username, "2 Joueurs", difficulty, false)
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

	// Récupérer le nom d'utilisateur pour les logs
	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		username = "Inconnu"
	}

	// Logger le démarrage de partie
	logger.LogGameStart(username, "IA", difficulty, true)

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
