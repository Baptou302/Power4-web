package models

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB

// ConnectDB initialise la connexion à la base de données et crée les tables nécessaires
func ConnectDB() error {
	var err error
	DB, err = sql.Open("mysql", "power4user:motdepasse@tcp(127.0.0.1:3306)/power4weboffi")
	if err != nil {
		return fmt.Errorf("impossible de se connecter à MySQL: %v", err)
	}

	err = DB.Ping()
	if err != nil {
		return fmt.Errorf("impossible de pinger MySQL: %v", err)
	}

	fmt.Println("✅ Connexion MySQL établie avec succès")

	// Création de la table des utilisateurs si elle n'existe pas
	createTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(255) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		role VARCHAR(16) NOT NULL DEFAULT 'user',
		created_at TIMESTAMP NULL DEFAULT NULL,
		updated_at TIMESTAMP NULL DEFAULT NULL
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	_, err = DB.Exec(createTable)
	if err != nil {
		return fmt.Errorf("erreur lors de la création de la table users: %v", err)
	}

	// Vérifier et ajouter la colonne role si elle n'existe pas
	var columnExists int
	err = DB.QueryRow(`
		SELECT COUNT(*) 
		FROM information_schema.COLUMNS 
		WHERE TABLE_SCHEMA = DATABASE() 
		AND TABLE_NAME = 'users' 
		AND COLUMN_NAME = 'role'
	`).Scan(&columnExists)

	if err == nil && columnExists == 0 {
		// Ajouter la colonne role si elle n'existe pas
		_, err = DB.Exec("ALTER TABLE users ADD COLUMN role VARCHAR(16) NOT NULL DEFAULT 'user'")
		if err != nil {
			// Ignorer l'erreur si la colonne existe déjà (peut arriver en cas de race condition)
			log.Printf("Note: Impossible d'ajouter la colonne role (peut déjà exister): %v", err)
		}
	}

	// Initialiser les colonnes XP
	if err := InitializeXPColumns(); err != nil {
		log.Printf("Note: Erreur lors de l'initialisation des colonnes XP: %v", err)
	}

	// Créer la table game_history pour stocker l'historique des parties
	createGameHistoryTable := `
	CREATE TABLE IF NOT EXISTS game_history (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(255) NOT NULL,
		result VARCHAR(20) NOT NULL,
		mode VARCHAR(50) NOT NULL,
		difficulty VARCHAR(50) DEFAULT '',
		is_ai_mode BOOLEAN NOT NULL DEFAULT FALSE,
		xp_gained INT DEFAULT 0,
		played_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_username (username),
		INDEX idx_played_at (played_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	_, err = DB.Exec(createGameHistoryTable)
	if err != nil {
		log.Printf("Note: Erreur lors de la création de la table game_history: %v", err)
	}

	// Créer la table tickets pour le système de support
	createTicketsTable := `
	CREATE TABLE IF NOT EXISTS tickets (
		id INT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(255) NOT NULL,
		subject VARCHAR(255) NOT NULL,
		message TEXT NOT NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'open',
		admin_response TEXT DEFAULT NULL,
		admin_username VARCHAR(255) DEFAULT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		INDEX idx_username (username),
		INDEX idx_status (status),
		INDEX idx_created_at (created_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	_, err = DB.Exec(createTicketsTable)
	if err != nil {
		log.Printf("Note: Erreur lors de la création de la table tickets: %v", err)
	}

	// Vérifier si l'utilisateur admin existe, sinon le créer
	adminPassword := "admin123" // À changer en production !
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("erreur lors du hachage du mot de passe admin: %v", err)
	}

	_, err = DB.Exec(`
		INSERT IGNORE INTO users (username, password, role) 
		VALUES (?, ?, 'admin')`, "admin", hashedPassword)
	if err != nil {
		return fmt.Errorf("erreur lors de la création de l'utilisateur admin: %v", err)
	}

	return nil
}

// IsUsernameTaken vérifie si un nom d'utilisateur est déjà pris
func IsUsernameTaken(username string) (bool, error) {
	if DB == nil {
		return false, errors.New("base de données non initialisée")
	}

	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("erreur lors de la vérification du nom d'utilisateur: %v", err)
	}
	return exists, nil
}

func RegisterUser(username, password string) error {
	if DB == nil {
		return errors.New("base de données non initialisée")
	}

	// Vérifier si le nom d'utilisateur est disponible
	taken, err := IsUsernameTaken(username)
	if err != nil {
		return fmt.Errorf("erreur lors de la vérification du nom d'utilisateur: %v", err)
	}
	if taken {
		return errors.New("ce pseudo est déjà utilisé")
	}

	// Hacher le mot de passe avant de le stocker
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("erreur lors du hachage du mot de passe: %v", err)
	}

	// Insérer le nouvel utilisateur
	_, err = DB.Exec(
		"INSERT INTO users (username, password, role) VALUES (?, ?, 'user')",
		username,
		hashedPassword,
	)

	if err != nil {
		return fmt.Errorf("erreur lors de l'enregistrement de l'utilisateur: %v", err)
	}

	return nil
}

