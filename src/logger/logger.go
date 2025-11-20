package logger

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"time"
)

var logServerURL = getEnv("LOG_SERVER_URL", "http://localhost:8080/log")

// getEnv récupère une variable d'environnement ou retourne une valeur par défaut
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

// LogEvent envoie un événement au bot Discord
func LogEvent(eventType string, data map[string]interface{}) {
	// Ne pas bloquer si le bot n'est pas disponible
	go func() {
		payload := map[string]interface{}{
			"type": eventType,
		}
		for k, v := range data {
			payload[k] = v
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return
		}

		client := &http.Client{
			Timeout: 2 * time.Second,
		}

		req, err := http.NewRequest("POST", logServerURL, bytes.NewBuffer(jsonData))
		if err != nil {
			return
		}

		req.Header.Set("Content-Type", "application/json")

		_, err = client.Do(req)
		// Ignorer les erreurs silencieusement (bot peut ne pas être démarré)
		if err != nil {
			// Optionnel: décommenter pour debug
			// fmt.Printf("Erreur envoi log au bot: %v\n", err)
		}
	}()
}

// LogLogin envoie un log de connexion
func LogLogin(username, role string) {
	LogEvent("login", map[string]interface{}{
		"username": username,
		"role":     role,
	})
}

// LogGameStart envoie un log de début de partie
func LogGameStart(username, mode, difficulty string, isAIMode bool) {
	LogEvent("game_start", map[string]interface{}{
		"username":   username,
		"mode":       mode,
		"difficulty": difficulty,
		"is_ai_mode": isAIMode,
	})
}

// LogGameWin envoie un log de victoire
func LogGameWin(username, mode, difficulty string, isAIMode bool) {
	LogEvent("game_win", map[string]interface{}{
		"username":   username,
		"mode":       mode,
		"difficulty": difficulty,
		"is_ai_mode": isAIMode,
	})
}

// LogGameLoss envoie un log de défaite
func LogGameLoss(username, mode, difficulty string, isAIMode bool) {
	LogEvent("game_loss", map[string]interface{}{
		"username":   username,
		"mode":       mode,
		"difficulty": difficulty,
		"is_ai_mode": isAIMode,
	})
}

// LogGameDraw envoie un log de match nul
func LogGameDraw(username, mode, difficulty string, isAIMode bool) {
	LogEvent("game_draw", map[string]interface{}{
		"username":   username,
		"mode":       mode,
		"difficulty": difficulty,
		"is_ai_mode": isAIMode,
	})
}

// LogXP envoie un log de gain d'XP
func LogXP(username string, amount int, oldXP int, newXP int, oldLevel int, newLevel int, levelUp bool) {
	LogEvent("xp_gain", map[string]interface{}{
		"username":  username,
		"amount":    amount,
		"old_xp":    oldXP,
		"new_xp":    newXP,
		"old_level": oldLevel,
		"new_level": newLevel,
		"level_up":  levelUp,
	})
}
