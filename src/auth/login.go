package auth

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"power4/src/models"
	"power4/src/logger"
)

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")

		err := models.ValidateUser(username, password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Récupérer le rôle de l'utilisateur
		user, err := models.GetUserByUsername(username)
		if err != nil {
			log.Printf("Erreur GetUserByUsername pour %s: %v", username, err)
			http.Error(w, "Erreur lors de la récupération du rôle: "+err.Error(), http.StatusInternalServerError)
			return
		}

		err = CreateSession(w, username, user.Role)
		if err != nil {
			http.Error(w, "Erreur lors de la création de la session", http.StatusInternalServerError)
			return
		}

		// Logger la connexion
		logger.LogLogin(username, user.Role)

		http.Redirect(w, r, "/mode-selection", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles(filepath.Join("templates", "auth", "login.html"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}
