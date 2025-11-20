package models

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Ticket représente un ticket de support
type Ticket struct {
	ID            int       `json:"id"`
	Username      string    `json:"username"`
	Subject       string    `json:"subject"`
	Message       string    `json:"message"`
	Status        string    `json:"status"` // open, in_progress, closed, resolved
	AdminResponse string    `json:"admin_response,omitempty"`
	AdminUsername string    `json:"admin_username,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreateTicket crée un nouveau ticket
func CreateTicket(username, subject, message string) (*Ticket, error) {
	if DB == nil {
		return nil, errors.New("base de données non initialisée")
	}

	if subject == "" {
		return nil, errors.New("le sujet est requis")
	}

	if message == "" {
		return nil, errors.New("le message est requis")
	}

	result, err := DB.Exec(`
		INSERT INTO tickets (username, subject, message, status)
		VALUES (?, ?, ?, 'open')`,
		username, subject, message)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la création du ticket: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération de l'ID du ticket: %v", err)
	}

	return GetTicketByID(int(id))
}

// GetTicketByID récupère un ticket par son ID
func GetTicketByID(id int) (*Ticket, error) {
	if DB == nil {
		return nil, errors.New("base de données non initialisée")
	}

	var ticket Ticket
	var adminResponse sql.NullString
	var adminUsername sql.NullString
	var createdAtStr, updatedAtStr string

	err := DB.QueryRow(`
		SELECT id, username, subject, message, status, 
		       admin_response, admin_username, created_at, updated_at
		FROM tickets
		WHERE id = ?`,
		id,
	).Scan(
		&ticket.ID,
		&ticket.Username,
		&ticket.Subject,
		&ticket.Message,
		&ticket.Status,
		&adminResponse,
		&adminUsername,
		&createdAtStr,
		&updatedAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("ticket non trouvé")
		}
		return nil, fmt.Errorf("erreur lors de la récupération du ticket: %v", err)
	}

	// Parser les dates
	ticket.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
	ticket.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAtStr)

	if adminResponse.Valid {
		ticket.AdminResponse = adminResponse.String
	}
	if adminUsername.Valid {
		ticket.AdminUsername = adminUsername.String
	}

	return &ticket, nil
}

// GetUserTickets récupère tous les tickets d'un utilisateur
func GetUserTickets(username string) ([]Ticket, error) {
	if DB == nil {
		return nil, errors.New("base de données non initialisée")
	}

	rows, err := DB.Query(`
		SELECT id, username, subject, message, status, 
		       admin_response, admin_username, created_at, updated_at
		FROM tickets
		WHERE username = ?
		ORDER BY created_at DESC`,
		username)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des tickets: %v", err)
	}
	defer rows.Close()

	var tickets []Ticket
	for rows.Next() {
		var ticket Ticket
		var adminResponse sql.NullString
		var adminUsername sql.NullString
		var createdAtStr, updatedAtStr string

		err := rows.Scan(
			&ticket.ID,
			&ticket.Username,
			&ticket.Subject,
			&ticket.Message,
			&ticket.Status,
			&adminResponse,
			&adminUsername,
			&createdAtStr,
			&updatedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("erreur lors du scan des tickets: %v", err)
		}

		// Parser les dates
		ticket.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		ticket.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAtStr)

		if adminResponse.Valid {
			ticket.AdminResponse = adminResponse.String
		}
		if adminUsername.Valid {
			ticket.AdminUsername = adminUsername.String
		}

		tickets = append(tickets, ticket)
	}

	return tickets, nil
}

// GetAllTickets récupère tous les tickets (pour l'admin)
func GetAllTickets(statusFilter string) ([]Ticket, error) {
	if DB == nil {
		return nil, errors.New("base de données non initialisée")
	}

	var query string
	var args []interface{}

	if statusFilter != "" && statusFilter != "all" {
		query = `
			SELECT id, username, subject, message, status, 
			       admin_response, admin_username, created_at, updated_at
			FROM tickets
			WHERE status = ?
			ORDER BY created_at DESC`
		args = []interface{}{statusFilter}
	} else {
		query = `
			SELECT id, username, subject, message, status, 
			       admin_response, admin_username, created_at, updated_at
			FROM tickets
			ORDER BY created_at DESC`
		args = []interface{}{}
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des tickets: %v", err)
	}
	defer rows.Close()

	var tickets []Ticket
	for rows.Next() {
		var ticket Ticket
		var adminResponse sql.NullString
		var adminUsername sql.NullString
		var createdAtStr, updatedAtStr string

		err := rows.Scan(
			&ticket.ID,
			&ticket.Username,
			&ticket.Subject,
			&ticket.Message,
			&ticket.Status,
			&adminResponse,
			&adminUsername,
			&createdAtStr,
			&updatedAtStr,
		)
		if err != nil {
			return nil, fmt.Errorf("erreur lors du scan des tickets: %v", err)
		}

		// Parser les dates
		ticket.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		ticket.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAtStr)

		if adminResponse.Valid {
			ticket.AdminResponse = adminResponse.String
		}
		if adminUsername.Valid {
			ticket.AdminUsername = adminUsername.String
		}

		tickets = append(tickets, ticket)
	}

	return tickets, nil
}

// UpdateTicketStatus met à jour le statut d'un ticket
func UpdateTicketStatus(id int, status string) error {
	if DB == nil {
		return errors.New("base de données non initialisée")
	}

	validStatuses := map[string]bool{
		"open":        true,
		"in_progress": true,
		"closed":      true,
		"resolved":    true,
	}

	if !validStatuses[status] {
		return errors.New("statut invalide")
	}

	_, err := DB.Exec(`
		UPDATE tickets
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		status, id)
	if err != nil {
		return fmt.Errorf("erreur lors de la mise à jour du statut: %v", err)
	}

	return nil
}

