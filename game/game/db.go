package game

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB

// ConnectDB initialise la connexion à MySQL
func ConnectDB() {
	var err error
	// Remplace "root:password@tcp(localhost:3306)/power4web" par tes infos
	DB, err = sql.Open("mysql", "power4user:motdepasse@tcp(127.0.0.1:3306)/power4web")
	if err != nil {
		log.Fatal("❌ Impossible de se connecter à MySQL:", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal("❌ Impossible de ping MySQL:", err)
	}

	fmt.Println("✅ Connexion MySQL OK")

	// Création table users si elle n'existe pas
	createTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(255) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`
	_, err = DB.Exec(createTable)
	if err != nil {
		log.Fatal("❌ Erreur création table users:", err)
	}
}

// IsUsernameTaken vérifie si le pseudo existe déjà
func IsUsernameTaken(username string) (bool, error) {
	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", username).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// RegisterUser enregistre un nouvel utilisateur
func RegisterUser(username, password string) error {
	if DB == nil {
		return errors.New("DB non initialisée")
	}

	taken, err := IsUsernameTaken(username)
	if err != nil {
		return err
	}
	if taken {
		return errors.New("Ce pseudo est déjà utilisé")
	}

	_, err = DB.Exec("INSERT INTO users (username, password) VALUES (?, ?)", username, password)
	if err != nil {
		fmt.Println("Erreur MySQL:", err)
		return errors.New("Impossible d’enregistrer l’utilisateur")
	}
	return nil
}

// ValidateUser vérifie les identifiants
func ValidateUser(username, password string) error {
	if DB == nil {
		return fmt.Errorf("DB non initialisée")
	}

	var hashedPassword string
	row := DB.QueryRow("SELECT password FROM users WHERE username = ?", username)
	err := row.Scan(&hashedPassword)
	if err != nil {
		return fmt.Errorf("Utilisateur non trouvé")
	}

	// Compare le mot de passe saisi avec le hash stocké
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return fmt.Errorf("Mot de passe incorrect")
	}

	return nil
}
