package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"
)

type Session struct {
	Username  string
	Role      string
	ExpiresAt time.Time
}

var (
	sessionStore = make(map[string]Session)
	mu           sync.Mutex
)

func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func CreateSession(w http.ResponseWriter, username string, role string) error {
	token := randomToken(32)

	// Si le rôle n'est pas fourni, utiliser "user" par défaut
	if role == "" {
		role = "user"
	}

	mu.Lock()
	sessionStore[token] = Session{
		Username:  username,
		Role:      role,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Mettre à true en production avec HTTPS
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

// InitSessionStore initialise le store de session (fonction vide pour compatibilité)
func InitSessionStore() {
	// Le store est déjà initialisé avec la déclaration var
	// Cette fonction existe pour compatibilité avec le code existant
}

func GetUsernameFromRequest(r *http.Request) string {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return ""
	}

	mu.Lock()
	defer mu.Unlock()

	session, ok := sessionStore[cookie.Value]
	if !ok || time.Now().After(session.ExpiresAt) {
		return ""
	}
	return session.Username
}

func GetRoleFromRequest(r *http.Request) string {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return ""
	}

	mu.Lock()
	defer mu.Unlock()

	session, ok := sessionStore[cookie.Value]
	if !ok || time.Now().After(session.ExpiresAt) {
		return ""
	}
	return session.Role
}

func DeleteSession(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		mu.Lock()
		delete(sessionStore, cookie.Value)
		mu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// RequireAuth s'utilise pour protéger les handlers exigeant une session.
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := GetUsernameFromRequest(r)
		if username == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// RequireAdmin protège les handlers réservés aux admins.
func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if GetRoleFromRequest(r) != "admin" {
			http.Error(w, "Accès refusé", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
