package models

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

// Session représente une session utilisateur
type Session struct {
	Username string
	Role     string
	Expires  time.Time
}

var (
	sessions      = make(map[string]Session)
	sessionsMutex sync.RWMutex
	sessionDuration = 24 * time.Hour
)

// generateSessionToken génère un token de session sécurisé
func generateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateSession crée une nouvelle session pour l'utilisateur
func CreateSession(w http.ResponseWriter, username, role string) (string, error) {
	token, err := generateSessionToken()
	if err != nil {
		return "", err
	}

	expires := time.Now().Add(sessionDuration)
	session := Session{
		Username: username,
		Role:     role,
		Expires:  expires,
	}

	sessionsMutex.Lock()
	sessions[token] = session
	sessionsMutex.Unlock()

	// Définir le cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Expires:  expires,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // Mettre à true en production avec HTTPS
	})

	return token, nil
}

// GetSession récupère une session à partir du token
func GetSession(r *http.Request) (Session, bool) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return Session{}, false
	}

	token := cookie.Value

	sessionsMutex.RLock()
	session, exists := sessions[token]
	sessionsMutex.RUnlock()

	// Vérifier si la session a expiré
	if exists && time.Now().After(session.Expires) {
		sessionsMutex.Lock()
		delete(sessions, token)
		sessionsMutex.Unlock()
		return Session{}, false
	}

	return session, exists
}

// DeleteSession supprime une session
func DeleteSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return
	}

	token := cookie.Value

	sessionsMutex.Lock()
	delete(sessions, token)
	sessionsMutex.Unlock()

	// Supprimer le cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
	})
}

// GetUsernameFromRequest récupère le nom d'utilisateur à partir de la requête
func GetUsernameFromRequest(r *http.Request) string {
	session, exists := GetSession(r)
	if !exists {
		return ""
	}
	return session.Username
}

// GetRoleFromRequest récupère le rôle de l'utilisateur à partir de la requête
func GetRoleFromRequest(r *http.Request) string {
	session, exists := GetSession(r)
	if !exists {
		return ""
	}
	return session.Role
}

// RequireAuth est un middleware qui vérifie si l'utilisateur est authentifié
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if GetUsernameFromRequest(r) == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// RequireAdmin est un middleware qui vérifie si l'utilisateur est administrateur
func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if GetRoleFromRequest(r) != "admin" {
			http.Error(w, "Accès refusé : droits insuffisants", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// CleanupSessions nettoie les sessions expirées
func CleanupSessions() {
	for {
		time.Sleep(time.Hour)
		now := time.Now()
		
		sessionsMutex.Lock()
		for token, session := range sessions {
			if now.After(session.Expires) {
				delete(sessions, token)
			}
		}
		sessionsMutex.Unlock()
	}
}
