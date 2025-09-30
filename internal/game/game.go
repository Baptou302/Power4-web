package game

import (
	"errors"
	"sync"
)

const (
	Rows = 6
	Cols = 7
)

type Game struct {
	mu            sync.Mutex
	Board         [Rows][Cols]int // 0 = vide, 1 = joueur1, 2 = joueur2
	CurrentPlayer int
	Winner        int
	MoveCount     int
	Finished      bool
}

func New() *Game {
	return &Game{CurrentPlayer: 1}
}

// Play essaie de poser un jeton dans la colonne col (0..6)
// Retourne erreur si colonne invalide, pleine ou si le jeu est fini.
func (g *Game) Play(col int) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Finished {
		return errors.New("jeu terminé")
	}
	if col < 0 || col >= Cols {
		return errors.New("colonne invalide")
	}
	// déposer en partant du bas
	for r := Rows - 1; r >= 0; r-- {
		if g.Board[r][col] == 0 {
			g.Board[r][col] = g.CurrentPlayer
			g.MoveCount++
			if g.checkWin(r, col) {
				g.Winner = g.CurrentPlayer
				g.Finished = true
			} else if g.MoveCount == Rows*Cols {
				g.Finished = true // match nul
			} else {
				if g.CurrentPlayer == 1 {
					g.CurrentPlayer = 2
				} else {
					g.CurrentPlayer = 1
				}
			}
			return nil
		}
	}
	return errors.New("colonne pleine")
}

func (g *Game) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Board = [Rows][Cols]int{}
	g.CurrentPlayer = 1
	g.Winner = 0
	g.MoveCount = 0
	g.Finished = false
}

// checkWin vérifie s'il y a 4 alignés passant par (row,col)
func (g *Game) checkWin(row, col int) bool {
	player := g.Board[row][col]
	if player == 0 {
		return false
	}
	dirs := [][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}} // horizontale, verticale, deux diagonales
	for _, d := range dirs {
		count := 1 + g.countDir(row, col, d[0], d[1], player) + g.countDir(row, col, -d[0], -d[1], player)
		if count >= 4 {
			return true
		}
	}
	return false
}

func (g *Game) countDir(r, c, dr, dc, player int) int {
	cnt := 0
	r += dr
	c += dc
	for r >= 0 && r < Rows && c >= 0 && c < Cols && g.Board[r][c] == player {
		cnt++
		r += dr
		c += dc
	}
	return cnt
}
