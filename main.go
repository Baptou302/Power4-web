package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"

	"power4/internal/game"
)

var tpl = template.Must(template.ParseFiles("internal/templates/index.html"))

func main() {
	g := game.New()

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"Board":         g.Board,
			"CurrentPlayer": g.CurrentPlayer,
			"Winner":        g.Winner,
			"Finished":      g.Finished,
			"Columns":       []int{0, 1, 2, 3, 4, 5, 6},
		}
		if err := tpl.Execute(w, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	http.HandleFunc("/play", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		colStr := r.FormValue("col")
		col, err := strconv.Atoi(colStr)
		if err == nil {
			_ = g.Play(col) // on ignore l'erreur côté front (option: passer message d'erreur)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	http.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			g.Reset()
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	log.Println("Listening on :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
