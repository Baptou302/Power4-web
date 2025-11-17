package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"power4/src/auth"
	"power4/src/game"
	"power4/src/handlers"
	"power4/src/middleware"
	"power4/src/models"
	"time"
)

func main() {
	// Configuration
	httpsAddr := getEnv("HTTPS_PORT", "3443")

	// Initialisation de la base de données
	if err := models.ConnectDB(); err != nil {
		log.Fatalf("Erreur de connexion à la base de données: %v", err)
	}

	// Initialisation du store de session
	auth.InitSessionStore()

	// Configuration des routes
	handler := setupRoutes()

	// Générer les certificats SSL si nécessaire
	certFile := "server.crt"
	keyFile := "server.key"
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		log.Printf("Génération des certificats auto-signés...")
		if err := generateSelfSignedCert(certFile, keyFile); err != nil {
			log.Fatalf("Erreur lors de la génération du certificat: %v", err)
		}
	}

	// Démarrer le serveur HTTPS
	log.Printf("Serveur HTTPS démarré sur https://localhost:%s\n", httpsAddr)
	if err := http.ListenAndServeTLS(":"+httpsAddr, certFile, keyFile, handler); err != nil {
		log.Fatalf("Erreur lors du démarrage du serveur: %v", err)
	}
}

// getEnv récupère une variable d'environnement ou retourne une valeur par défaut
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

// generateSelfSignedCert génère un certificat auto-signé
func generateSelfSignedCert(certFile, keyFile string) error {
	// Créer une clé privée RSA
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("échec de génération de la clé privée: %v", err)
	}

	// Créer un template de certificat
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Power4 Web"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // Valide 1 an
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Ajouter les noms d'hôtes
	hostname, _ := os.Hostname()
	template.DNSNames = append(template.DNSNames, "localhost", hostname, "127.0.0.1", "::1")
	template.IPAddresses = []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}

	// Créer le certificat auto-signé
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("échec de création du certificat: %v", err)
	}

	// Écrire le fichier de certificat
	certOut, err := os.Create(certFile)
	if err != nil {
		return fmt.Errorf("échec d'ouverture %s: %v", certFile, err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		certOut.Close()
		return fmt.Errorf("échec d'écriture du certificat: %v", err)
	}
	certOut.Close()

	// Écrire la clé privée
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("échec d'ouverture %s: %v", keyFile, err)
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		keyOut.Close()
		return fmt.Errorf("échec de sérialisation de la clé privée: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		keyOut.Close()
		return fmt.Errorf("échec d'écriture de la clé privée: %v", err)
	}
	keyOut.Close()

	return nil
}

// setupRoutes configure toutes les routes de l'application
func setupRoutes() http.Handler {
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
