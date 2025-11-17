package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"
	"power4/src/models"
	"power4/src/auth"
)

// HandleLogin gère la page de connexion
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		username := r.FormValue("username")
		password := r.FormValue("password")

		// Valider les identifiants
		err := models.ValidateUser(username, password)
		if err != nil {
			http.Error(w, "Identifiants invalides", http.StatusUnauthorized)
			return
		}

		// Récupérer le rôle de l'utilisateur
		user, err := models.GetUserByUsername(username)
		if err != nil {
			http.Error(w, "Erreur lors de la récupération du profil", http.StatusInternalServerError)
			return
		}

		// Créer une session
		err = auth.CreateSession(w, username, user.Role)
		if err != nil {
			http.Error(w, "Erreur lors de la création de la session", http.StatusInternalServerError)
			return
		}

		// Rediriger vers la page de sélection du mode de jeu
		http.Redirect(w, r, "/mode-selection", http.StatusSeeOther)
		return
	}

	// Afficher le formulaire de connexion
	tmpl, err := template.ParseFiles(filepath.Join("templates", "auth", "login.html"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// HandleLogout déconnecte l'utilisateur
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	auth.DeleteSession(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// HandleRegister gère l'inscription d'un nouvel utilisateur
func HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		username := r.FormValue("username")
		password := r.FormValue("password")

		// Vérifier si le nom d'utilisateur est disponible
		taken, err := models.IsUsernameTaken(username)
		if err != nil {
			http.Error(w, "Erreur lors de la vérification du nom d'utilisateur", http.StatusInternalServerError)
			return
		}
		if taken {
			http.Error(w, "Ce nom d'utilisateur est déjà pris", http.StatusBadRequest)
			return
		}

		// Créer le nouvel utilisateur
		err = models.RegisterUser(username, password)
		if err != nil {
			http.Error(w, "Erreur lors de la création du compte: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Récupérer l'utilisateur pour obtenir son rôle
		user, err := models.GetUserByUsername(username)
		if err != nil {
			http.Error(w, "Erreur lors de la récupération du rôle", http.StatusInternalServerError)
			return
		}

		// Créer une nouvelle session pour l'utilisateur avec son rôle
		err = auth.CreateSession(w, username, user.Role)
		if err != nil {
			http.Error(w, "Compte créé mais erreur de connexion: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/mode-selection", http.StatusSeeOther)
		return
	}

	// Afficher le formulaire d'inscription
	tmpl, err := template.ParseFiles(filepath.Join("templates", "auth", "register.html"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

// HandleWhoami renvoie les informations de l'utilisateur connecté
func HandleWhoami(w http.ResponseWriter, r *http.Request) {
	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		http.Error(w, "Non connecté", http.StatusUnauthorized)
		return
	}

	role := auth.GetRoleFromRequest(r)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"username": username,
		"role":     role,
	})
}

// HandleAdmin gère le panneau d'administration
func HandleAdmin(w http.ResponseWriter, r *http.Request) {
	// Vérifier le rôle admin
	if auth.GetRoleFromRequest(r) != "admin" {
		http.Error(w, "Accès refusé", http.StatusForbidden)
		return
	}

	// Traitement des actions (POST)
	if r.Method == http.MethodPost {
		r.ParseForm()
		action := r.FormValue("action")

		switch action {
		case "add":
			username := r.FormValue("username")
			password := r.FormValue("password")
			role := r.FormValue("role")

			// Créer l'utilisateur
			err := models.RegisterUser(username, password)
			if err != nil {
				http.Error(w, "Erreur lors de la création de l'utilisateur: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// Mettre à jour le rôle si nécessaire
			if role != "user" {
				_, err = models.DB.Exec("UPDATE users SET role = ? WHERE username = ?", role, username)
				if err != nil {
					http.Error(w, "Erreur lors de la mise à jour du rôle: "+err.Error(), http.StatusInternalServerError)
					return
				}
			}

		case "edit":
			id := r.FormValue("id")
			username := r.FormValue("username")
			role := r.FormValue("role")

			// Vérifier qu'on ne modifie pas le dernier admin
			if role != "admin" {
				count, err := models.CountAdmins()
				if err != nil {
					http.Error(w, "Erreur lors de la vérification des administrateurs", http.StatusInternalServerError)
					return
				}
				if count <= 1 {
					http.Error(w, "Impossible de modifier le rôle : il doit rester au moins un administrateur", http.StatusBadRequest)
					return
				}
			}

			_, err := models.DB.Exec("UPDATE users SET username = ?, role = ? WHERE id = ?", username, role, id)
			if err != nil {
				http.Error(w, "Erreur lors de la mise à jour de l'utilisateur: "+err.Error(), http.StatusInternalServerError)
				return
			}

		case "delete":
			id := r.FormValue("id")

			// Vérifier qu'on ne supprime pas le dernier admin
			var role string
			err := models.DB.QueryRow("SELECT role FROM users WHERE id = ?", id).Scan(&role)
			if err != nil {
				http.Error(w, "Erreur lors de la récupération du rôle", http.StatusInternalServerError)
				return
			}

			if role == "admin" {
				count, err := models.CountAdmins()
				if err != nil {
					http.Error(w, "Erreur lors de la vérification des administrateurs", http.StatusInternalServerError)
					return
				}
				if count <= 1 {
					http.Error(w, "Impossible de supprimer le dernier administrateur", http.StatusBadRequest)
					return
				}
			}

			_, err = models.DB.Exec("DELETE FROM users WHERE id = ?", id)
			if err != nil {
				http.Error(w, "Erreur lors de la suppression de l'utilisateur: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// Rediriger pour éviter la soumission multiple
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	// Récupérer la liste des utilisateurs
	users, err := models.GetAllUsers()
	if err != nil {
		http.Error(w, "Erreur lors de la récupération des utilisateurs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Compter le nombre d'administrateurs
	nbAdmins, err := models.CountAdmins()
	if err != nil {
		http.Error(w, "Erreur lors du comptage des administrateurs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Préparer les données pour le template
	data := struct {
		Users    []models.User
		NbAdmins int
	}{
		Users:    users,
		NbAdmins: nbAdmins,
	}

	// Charger et exécuter le template
	tmpl, err := template.ParseFiles(filepath.Join("templates", "admin", "admin.html"))
	if err != nil {
		http.Error(w, "Erreur lors du chargement du template: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Erreur lors de l'exécution du template: "+err.Error(), http.StatusInternalServerError)
	}
}

// HandleModeSelection affiche la page de sélection du mode de jeu
func HandleModeSelection(w http.ResponseWriter, r *http.Request) {
	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	tmpl, err := template.ParseFiles(filepath.Join("templates", "index", "mode-selection.html"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Récupérer le rôle de l'utilisateur pour afficher ou non le bouton admin
	role := auth.GetRoleFromRequest(r)
	data := struct {
		IsAdmin bool
	}{
		IsAdmin: role == "admin",
	}

	tmpl.Execute(w, data)
}

// HandleIndex est le gestionnaire de la page d'accueil
func HandleIndex(w http.ResponseWriter, r *http.Request) {
	// Vérifier si l'utilisateur est connecté
	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		// Si non connecté, rediriger vers la page de connexion
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Si connecté, afficher la page de jeu
	http.ServeFile(w, r, filepath.Join("templates", "index", "index.html"))
}
