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
	mux.Handle("/profile", middleware.RequireAuth(http.HandlerFunc(handlers.HandleProfile)))
	mux.Handle("/leaderboard", middleware.RequireAuth(http.HandlerFunc(handlers.HandleLeaderboard)))
	mux.Handle("/history", middleware.RequireAuth(http.HandlerFunc(handlers.HandleHistory)))
	mux.Handle("/shop", middleware.RequireAuth(http.HandlerFunc(handlers.HandleShop)))
	mux.Handle("/support", middleware.RequireAuth(http.HandlerFunc(handlers.HandleSupport)))
	mux.Handle("/api/tickets/create", middleware.RequireAuth(http.HandlerFunc(handlers.HandleCreateTicket)))
	mux.Handle("/api/paypal/config", middleware.RequireAuth(http.HandlerFunc(handlers.HandlePayPalConfig)))
	mux.Handle("/api/paypal/create-order", middleware.RequireAuth(http.HandlerFunc(handlers.HandlePayPalCreateOrder)))
	mux.Handle("/api/paypal/capture-order", middleware.RequireAuth(http.HandlerFunc(handlers.HandlePayPalCaptureOrder)))
	mux.Handle("/api/stats", middleware.RequireAuth(http.HandlerFunc(handlers.HandleStats)))
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

	var adminTicketsHandler http.Handler = http.HandlerFunc(handlers.HandleAdminTickets)
	adminTicketsHandler = middleware.RequireAdmin(adminTicketsHandler)
	adminTicketsHandler = middleware.RequireAuth(adminTicketsHandler)
	mux.Handle("/admin/tickets", adminTicketsHandler)

	mux.Handle("/api/tickets/respond", middleware.RequireAuth(middleware.RequireAdmin(http.HandlerFunc(handlers.HandleTicketResponse))))
	mux.Handle("/api/tickets/status", middleware.RequireAuth(middleware.RequireAdmin(http.HandlerFunc(handlers.HandleUpdateTicketStatus))))
	mux.Handle("/api/tickets/delete", middleware.RequireAuth(middleware.RequireAdmin(http.HandlerFunc(handlers.HandleDeleteTicket))))
	mux.Handle("/api/tickets/user-delete", middleware.RequireAuth(http.HandlerFunc(handlers.HandleUserDeleteTicket)))

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
