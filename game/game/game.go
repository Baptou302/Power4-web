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

// --- STRUCTURE DU JEU ---
type Game struct {
	Rows, Cols int
	Board      [][]int
	Current    int
	Over       bool
	Message    string
	Turn       int
	GravityUp  bool
	Mutex      sync.Mutex
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
		GravityUp: false, // Gravité vers le bas par défaut
	}
}

// --- HANDLERS WEB ---
func HandleIndex(w http.ResponseWriter, r *http.Request) {
	user := GetUsernameFromRequest(r)
	if user == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if currentGame == nil {
		currentGame = newGame(6, 7)
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

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
        "board": %s,
        "currentPlayer": %d,
        "message": "%s",
        "over": %t
    }`, toJSON(currentGame.Board), currentGame.Current, currentGame.Message, currentGame.Over)
}

func HandleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	currentGame = newGame(6, 7)

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
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// --- LOGIQUE DU JEU ---
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
	// La gravité reste toujours vers le bas pour un Puissance 4 normal
	// g.GravityUp reste false par défaut

	if checkWin(g.Board, g.Current) {
		g.Over = true
		g.Message = fmt.Sprintf("Le joueur %d gagne !", g.Current)
		return
	}

	if isFull(g.Board) {
		g.Over = true
		g.Message = "Match nul !"
		return
	}

	g.Current = 3 - g.Current
}

// --- UTILITAIRES DB ---
// La fonction isUsernameTaken est maintenant dans db.go

// --- FONCTIONS UTILITAIRES ---
func checkWin(board [][]int, player int) bool {
	rows := len(board)
	cols := len(board[0])

	// horizontal
	for r := 0; r < rows; r++ {
		for c := 0; c < cols-3; c++ {
			if board[r][c] == player && board[r][c+1] == player &&
				board[r][c+2] == player && board[r][c+3] == player {
				return true
			}
		}
	}

	// vertical
	for r := 0; r < rows-3; r++ {
		for c := 0; c < cols; c++ {
			if board[r][c] == player && board[r+1][c] == player &&
				board[r+2][c] == player && board[r+3][c] == player {
				return true
			}
		}
	}

	// diagonales ↘ et ↙
	for r := 0; r < rows-3; r++ {
		for c := 0; c < cols-3; c++ {
			if board[r][c] == player && board[r+1][c+1] == player &&
				board[r+2][c+2] == player && board[r+3][c+3] == player {
				return true
			}
		}
		for c := 3; c < cols; c++ {
			if board[r][c] == player && board[r+1][c-1] == player &&
				board[r+2][c-2] == player && board[r+3][c-3] == player {
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
	DeleteSession(w, r) // ou ta logique pour supprimer le cookie/session
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
func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
