package main

import (
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

type Game struct {
	Board         [rows][columns]int
	CurrentPlayer int
	Message       string
	Over          bool
	Mutex         sync.Mutex
}

var tmpl = template.Must(template.New("index.html").Funcs(template.FuncMap{
	"seq": func(start, end int) []int {
		s := make([]int, end-start+1)
		for i := range s {
			s[i] = start + i
		}
		return s
	},
}).ParseFiles("internal/templates/index.html"))

var game = &Game{CurrentPlayer: 1}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/play", handlePlay)
	mux.HandleFunc("/reset", handleReset)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("internal/static"))))

	addr := ":3000"
	fmt.Println("Serveur se lance sur le port" + addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	game.Mutex.Lock()
	defer game.Mutex.Unlock()

	data := map[string]interface{}{
		"Board":         game.Board,
		"CurrentPlayer": game.CurrentPlayer,
		"Message":       game.Message,
		"Over":          game.Over,
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handlePlay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	colStr := r.FormValue("col")
	col, err := strconv.Atoi(colStr)
	if err != nil || col < 0 || col >= columns {
		http.Error(w, "colonne invalide", http.StatusBadRequest)
		return
	}

	game.Mutex.Lock()
	defer game.Mutex.Unlock()

	if game.Over {
		http.Redirect(w, r, "/", http.StatusSeeOther)
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
		game.Message = "La collone est pleine choisis en une autre !"
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if !game.Over && isBoardFull(&game.Board) {
		game.Message = "Le tableux de jeux est plein !"
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

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	game.Mutex.Lock()
	defer game.Mutex.Unlock()
	game.Board = [rows][columns]int{}
	game.CurrentPlayer = 1
	game.Message = ""
	game.Over = false
	http.Redirect(w, r, "/", http.StatusSeeOther)
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
