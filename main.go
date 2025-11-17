package main

import (
	"log"
	"net/http"
	"os"
	"power4/src/auth"
	"power4/src/models"
	"power4/src/server"
	"power4/src/utils"
)

func main() {
	httpsAddr := utils.GetEnv("HTTPS_PORT", "3443")
	certFile := "server.crt"
	keyFile := "server.key"
	if err := models.ConnectDB(); err != nil {
		log.Fatalf("Erreur de connexion à la base de données: %v", err)
	}
	auth.InitSessionStore()
	handler := server.SetupRoutes()
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		log.Printf("Génération des certificats auto-signés...")
		if err := utils.GenerateSelfSignedCert(certFile, keyFile); err != nil {
			log.Fatalf("Erreur lors de la génération du certificat: %v", err)
		}
	}
	log.Printf("Serveur HTTPS démarré sur https://localhost:%s\n", httpsAddr)
	if err := http.ListenAndServeTLS(":"+httpsAddr, certFile, keyFile, handler); err != nil {
		log.Fatalf("Erreur lors du démarrage du serveur: %v", err)
	}
}
