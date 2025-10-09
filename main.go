package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"
)

const (
	rows    = 6
	columns = 7
)

// === STRUCTURE DU JEU ===
type Game struct {
	Board         [rows][columns]int `json:"board"`
	CurrentPlayer int                `json:"currentPlayer"`
	Message       string             `json:"message"`
	Over          bool               `json:"over"`
	Mutex         sync.Mutex
}

// === VARIABLES GLOBALES ===
var (
	game = &Game{CurrentPlayer: 1}

	tmplIndex = template.Must(template.New("index.html").Funcs(template.FuncMap{
		"seq": func(start, end int) []int {
			s := make([]int, end-start+1)
			for i := range s {
				s[i] = start + i
			}
			return s
		},
	}).ParseFiles("templates/index.html"))

	tmplLogin = template.Must(template.ParseFiles("templates/login.html"))
)

// === MAIN ===
func main() {
	mux := http.NewServeMux()

	// Routes principales
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/login", handleLogin)
	mux.HandleFunc("/play", handlePlay)
	mux.HandleFunc("/reset", handleReset)

	// Fichiers statiques
	mux.Handle("/assets/style/", http.StripPrefix("/assets/style/", http.FileServer(http.Dir("assets/style"))))
	mux.Handle("/assets/music/", http.StripPrefix("/assets/music/", http.FileServer(http.Dir("assets/music"))))
	mux.Handle("/assets/musique/", http.StripPrefix("/assets/musique/", http.FileServer(http.Dir("assets/musique"))))

	addr := ":3000"
	fmt.Println("Serveur lancé sur le port " + addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

// === HANDLERS ===

// Page du jeu
func handleIndex(w http.ResponseWriter, r *http.Request) {
	game.Mutex.Lock()
	defer game.Mutex.Unlock()

	data := map[string]interface{}{
		"Board":         game.Board,
		"CurrentPlayer": game.CurrentPlayer,
		"Message":       game.Message,
		"Over":          game.Over,
	}

	if err := tmplIndex.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Page de connexion / inscription
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := tmplLogin.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Jouer un coup
func handlePlay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	colStr := r.FormValue("col")
	col, err := strconv.Atoi(colStr)
	if err != nil || col < 0 || col >= columns {
		http.Error(w, "Colonne invalide", http.StatusBadRequest)
		return
	}

	game.Mutex.Lock()
	defer game.Mutex.Unlock()

	if game.Over {
		writeJSON(w, game)
		return
	}

	placed := false
	for rIdx := rows - 1; rIdx >= 0; rIdx-- {
		if game.Board[rIdx][col] == 0 {
			game.Board[rIdx][col] = game.CurrentPlayer
			placed = true
			if checkWin(&game.Board, rIdx, col, game.CurrentPlayer) {
				game.Message = fmt.Sprintf("Joueur %d a gagné !", game.CurrentPlayer)
				game.Over = true
				break
			}
			break
		}
	}

	if !placed {
		game.Message = "La colonne est pleine, choisis-en une autre !"
		writeJSON(w, game)
		return
	}

	if !game.Over && isBoardFull(&game.Board) {
		game.Message = "Le tableau est plein !"
		game.Over = true
	}

	if !game.Over {
		if game.CurrentPlayer == 1 {
			game.CurrentPlayer = 2
		} else {
			game.CurrentPlayer = 1
		}
		game.Message = ""
	}

	writeJSON(w, game)
}

// Réinitialiser le plateau
func handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}
	game.Mutex.Lock()
	defer game.Mutex.Unlock()
	game.Board = [rows][columns]int{}
	game.CurrentPlayer = 1
	game.Message = ""
	game.Over = false
	writeJSON(w, game)
}

// === UTILITAIRES ===

func writeJSON(w http.ResponseWriter, g *Game) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(g)
}

func isBoardFull(b *[rows][columns]int) bool {
	for r := 0; r < rows; r++ {
		for c := 0; c < columns; c++ {
			if b[r][c] == 0 {
				return false
			}
		}
	}
	return true
}

func checkWin(b *[rows][columns]int, row, col, player int) bool {
	dirs := [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}
	for _, d := range dirs {
		dr, dc := d[0], d[1]
		count := 1
		rn, cn := row+dr, col+dc
		for inBounds(rn, cn) && b[rn][cn] == player {
			count++
			rn += dr
			cn += dc
		}
		rn, cn = row-dr, col-dc
		for inBounds(rn, cn) && b[rn][cn] == player {
			count++
			rn -= dr
			cn -= dc
		}
		if count >= 4 {
			return true
		}
	}
	return false
}

func inBounds(r, c int) bool {
	return r >= 0 && r < rows && c >= 0 && c < columns
}
