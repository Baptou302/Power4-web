package game

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"
)

type Session struct {
	Username  string
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

func CreateSession(w http.ResponseWriter, username string) {
	token := randomToken(32)
	mu.Lock()
	sessionStore[token] = Session{
		Username:  username,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	mu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
	})
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
