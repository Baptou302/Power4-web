package game

import (
	"html/template"
	"net/http"
	"path/filepath"
)

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		err := ValidateUser(username, password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		CreateSession(w, username)
		http.Redirect(w, r, "/mode-selection", http.StatusSeeOther)
		return
	}

	tmpl, _ := template.ParseFiles(filepath.Join("templates", "login.html"))
	tmpl.Execute(w, nil)
}