// ValidateUser vérifie les identifiants de l'utilisateur
func ValidateUser(username, password string) error {
	if DB == nil {
		return errors.New("base de données non initialisée")
	}

	// Récupérer le mot de passe haché de l'utilisateur
	var hashedPassword string
	row := DB.QueryRow("SELECT password FROM users WHERE username = ?", username)
	err := row.Scan(&hashedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("utilisateur non trouvé")
		}
		return fmt.Errorf("erreur lors de la récupération de l'utilisateur: %v", err)
	}

	// Vérifier le mot de passe
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return errors.New("mot de passe incorrect")
	}

	return nil
}

// User représente un utilisateur dans le système
type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"-"` // Le mot de passe n'est jamais sérialisé en JSON
	Role      string `json:"role"`
	XP        int    `json:"xp"`
	Level     int    `json:"level"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

func GetUserByUsername(username string) (User, error) {
	if DB == nil {
		return User{}, errors.New("base de données non initialisée")
	}

	var user User
	var role sql.NullString

	// Récupérer les champs essentiels avec gestion des valeurs NULL
	err := DB.QueryRow(`
		SELECT id, username, password, role
		FROM users 
		WHERE username = ?`,
		username,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Password,
		&role,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return User{}, errors.New("utilisateur non trouvé")
		}
		return User{}, fmt.Errorf("erreur lors de la récupération de l'utilisateur: %v", err)
	}

	// Gérer le rôle (peut être NULL)
	if role.Valid {
		user.Role = role.String
	} else {
		user.Role = "user" // Valeur par défaut
	}

	// S'assurer qu'un rôle est défini
	if user.Role == "" {
		user.Role = "user"
	}

	return user, nil
}

// GetAllUsers récupère tous les utilisateurs (pour l'administration)
func GetAllUsers() ([]User, error) {
	if DB == nil {
		return nil, errors.New("base de données non initialisée")
	}

	// Récupérer uniquement les colonnes qui existent
	rows, err := DB.Query(`
		SELECT id, username, COALESCE(role, 'user') as role, COALESCE(xp, 0) as xp
		FROM users 
		ORDER BY id DESC`)

	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des utilisateurs: %v", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var role sql.NullString
		var xp sql.NullInt64
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&role,
			&xp,
		)
		if err != nil {
			return nil, fmt.Errorf("erreur lors du scan des utilisateurs: %v", err)
		}

		// Gérer le rôle
		if role.Valid {
			user.Role = role.String
		} else {
			user.Role = "user"
		}

		if user.Role == "" {
			user.Role = "user"
		}

		// Gérer l'XP
		if xp.Valid {
			user.XP = int(xp.Int64)
		} else {
			user.XP = 0
		}

		// Calculer le niveau et le titre
		user.Level = CalculateLevel(user.XP)
		user.Title = GetTitle(user.Level)

		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("erreur lors de l'itération sur les utilisateurs: %v", err)
	}

	return users, nil
}

// UpdateUser met à jour les informations d'un utilisateur
func UpdateUser(id int, username, role string) error {
	if DB == nil {
		return errors.New("base de données non initialisée")
	}

	// Vérifier si l'utilisateur existe
	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("erreur lors de la vérification de l'utilisateur: %v", err)
	}
	if !exists {
		return errors.New("utilisateur non trouvé")
	}

	// Mettre à jour l'utilisateur
	_, err = DB.Exec(
		"UPDATE users SET username = ?, role = ? WHERE id = ?",
		username,
		role,
		id,
	)

	if err != nil {
		return fmt.Errorf("erreur lors de la mise à jour de l'utilisateur: %v", err)
	}

	return nil
}

// DeleteUser supprime un utilisateur par son ID
func DeleteUser(id int) error {
	if DB == nil {
		return errors.New("base de données non initialisée")
	}

	// Vérifier si l'utilisateur existe
	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("erreur lors de la vérification de l'utilisateur: %v", err)
	}
	if !exists {
		return errors.New("utilisateur non trouvé")
	}

	// Supprimer l'utilisateur
	_, err = DB.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression de l'utilisateur: %v", err)
	}

	return nil
}

// CountAdmins compte le nombre d'administrateurs dans la base de données
func CountAdmins() (int, error) {
	if DB == nil {
		return 0, errors.New("base de données non initialisée")
	}

	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("erreur lors du comptage des administrateurs: %v", err)
	}

	return count, nil
}
