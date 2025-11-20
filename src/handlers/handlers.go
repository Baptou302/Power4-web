package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"
	"power4/src/auth"
	"power4/src/models"
	"power4/src/paypal"
	"strconv"
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

		case "update_xp":
			username := r.FormValue("username")
			xpStr := r.FormValue("xp")

			if username == "" {
				http.Error(w, "Nom d'utilisateur manquant", http.StatusBadRequest)
				return
			}

			xp, err := strconv.Atoi(xpStr)
			if err != nil {
				http.Error(w, "XP invalide: "+err.Error(), http.StatusBadRequest)
				return
			}

			err = models.SetXP(username, xp)
			if err != nil {
				http.Error(w, "Erreur lors de la mise à jour de l'XP: "+err.Error(), http.StatusInternalServerError)
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

	tmpl, err := template.ParseFiles(filepath.Join("templates", "mode-selection", "mode-selection.html"))
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

// HandleHistory affiche la page d'histoire du Puissance 4
func HandleHistory(w http.ResponseWriter, r *http.Request) {
	// Vérifier si l'utilisateur est connecté
	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		// Si non connecté, rediriger vers la page de connexion
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Afficher la page d'histoire
	http.ServeFile(w, r, filepath.Join("templates", "history", "history.html"))
}

// HandleShop affiche la page de boutique
func HandleShop(w http.ResponseWriter, r *http.Request) {
	// Vérifier si l'utilisateur est connecté
	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		// Si non connecté, rediriger vers la page de connexion
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Afficher la page de boutique
	http.ServeFile(w, r, filepath.Join("templates", "shop", "shop.html"))
}

// HandlePayPalCreateOrder crée une commande PayPal
func HandlePayPalCreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	var req paypal.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Erreur lors de la lecture de la requête: "+err.Error(), http.StatusBadRequest)
		return
	}

	order, err := paypal.CreateOrder(req.Items, req.Total)
	if err != nil {
		http.Error(w, "Erreur lors de la création de la commande: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

// HandlePayPalCaptureOrder capture un paiement PayPal
func HandlePayPalCaptureOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrderID string `json:"orderID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Erreur lors de la lecture de la requête: "+err.Error(), http.StatusBadRequest)
		return
	}

	capture, err := paypal.CaptureOrder(req.OrderID)
	if err != nil {
		http.Error(w, "Erreur lors de la capture du paiement: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(capture)
}

// HandlePayPalConfig renvoie la configuration PayPal (client ID uniquement)
func HandlePayPalConfig(w http.ResponseWriter, r *http.Request) {
	clientID := paypal.GetClientID()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"clientId": clientID,
	})
}

// HandleStats renvoie les statistiques de l'utilisateur (XP, niveau, titre)
func HandleStats(w http.ResponseWriter, r *http.Request) {
	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		http.Error(w, "Non connecté", http.StatusUnauthorized)
		return
	}

	xp, level, err := models.GetXP(username)
	if err != nil {
		http.Error(w, "Erreur lors de la récupération des stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	title := models.GetTitle(level)

	// Calculer l'XP nécessaire pour le prochain niveau
	xpForNextLevel := (level + 1) * 5
	if level >= 20 {
		xpForNextLevel = xp // Niveau max atteint
	}
	xpNeeded := xpForNextLevel - xp
	if xpNeeded < 0 {
		xpNeeded = 0
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"xp":               xp,
		"level":            level,
		"title":            title,
		"xpForNextLevel":   xpForNextLevel,
		"xpNeeded":         xpNeeded,
		"xpInCurrentLevel": xp % 5, // XP dans le niveau actuel (0-4)
	})
}

// HandleProfile gère la page de profil avec les statistiques
func HandleProfile(w http.ResponseWriter, r *http.Request) {
	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Récupérer les statistiques de l'utilisateur
	stats, err := models.GetUserStats(username)
	if err != nil {
		http.Error(w, "Erreur lors de la récupération des statistiques: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Récupérer aussi les stats XP
	xp, level, err := models.GetXP(username)
	if err != nil {
		http.Error(w, "Erreur lors de la récupération de l'XP: "+err.Error(), http.StatusInternalServerError)
		return
	}

	title := models.GetTitle(level)

	// Calculer les pourcentages pour l'affichage
	var lossRate, drawRate float64
	if stats.TotalGames > 0 {
		lossRate = float64(stats.Losses) / float64(stats.TotalGames) * 100
		drawRate = float64(stats.Draws) / float64(stats.TotalGames) * 100
	}

	// Calculer l'initiale pour l'avatar
	avatarInitial := ""
	if len(username) > 0 {
		runes := []rune(username)
		if len(runes) > 0 {
			firstChar := runes[0]
			// Convertir en majuscule si c'est une minuscule
			if firstChar >= 'a' && firstChar <= 'z' {
				firstChar = firstChar - 32
			}
			avatarInitial = string(firstChar)
		}
	}

	// Sérialiser l'historique XP en JSON pour le graphique
	xpHistoryJSON, _ := json.Marshal(stats.XPHistory)

	// Préparer les données pour le template
	data := map[string]interface{}{
		"Username":      username,
		"AvatarInitial": avatarInitial,
		"Title":         title,
		"Level":         level,
		"XP":            xp,
		"Stats":         stats,
		"LossRate":      lossRate,
		"DrawRate":      drawRate,
		"XPHistoryJSON": template.JS(string(xpHistoryJSON)), // Utiliser template.JS pour éviter l'échappement
	}

	// Ajouter une fonction helper pour le JSON dans le template
	funcMap := template.FuncMap{
		"json": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
	}

	// Charger le template avec les fonctions
	tmpl, err := template.New("profile.html").Funcs(funcMap).ParseFiles(filepath.Join("templates", "profile", "profile.html"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}

// HandleLeaderboard gère la page du leaderboard
func HandleLeaderboard(w http.ResponseWriter, r *http.Request) {
	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Récupérer le classement (top 50)
	leaderboard, err := models.GetLeaderboard(50)
	if err != nil {
		http.Error(w, "Erreur lors de la récupération du leaderboard: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Trouver le rang de l'utilisateur actuel
	currentUserRank := -1
	for i, entry := range leaderboard {
		if entry.Username == username {
			currentUserRank = i + 1
			break
		}
	}

	// Si l'utilisateur n'est pas dans le top 50, récupérer son rang
	if currentUserRank == -1 {
		allLeaderboard, err := models.GetLeaderboard(1000) // Récupérer tous les utilisateurs
		if err == nil {
			for i, entry := range allLeaderboard {
				if entry.Username == username {
					currentUserRank = i + 1
					break
				}
			}
		}
	}

	// Récupérer les stats de l'utilisateur actuel
	currentUserStats, _ := models.GetUserStats(username)
	currentUserXP, currentUserLevel, _ := models.GetXP(username)
	currentUserTitle := models.GetTitle(currentUserLevel)

	// Préparer les données pour le template
	data := map[string]interface{}{
		"Username":        username,
		"Leaderboard":    leaderboard,
		"CurrentUserRank": currentUserRank,
		"CurrentUserStats": currentUserStats,
		"CurrentUserXP":   currentUserXP,
		"CurrentUserLevel": currentUserLevel,
		"CurrentUserTitle": currentUserTitle,
	}

	// Charger le template
	tmpl, err := template.ParseFiles(filepath.Join("templates", "leaderboard", "leaderboard.html"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}

// HandleSupport gère la page de support pour les utilisateurs
func HandleSupport(w http.ResponseWriter, r *http.Request) {
	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Récupérer les tickets de l'utilisateur
	tickets, err := models.GetUserTickets(username)
	if err != nil {
		http.Error(w, "Erreur lors de la récupération des tickets: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Charger le template
	tmpl, err := template.ParseFiles(filepath.Join("templates", "support", "support.html"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Username": username,
		"Tickets":  tickets,
	}

	tmpl.Execute(w, data)
}

// HandleCreateTicket gère la création d'un nouveau ticket
func HandleCreateTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		http.Error(w, "Non connecté", http.StatusUnauthorized)
		return
	}

	subject := r.FormValue("subject")
	message := r.FormValue("message")

	if subject == "" || message == "" {
		http.Error(w, "Le sujet et le message sont requis", http.StatusBadRequest)
		return
	}

	ticket, err := models.CreateTicket(username, subject, message)
	if err != nil {
		http.Error(w, "Erreur lors de la création du ticket: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"ticket":  ticket,
	})
}

// HandleAdminTickets gère la page admin pour les tickets
func HandleAdminTickets(w http.ResponseWriter, r *http.Request) {
	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Vérifier que l'utilisateur est admin
	role := auth.GetRoleFromRequest(r)
	if role != "admin" {
		http.Error(w, "Accès refusé", http.StatusForbidden)
		return
	}

	// Récupérer le filtre de statut
	statusFilter := r.URL.Query().Get("status")
	if statusFilter == "" {
		statusFilter = "all"
	}

	// Récupérer tous les tickets
	tickets, err := models.GetAllTickets(statusFilter)
	if err != nil {
		http.Error(w, "Erreur lors de la récupération des tickets: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Récupérer les statistiques
	stats, err := models.GetTicketStats()
	if err != nil {
		stats = make(map[string]int)
	}

	// Charger le template
	tmpl, err := template.ParseFiles(filepath.Join("templates", "admin", "tickets.html"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Username":    username,
		"Tickets":      tickets,
		"Stats":        stats,
		"StatusFilter": statusFilter,
	}

	tmpl.Execute(w, data)
}

// HandleTicketResponse gère la réponse d'un admin à un ticket
func HandleTicketResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		http.Error(w, "Non connecté", http.StatusUnauthorized)
		return
	}

	// Vérifier que l'utilisateur est admin
	role := auth.GetRoleFromRequest(r)
	if role != "admin" {
		http.Error(w, "Accès refusé", http.StatusForbidden)
		return
	}

	ticketIDStr := r.FormValue("ticket_id")
	response := r.FormValue("response")

	if ticketIDStr == "" || response == "" {
		http.Error(w, "L'ID du ticket et la réponse sont requis", http.StatusBadRequest)
		return
	}

	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		http.Error(w, "ID de ticket invalide", http.StatusBadRequest)
		return
	}

	err = models.RespondToTicket(ticketID, username, response)
	if err != nil {
		http.Error(w, "Erreur lors de l'ajout de la réponse: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// HandleDeleteTicket gère la suppression d'un ticket (admin uniquement)
func HandleDeleteTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		http.Error(w, "Non connecté", http.StatusUnauthorized)
		return
	}

	// Vérifier que l'utilisateur est admin
	role := auth.GetRoleFromRequest(r)
	if role != "admin" {
		http.Error(w, "Accès refusé", http.StatusForbidden)
		return
	}

	ticketIDStr := r.FormValue("ticket_id")
	if ticketIDStr == "" {
		http.Error(w, "L'ID du ticket est requis", http.StatusBadRequest)
		return
	}

	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		http.Error(w, "ID de ticket invalide", http.StatusBadRequest)
		return
	}

	err = models.DeleteTicket(ticketID)
	if err != nil {
		http.Error(w, "Erreur lors de la suppression du ticket: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// HandleUserDeleteTicket gère la suppression d'un ticket par l'utilisateur
func HandleUserDeleteTicket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		http.Error(w, "Non connecté", http.StatusUnauthorized)
		return
	}

	role := auth.GetRoleFromRequest(r)
	isAdmin := role == "admin"

	ticketIDStr := r.FormValue("ticket_id")
	if ticketIDStr == "" {
		http.Error(w, "L'ID du ticket est requis", http.StatusBadRequest)
		return
	}

	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		http.Error(w, "ID de ticket invalide", http.StatusBadRequest)
		return
	}

	// Vérifier si l'utilisateur peut supprimer ce ticket
	canDelete, err := models.CanUserDeleteTicket(ticketID, username, isAdmin)
	if err != nil {
		http.Error(w, "Erreur lors de la vérification: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !canDelete {
		http.Error(w, "Vous n'avez pas la permission de supprimer ce ticket", http.StatusForbidden)
		return
	}

	err = models.DeleteTicket(ticketID)
	if err != nil {
		http.Error(w, "Erreur lors de la suppression du ticket: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// HandleUpdateTicketStatus gère la mise à jour du statut d'un ticket
func HandleUpdateTicketStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	username := auth.GetUsernameFromRequest(r)
	if username == "" {
		http.Error(w, "Non connecté", http.StatusUnauthorized)
		return
	}

	// Vérifier que l'utilisateur est admin
	role := auth.GetRoleFromRequest(r)
	if role != "admin" {
		http.Error(w, "Accès refusé", http.StatusForbidden)
		return
	}

	ticketIDStr := r.FormValue("ticket_id")
	status := r.FormValue("status")

	if ticketIDStr == "" || status == "" {
		http.Error(w, "L'ID du ticket et le statut sont requis", http.StatusBadRequest)
		return
	}

	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		http.Error(w, "ID de ticket invalide", http.StatusBadRequest)
		return
	}

	err = models.UpdateTicketStatus(ticketID, status)
	if err != nil {
		http.Error(w, "Erreur lors de la mise à jour du statut: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
