package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// GameHistory représente une partie dans l'historique
type GameHistory struct {
	ID         int       `json:"id"`
	Username   string    `json:"username"`
	Result     string    `json:"result"` // "win", "loss", "draw"
	Mode       string    `json:"mode"`
	Difficulty string    `json:"difficulty"`
	IsAIMode   bool      `json:"is_ai_mode"`
	XPGained   int       `json:"xp_gained"`
	PlayedAt   time.Time `json:"played_at"`
}

// UserStats représente les statistiques d'un utilisateur
type UserStats struct {
	Username        string         `json:"username"`
	TotalGames      int            `json:"total_games"`
	Wins            int            `json:"wins"`
	Losses          int            `json:"losses"`
	Draws           int            `json:"draws"`
	WinRate         float64        `json:"win_rate"`
	CurrentStreak   int            `json:"current_streak"`
	BestStreak      int            `json:"best_streak"`
	XPHistory       []XPHistory    `json:"xp_history"`
	RecentGames     []GameHistory  `json:"recent_games"`
}

// XPHistory représente l'évolution de l'XP dans le temps
type XPHistory struct {
	Date time.Time `json:"date"`
	XP   int       `json:"xp"`
}

// MarshalJSON personnalise la sérialisation JSON pour formater la date
func (x XPHistory) MarshalJSON() ([]byte, error) {
	type Alias XPHistory
	return json.Marshal(&struct {
		Date string `json:"date"`
		XP   int    `json:"xp"`
	}{
		Date: x.Date.Format("2006-01-02T15:04:05Z07:00"),
		XP:   x.XP,
	})
}

// SaveGameResult enregistre le résultat d'une partie dans l'historique
func SaveGameResult(username, result, mode, difficulty string, isAIMode bool, xpGained int) error {
	if DB == nil {
		return errors.New("base de données non initialisée")
	}

	_, err := DB.Exec(`
		INSERT INTO game_history (username, result, mode, difficulty, is_ai_mode, xp_gained)
		VALUES (?, ?, ?, ?, ?, ?)`,
		username, result, mode, difficulty, isAIMode, xpGained)
	if err != nil {
		return fmt.Errorf("erreur lors de l'enregistrement de la partie: %v", err)
	}

	return nil
}

// GetUserStats récupère les statistiques complètes d'un utilisateur
func GetUserStats(username string) (*UserStats, error) {
	if DB == nil {
		return nil, errors.New("base de données non initialisée")
	}

	stats := &UserStats{
		Username: username,
	}

	// Compter les parties totales, victoires, défaites, matchs nuls
	err := DB.QueryRow(`
		SELECT 
			COUNT(*) as total_games,
			SUM(CASE WHEN result = 'win' THEN 1 ELSE 0 END) as wins,
			SUM(CASE WHEN result = 'loss' THEN 1 ELSE 0 END) as losses,
			SUM(CASE WHEN result = 'draw' THEN 1 ELSE 0 END) as draws
		FROM game_history
		WHERE username = ?`,
		username).Scan(&stats.TotalGames, &stats.Wins, &stats.Losses, &stats.Draws)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("erreur lors de la récupération des stats: %v", err)
	}

	// Calculer le taux de victoire
	if stats.TotalGames > 0 {
		stats.WinRate = float64(stats.Wins) / float64(stats.TotalGames) * 100
	}

	// Calculer le streak actuel et le meilleur streak
	rows, err := DB.Query(`
		SELECT result
		FROM game_history
		WHERE username = ?
		ORDER BY played_at DESC`,
		username)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("erreur lors de la récupération du streak: %v", err)
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil {
			continue
		}
		results = append(results, result)
	}

	// Calculer le streak actuel (dernières victoires consécutives)
	stats.CurrentStreak = 0
	if len(results) > 0 && results[0] == "win" {
		for i := 0; i < len(results) && results[i] == "win"; i++ {
			stats.CurrentStreak++
		}
	}

	// Calculer le meilleur streak
	stats.BestStreak = 0
	currentStreak := 0
	for _, result := range results {
		if result == "win" {
			currentStreak++
			if currentStreak > stats.BestStreak {
				stats.BestStreak = currentStreak
			}
		} else {
			currentStreak = 0
		}
	}

	// Récupérer l'historique des 30 dernières parties
	recentRows, err := DB.Query(`
		SELECT id, username, result, mode, difficulty, is_ai_mode, xp_gained, played_at
		FROM game_history
		WHERE username = ?
		ORDER BY played_at DESC
		LIMIT 30`,
		username)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("erreur lors de la récupération de l'historique: %v", err)
	}
	defer recentRows.Close()

	for recentRows.Next() {
		var game GameHistory
		var playedAtStr string
		err := recentRows.Scan(
			&game.ID,
			&game.Username,
			&game.Result,
			&game.Mode,
			&game.Difficulty,
			&game.IsAIMode,
			&game.XPGained,
			&playedAtStr,
		)
		if err != nil {
			continue
		}

		// Parser la date
		game.PlayedAt, _ = time.Parse("2006-01-02 15:04:05", playedAtStr)
		stats.RecentGames = append(stats.RecentGames, game)
	}

	// Récupérer l'historique de l'XP (évolution sur les 30 derniers jours)
	// On récupère toutes les parties avec XP gagné, triées par date
	xpRows, err := DB.Query(`
		SELECT played_at, xp_gained
		FROM game_history
		WHERE username = ? AND xp_gained > 0
		AND played_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)
		ORDER BY played_at ASC`,
		username)
	if err != nil && err != sql.ErrNoRows {
		// Ne pas bloquer si l'historique XP n'est pas disponible
		stats.XPHistory = []XPHistory{}
	} else {
		defer xpRows.Close()

		var xpHistory []XPHistory
		currentXP, _, _ := GetXP(username)
		cumulativeXP := currentXP

		// Parcourir les résultats en sens inverse pour calculer l'XP à chaque date
		var allEntries []struct {
			date    time.Time
			xpGained int
		}

		for xpRows.Next() {
			var playedAtStr string
			var xpGained int
			if err := xpRows.Scan(&playedAtStr, &xpGained); err != nil {
				continue
			}
			playedAt, _ := time.Parse("2006-01-02 15:04:05", playedAtStr)
			allEntries = append(allEntries, struct {
				date    time.Time
				xpGained int
			}{playedAt, xpGained})
		}

		// Calculer l'XP en remontant dans le temps
		for i := len(allEntries) - 1; i >= 0; i-- {
			entry := allEntries[i]
			cumulativeXP -= entry.xpGained
			xpHistory = append([]XPHistory{{
				Date: entry.date,
				XP:   cumulativeXP,
			}}, xpHistory...)
		}

		// Ajouter le point actuel à la fin
		if len(xpHistory) > 0 {
			xpHistory = append(xpHistory, XPHistory{
				Date: time.Now(),
				XP:   currentXP,
			})
		} else {
			// Si pas d'historique, créer un point avec l'XP actuel
			xpHistory = append(xpHistory, XPHistory{
				Date: time.Now(),
				XP:   currentXP,
			})
		}

		stats.XPHistory = xpHistory
	}

	return stats, nil
}

