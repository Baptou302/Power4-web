package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

// GenerateSelfSignedCert génère un certificat auto-signé
func GenerateSelfSignedCert(certFile, keyFile string) error {
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

// GetEnv récupère une variable d'environnement ou retourne une valeur par défaut
func GetEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}
