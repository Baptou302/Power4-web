package models

import (
	"database/sql"
	"errors"
	"fmt"
	"power4/src/logger"
)

// GetTitle retourne le titre selon le niveau
func GetTitle(level int) string {
	if level >= 20 {
		return "Grand Maître"
	} else if level >= 15 {
		return "Expérimenté"
	} else if level >= 10 {
		return "Amateur"
	} else if level >= 5 {
		return "Débutant"
	}
	return "Novice"
}

// CalculateLevel calcule le niveau à partir de l'XP
// Chaque niveau nécessite 5 victoires (5 XP par victoire)
func CalculateLevel(xp int) int {
	// Chaque niveau nécessite 5 XP (1 victoire = 5 XP)
	level := xp / 5
	// Niveau maximum : 20
	if level > 20 {
		level = 20
	}
	return level
}

// GetXP récupère l'XP et le niveau d'un utilisateur
func GetXP(username string) (int, int, error) {
	if DB == nil {
		return 0, 0, errors.New("base de données non initialisée")
	}

	var xp sql.NullInt64
	err := DB.QueryRow("SELECT COALESCE(xp, 0) FROM users WHERE username = ?", username).Scan(&xp)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, 0, errors.New("utilisateur non trouvé")
		}
		return 0, 0, fmt.Errorf("erreur lors de la récupération de l'XP: %v", err)
	}

	xpValue := 0
	if xp.Valid {
		xpValue = int(xp.Int64)
	}

	level := CalculateLevel(xpValue)
	return xpValue, level, nil
}

// AddXP ajoute de l'XP à un utilisateur
func AddXP(username string, amount int) error {
	if DB == nil {
		return errors.New("base de données non initialisée")
	}

	if amount <= 0 {
		return errors.New("le montant d'XP doit être positif")
	}

	// Récupérer l'XP actuel
	currentXP, _, err := GetXP(username)
	if err != nil {
		return err
	}

	// Calculer le nouveau niveau
	newXP := currentXP + amount

	// Bloquer l'XP à 100 maximum
	maxXP := 100
	if newXP > maxXP {
		newXP = maxXP
	}

	newLevel := CalculateLevel(newXP)
	oldLevel := CalculateLevel(currentXP)
	levelUp := newLevel > oldLevel

	// Mettre à jour l'XP dans la base de données
	_, err = DB.Exec("UPDATE users SET xp = ? WHERE username = ?", newXP, username)
	if err != nil {
		return fmt.Errorf("erreur lors de la mise à jour de l'XP: %v", err)
	}

	// Logger le gain d'XP dans le bot Discord
	logger.LogXP(username, amount, currentXP, newXP, oldLevel, newLevel, levelUp)

	return nil
}

// InitializeXPColumns ajoute les colonnes XP et level si elles n'existent pas
func InitializeXPColumns() error {
	if DB == nil {
		return errors.New("base de données non initialisée")
	}

	// Vérifier si la colonne xp existe
	var xpExists int
	err := DB.QueryRow(`
		SELECT COUNT(*) 
		FROM information_schema.COLUMNS 
		WHERE TABLE_SCHEMA = DATABASE() 
		AND TABLE_NAME = 'users' 
		AND COLUMN_NAME = 'xp'
	`).Scan(&xpExists)

	if err == nil && xpExists == 0 {
		// Ajouter la colonne xp
		_, err = DB.Exec("ALTER TABLE users ADD COLUMN xp INT NOT NULL DEFAULT 0")
		if err != nil {
			return fmt.Errorf("erreur lors de l'ajout de la colonne xp: %v", err)
		}
	}

	return nil
}

// SetXP définit l'XP d'un utilisateur (pour l'admin)
func SetXP(username string, xp int) error {
	if DB == nil {
		return errors.New("base de données non initialisée")
	}

	if xp < 0 {
		return errors.New("l'XP ne peut pas être négatif")
	}

	// Limiter l'XP au maximum (niveau 20 = 100 XP)
	maxXP := 100
	if xp > maxXP {
		xp = maxXP
	}

	_, err := DB.Exec("UPDATE users SET xp = ? WHERE username = ?", xp, username)
	if err != nil {
		return fmt.Errorf("erreur lors de la mise à jour de l'XP: %v", err)
	}

	return nil
}
