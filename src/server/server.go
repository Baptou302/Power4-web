package server

import (
	"net/http"
	"power4/src/auth"
	"power4/src/game"
	"power4/src/handlers"
	"power4/src/middleware"
)

// SetupRoutes configure toutes les routes de l'application
func SetupRoutes() http.Handler {
	mux := http.NewServeMux()

	// Routes publiques
	mux.HandleFunc("/", handlers.HandleIndex)
	mux.HandleFunc("/login", auth.HandleLogin)
	mux.HandleFunc("/register", handlers.HandleRegister)

	// Routes protégées (avec middleware RequireAuth)
	mux.Handle("/mode-selection", middleware.RequireAuth(http.HandlerFunc(handlers.HandleModeSelection)))
	mux.Handle("/play", middleware.RequireAuth(http.HandlerFunc(game.HandlePlay)))
	mux.Handle("/reset", middleware.RequireAuth(http.HandlerFunc(game.HandleReset)))
	mux.Handle("/new-ai-game", middleware.RequireAuth(http.HandlerFunc(game.HandleNewAIGame)))
	mux.Handle("/whoami", middleware.RequireAuth(http.HandlerFunc(handlers.HandleWhoami)))
	mux.Handle("/logout", middleware.RequireAuth(http.HandlerFunc(handlers.HandleLogout)))

	// Admin routes (protégées par middleware RequireAdmin et RequireAuth)
	var adminHandler http.Handler = http.HandlerFunc(handlers.HandleAdmin)
	adminHandler = middleware.RequireAdmin(adminHandler)
	adminHandler = middleware.RequireAuth(adminHandler)
	mux.Handle("/admin", adminHandler)

	// Fichiers statiques
	fs := http.FileServer(http.Dir("assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))
	mux.Handle("/style/", http.StripPrefix("/style/", http.FileServer(http.Dir("assets/styles"))))

	return addSecurityHeaders(mux)
}

// addSecurityHeaders ajoute des en-têtes de sécurité
func addSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if r.URL.Path != "/" && r.URL.Path != "/login" && r.URL.Path != "/register" {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			w.Header().Set("Pragma", "no-cache")
		}
		next.ServeHTTP(w, r)
	})
}
