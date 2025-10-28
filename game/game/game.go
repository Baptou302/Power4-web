package game

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"

	"golang.org/x/crypto/bcrypt"
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
	game.AI = NewAI(difficulty, 2)
	return game
}

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	user := GetUsernameFromRequest(r)
	if user == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	mode := r.URL.Query().Get("mode")
	difficulty := r.URL.Query().Get("difficulty")

	if mode == "" {
		http.Redirect(w, r, "/mode-selection", http.StatusSeeOther)
		return
	}

	if mode == "ai" && difficulty != "" {
		if currentGame == nil || !currentGame.IsAIMode || currentGame.AIDifficulty != difficulty {
			currentGame = newAIGame(6, 7, difficulty)
		}
	} else if mode == "human" {
		// Difficultés locales: easy (6x7, win 3), normal (6x7, win 4), hard (7x8, win 7)
		hd := difficulty
		if hd == "" {
			hd = r.URL.Query().Get("hdifficulty")
		}
		if currentGame == nil || currentGame.IsAIMode || hd != "" {
			switch hd {
			case "easy":
				currentGame = newGame(6, 7)
				currentGame.WinLength = 3
			case "hard":
				currentGame = newGame(7, 8)
				currentGame.WinLength = 7
			default:
				currentGame = newGame(6, 7)
				currentGame.WinLength = 4
			}
		}
	} else {
		http.Redirect(w, r, "/mode-selection", http.StatusSeeOther)
		return
	}

	tmplPath := filepath.Join("templates", "index.html")
	funcMap := template.FuncMap{
		"seq": func(start, end int) []int {
			if end < start {
				return []int{}
			}
			s := make([]int, end-start+1)
			for i := range s {
				s[i] = start + i
			}
			return s
		},
	}

	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Board":         currentGame.Board,
		"CurrentPlayer": currentGame.Current,
		"Message":       currentGame.Message,
		"Over":          currentGame.Over,
		"Rows":          currentGame.Rows,
		"Cols":          currentGame.Cols,
		"Gravity":       currentGame.GravityUp,
		"Username":      user,
		"WinLength":     currentGame.WinLength,
	}

	tmpl.Execute(w, data)
}
func HandlePlay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	user := GetUsernameFromRequest(r)
	if user == "" {
		http.Error(w, "Non autorisé", http.StatusUnauthorized)
		return
	}

	colStr := r.FormValue("col")
	col, err := strconv.Atoi(colStr)
	if err != nil {
		http.Error(w, "Colonne invalide", http.StatusBadRequest)
		return
	}

	currentGame.Mutex.Lock()
	defer currentGame.Mutex.Unlock()

	if currentGame.Over {
		http.Error(w, "Partie terminée", http.StatusConflict)
		return
	}

	placeToken(col)

	if currentGame.IsAIMode && currentGame.Current == 2 && !currentGame.Over {
		aiCol := currentGame.AI.PlayMove(currentGame)
		if aiCol != -1 {
			placeToken(aiCol)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
        "board": %s,
        "currentPlayer": %d,
        "message": "%s",
        "over": %t,
        "isAIMode": %t,
        "aiDifficulty": "%s"
    }`, toJSON(currentGame.Board), currentGame.Current, currentGame.Message, currentGame.Over, currentGame.IsAIMode, currentGame.AIDifficulty)
}

func HandleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	if currentGame == nil {
		currentGame = newGame(6, 7)
	} else {
		if currentGame.IsAIMode {
			// Redémarre une partie IA avec les mêmes paramètres
			currentGame = newAIGame(6, 7, currentGame.AIDifficulty)
		} else {
			// Redémarre une partie locale avec la même taille et condition de victoire
			g := newGame(currentGame.Rows, currentGame.Cols)
			g.WinLength = currentGame.WinLength
			currentGame = g
		}
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
        "board": %s,
        "currentPlayer": %d,
        "message": "%s",
        "over": %t
    }`, toJSON(currentGame.Board), currentGame.Current, currentGame.Message, currentGame.Over)
}

func HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl, _ := template.ParseFiles(filepath.Join("templates", "register.html"))
		tmpl.Execute(w, nil)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Erreur hash mot de passe", http.StatusInternalServerError)
		return
	}

	err = RegisterUser(username, string(hashedPassword))
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	CreateSession(w, username)
	http.Redirect(w, r, "/mode-selection", http.StatusSeeOther)
}

func placeToken(col int) {
	g := currentGame

	if g.GravityUp {
		for row := 0; row < g.Rows; row++ {
			if g.Board[row][col] == 0 {
				g.Board[row][col] = g.Current
				break
			}
		}
	} else {
		for row := g.Rows - 1; row >= 0; row-- {
			if g.Board[row][col] == 0 {
				g.Board[row][col] = g.Current
				break
			}
		}
	}

	g.Turn++

	if checkWin(g.Board, g.Current, g.WinLength) {
		g.Over = true
		if g.IsAIMode && g.Current == 2 {
			g.Message = "L'IA gagne ! 🤖"
		} else if g.IsAIMode && g.Current == 1 {
			g.Message = "Vous gagnez ! 🎉"
		} else {
			g.Message = fmt.Sprintf("Le joueur %d gagne !", g.Current)
		}
		return
	}

	if isFull(g.Board) {
		g.Over = true
		g.Message = "Match nul !"
		return
	}

	g.Current = 3 - g.Current
}

func checkWin(board [][]int, player int, winLen int) bool {
	rows := len(board)
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

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	DeleteSession(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func HandleWhoami(w http.ResponseWriter, r *http.Request) {
	username := GetUsernameFromRequest(r)
	if username == "" {
		http.Error(w, "Non connecté", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"username": "%s"}`, username)
}

func HandleModeSelection(w http.ResponseWriter, r *http.Request) {
	user := GetUsernameFromRequest(r)
	if user == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles(filepath.Join("templates", "mode-selection.html"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, nil)
}

func HandleTest(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles(filepath.Join("templates", "test-login.html"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, nil)
}

func HandleNewAIGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	user := GetUsernameFromRequest(r)
	if user == "" {
		http.Error(w, "Non autorisé", http.StatusUnauthorized)
		return
	}

	difficulty := r.FormValue("difficulty")
	if difficulty == "" {
		difficulty = DifficultyEasy
	}

	if difficulty != DifficultyEasy && difficulty != DifficultyMedium && difficulty != DifficultyHard {
		http.Error(w, "Difficulté invalide", http.StatusBadRequest)
		return
	}

	currentGame = newAIGame(6, 7, difficulty)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
        "board": %s,
        "currentPlayer": %d,
        "message": "Nouvelle partie contre l'IA (%s) ! Vous commencez.",
        "over": %t,
        "isAIMode": %t,
        "aiDifficulty": "%s"
    }`, toJSON(currentGame.Board), currentGame.Current, difficulty, currentGame.Over, currentGame.IsAIMode, currentGame.AIDifficulty)
}
func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