// RespondToTicket ajoute une réponse admin à un ticket
func RespondToTicket(id int, adminUsername, response string) error {
	if DB == nil {
		return errors.New("base de données non initialisée")
	}

	if response == "" {
		return errors.New("la réponse ne peut pas être vide")
	}

	_, err := DB.Exec(`
		UPDATE tickets
		SET admin_response = ?, admin_username = ?, 
		    status = 'resolved', updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		response, adminUsername, id)
	if err != nil {
		return fmt.Errorf("erreur lors de l'ajout de la réponse: %v", err)
	}

	return nil
}

// GetTicketStats récupère les statistiques des tickets
func GetTicketStats() (map[string]int, error) {
	if DB == nil {
		return nil, errors.New("base de données non initialisée")
	}

	stats := make(map[string]int)

	rows, err := DB.Query(`
		SELECT status, COUNT(*) as count
		FROM tickets
		GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la récupération des statistiques: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("erreur lors du scan des statistiques: %v", err)
		}
		stats[status] = count
	}

	// S'assurer que tous les statuts existent
	if _, ok := stats["open"]; !ok {
		stats["open"] = 0
	}
	if _, ok := stats["in_progress"]; !ok {
		stats["in_progress"] = 0
	}
	if _, ok := stats["closed"]; !ok {
		stats["closed"] = 0
	}
	if _, ok := stats["resolved"]; !ok {
		stats["resolved"] = 0
	}

	return stats, nil
}

// DeleteTicket supprime un ticket par son ID
func DeleteTicket(id int) error {
	if DB == nil {
		return errors.New("base de données non initialisée")
	}

	// Vérifier si le ticket existe
	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM tickets WHERE id = ?)", id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("erreur lors de la vérification du ticket: %v", err)
	}
	if !exists {
		return errors.New("ticket non trouvé")
	}

	// Supprimer le ticket
	_, err = DB.Exec("DELETE FROM tickets WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("erreur lors de la suppression du ticket: %v", err)
	}

	return nil
}

// CanUserDeleteTicket vérifie si un utilisateur peut supprimer un ticket
func CanUserDeleteTicket(ticketID int, username string, isAdmin bool) (bool, error) {
	if DB == nil {
		return false, errors.New("base de données non initialisée")
	}

	// Les admins peuvent toujours supprimer
	if isAdmin {
		return true, nil
	}

	// Vérifier si l'utilisateur est le propriétaire du ticket
	var ticketUsername string
	err := DB.QueryRow("SELECT username FROM tickets WHERE id = ?", ticketID).Scan(&ticketUsername)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, errors.New("ticket non trouvé")
		}
		return false, fmt.Errorf("erreur lors de la vérification du ticket: %v", err)
	}

	return ticketUsername == username, nil
}

