package game

import (
	"database/sql"
	"errors"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func ConnectDB() {
	var err error
	DB, err = sql.Open("sqlite", "./power4web.db")
	if err != nil {
		log.Fatal("Erreur ouverture DB:", err)
	}

	createTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL
	);
	`
	_, err = DB.Exec(createTable)
	if err != nil {
		log.Fatal("Erreur création table users:", err)
	}
}

func RegisterUser(username, password string) error {
	if DB == nil {
		return errors.New("DB non initialisée")
	}
	_, err := DB.Exec("INSERT INTO users (username, password) VALUES (?, ?)", username, password)
	if err != nil {
		return errors.New("Nom d'utilisateur déjà existant")
	}
	return nil
}

func ValidateUser(username, password string) error {
	if DB == nil {
		return errors.New("DB non initialisée")
	}
	row := DB.QueryRow("SELECT password FROM users WHERE username = ?", username)
	var stored string
	err := row.Scan(&stored)
	if err != nil {
		return errors.New("Utilisateur non trouvé")
	}
	if stored != password {
		return errors.New("Mot de passe incorrect")
	}
	return nil
}
