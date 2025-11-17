package middleware

import (
	"net/http"
	"power4/src/auth"
)

// RequireAuth est un middleware qui vérifie si l'utilisateur est authentifié
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := auth.GetUsernameFromRequest(r)
		if username == "" {
			// Pour les requêtes API (JSON), retourner une erreur JSON au lieu d'une redirection
			if r.Header.Get("Content-Type") == "application/json" || r.Method == "POST" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error": "Non authentifié"}`))
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAdmin est un middleware qui vérifie si l'utilisateur est administrateur
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if role := auth.GetRoleFromRequest(r); role != "admin" {
			http.Error(w, "Accès refusé : droits insuffisants", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// JSONContentType définit le Content-Type sur application/json
func JSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