// GetRecentGames récupère les N dernières parties d'un utilisateur
func GetRecentGames(username string, limit int) ([]GameHistory, error) {
	if DB == nil {
		return nil, errors.New("base de données non initialisée")
	}

	if limit <= 0 {
		limit = 10
	}

	rows, err := DB.Query(`
		SELECT id, username, result, mode, difficulty, is_ai_mode, xp_gained, played_at
		FROM game_history
		WHERE username = ?
		ORDER BY played_at DESC
		LIMIT ?`,
		username, limit)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des parties récentes: %v", err)
	}
	defer rows.Close()

	var games []GameHistory
	for rows.Next() {
		var game GameHistory
		var playedAtStr string
		err := rows.Scan(
			&game.ID,
			&game.Username,
			&game.Result,
			&game.Mode,
			&game.Difficulty,
			&game.IsAIMode,
			&game.XPGained,
			&playedAtStr,
		)
		if err != nil {
			continue
		}

		game.PlayedAt, _ = time.Parse("2006-01-02 15:04:05", playedAtStr)
		games = append(games, game)
	}

	return games, nil
}

// LeaderboardEntry représente une entrée du classement
type LeaderboardEntry struct {
	Rank      int     `json:"rank"`
	Username  string  `json:"username"`
	XP        int     `json:"xp"`
	Level     int     `json:"level"`
	Title     string  `json:"title"`
	Wins      int     `json:"wins"`
	TotalGames int    `json:"total_games"`
	WinRate   float64 `json:"win_rate"`
	BestStreak int    `json:"best_streak"`
}

// GetLeaderboard récupère le classement des meilleurs joueurs
func GetLeaderboard(limit int) ([]LeaderboardEntry, error) {
	if DB == nil {
		return nil, errors.New("base de données non initialisée")
	}

	if limit <= 0 {
		limit = 50 // Par défaut, afficher les 50 meilleurs
	}

	// Récupérer les utilisateurs avec leur XP, triés par XP décroissant
	rows, err := DB.Query(`
		SELECT 
			u.username,
			COALESCE(u.xp, 0) as xp,
			COUNT(gh.id) as total_games,
			COALESCE(SUM(CASE WHEN gh.result = 'win' THEN 1 ELSE 0 END), 0) as wins
		FROM users u
		LEFT JOIN game_history gh ON u.username = gh.username
		GROUP BY u.username, u.xp
		ORDER BY u.xp DESC, wins DESC
		LIMIT ?`,
		limit)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération du leaderboard: %v", err)
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	rank := 1

	for rows.Next() {
		var entry LeaderboardEntry
		var totalGames int64
		var wins int64

		err := rows.Scan(
			&entry.Username,
			&entry.XP,
			&totalGames,
			&wins,
		)
		if err != nil {
			continue
		}

		entry.Rank = rank
		rank++

		entry.TotalGames = int(totalGames)
		entry.Wins = int(wins)

		// Calculer le niveau et le titre
		entry.Level = CalculateLevel(entry.XP)
		entry.Title = GetTitle(entry.Level)

		// Calculer le taux de victoire
		if entry.TotalGames > 0 {
			entry.WinRate = float64(entry.Wins) / float64(entry.TotalGames) * 100
		}

		// Récupérer le meilleur streak
		streakRows, err := DB.Query(`
			SELECT result
			FROM game_history
			WHERE username = ?
			ORDER BY played_at DESC`,
			entry.Username)
		if err == nil {
			var results []string
			for streakRows.Next() {
				var result string
				if err := streakRows.Scan(&result); err != nil {
					continue
				}
				results = append(results, result)
			}
			streakRows.Close() // Fermer immédiatement après utilisation

			// Calculer le meilleur streak
			currentStreak := 0
			entry.BestStreak = 0
			for _, result := range results {
				if result == "win" {
					currentStreak++
					if currentStreak > entry.BestStreak {
						entry.BestStreak = currentStreak
					}
				} else {
					currentStreak = 0
				}
			}
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

