package game

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"
)

// --- STRUCTURE DU JEU ---

type Game struct {
	Rows, Cols int
	Board      [][]int
	Current    int
	Over       bool
	Message    string
	Turn       int
	GravityUp  bool // pour le mode inverse
	Mutex      sync.Mutex
}

var currentGame *Game

// Nouvelle partie
func newGame(rows, cols int) *Game {
	board := make([][]int, rows)
	for i := range board {
		board[i] = make([]int, cols)
	}
	return &Game{
		Rows:    rows,
		Cols:    cols,
		Board:   board,
		Current: 1,
	}
}

// --- HANDLERS WEB ---

func HandleIndex(w http.ResponseWriter, r *http.Request) {
	if currentGame == nil {
		currentGame = newGame(6, 7)
	}

	tmplPath := filepath.Join("templates", "index.html")
	funcMap := template.FuncMap{
		"seq": func(start, end int) []int {
			s := make([]int, end-start)
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
		"Board":   currentGame.Board,
		"Current": currentGame.Current,
		"Message": currentGame.Message,
		"Over":    currentGame.Over,
		"Rows":    currentGame.Rows,
		"Cols":    currentGame.Cols,
		"Gravity": currentGame.GravityUp,
	}

	tmpl.Execute(w, data)
}

func HandlePlay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	colStr := r.FormValue("col")
	col, err := strconv.Atoi(colStr)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	currentGame.Mutex.Lock()
	defer currentGame.Mutex.Unlock()

	if currentGame.Over {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	placeToken(col)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func HandleReset(w http.ResponseWriter, r *http.Request) {
	currentGame = newGame(6, 7)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Page de login
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl, _ := template.ParseFiles(filepath.Join("templates", "login.html"))
		tmpl.Execute(w, nil)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	err := ValidateUser(username, password)
	if err != nil {
		http.Error(w, "Identifiants invalides", http.StatusUnauthorized)
		return
	}

	CreateSession(w, username)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		tmpl, _ := template.ParseFiles(filepath.Join("templates", "register.html"))
		tmpl.Execute(w, nil)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	err := RegisterUser(username, password)
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
	if g.Turn%5 == 0 { // tous les 5 tours, inversion de gravité
		g.GravityUp = !g.GravityUp
		g.Message = "⚡ Gravité inversée ! ⚡"
	}

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

	g.Current = 3 - g.Current // change de joueur
}

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
