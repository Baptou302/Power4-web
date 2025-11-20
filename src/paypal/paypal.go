package paypal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"power4/src/utils"
)

var (
	paypalClientID     = utils.GetEnv("PAYPAL_CLIENT_ID", "Aec3OfeVzd81VSqCObYVfG1DehqBZvop-xfmLoZVxtMIzJfdo5jLRP-mHKbSoeDBehIStSLGZ4DEPYOV")
	paypalClientSecret = utils.GetEnv("PAYPAL_CLIENT_SECRET", "EIRbvVImEd-dJsgRwSHUlpt854ritmErz_9Bq6pwRYmrfiM3_wwec037bmnEatKNpkO1-dtZSGZC7t8L")
	paypalBaseURL      = utils.GetEnv("PAYPAL_BASE_URL", "https://api.sandbox.paypal.com")
)

type PayPalOrder struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type CreateOrderRequest struct {
	Items []CartItem `json:"items"`
	Total float64    `json:"total"`
}

type CartItem struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

type CreateOrderResponse struct {
	ID string `json:"id"`
}

type CaptureOrderResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// getAccessToken obtient un token d'accès PayPal
func getAccessToken() (string, error) {
	url := paypalBaseURL + "/v1/oauth2/token"

	req, err := http.NewRequest("POST", url, bytes.NewBufferString("grant_type=client_credentials"))
	if err != nil {
		return "", fmt.Errorf("erreur création requête: %v", err)
	}

	req.SetBasicAuth(paypalClientID, paypalClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("erreur requête HTTP: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("erreur lecture réponse: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("erreur token (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken      string `json:"access_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("erreur parsing JSON: %v - body: %s", err, string(body))
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("erreur PayPal: %s - %s", tokenResp.Error, tokenResp.ErrorDescription)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("token d'accès vide - body: %s", string(body))
	}

	return tokenResp.AccessToken, nil
}

// CreateOrder crée une commande PayPal
func CreateOrder(items []CartItem, total float64) (*CreateOrderResponse, error) {
	// Valider que le panier n'est pas vide
	if len(items) == 0 {
		return nil, fmt.Errorf("le panier est vide")
	}

	// Calculer le total des items
	itemTotal := 0.0
	for _, item := range items {
		if item.Price <= 0 {
			return nil, fmt.Errorf("le prix de l'article '%s' est invalide: %.2f", item.Name, item.Price)
		}
		itemTotal += item.Price
	}

	// Valider que le total est supérieur à 0
	if total <= 0 {
		return nil, fmt.Errorf("le montant total doit être supérieur à 0, reçu: %.2f", total)
	}

	// Utiliser le total calculé si le total fourni est différent (pour éviter les erreurs d'arrondi)
	if total != itemTotal {
		// Utiliser le total calculé des items pour plus de précision
		total = itemTotal
	}

	accessToken, err := getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'obtention du token: %v", err)
	}

	// Construire la requête PayPal
	orderData := map[string]interface{}{
		"intent": "CAPTURE",
		"purchase_units": []map[string]interface{}{
			{
				"amount": map[string]interface{}{
					"currency_code": "EUR",
					"value":         fmt.Sprintf("%.2f", total),
					"breakdown": map[string]interface{}{
						"item_total": map[string]interface{}{
							"currency_code": "EUR",
							"value":         fmt.Sprintf("%.2f", itemTotal),
						},
					},
				},
				"items": buildPayPalItems(items),
			},
		},
		"application_context": map[string]interface{}{
			"return_url": utils.GetEnv("PAYPAL_RETURN_URL", "https://localhost:3443/shop?success=true"),
			"cancel_url": utils.GetEnv("PAYPAL_CANCEL_URL", "https://localhost:3443/shop?canceled=true"),
		},
	}

	jsonData, err := json.Marshal(orderData)
	if err != nil {
		return nil, err
	}

	url := paypalBaseURL + "/v2/checkout/orders"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lecture réponse: %v", err)
	}

	log.Printf("PayPal CreateOrder - Status: %d, Body: %s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("erreur PayPal (status %d): %s", resp.StatusCode, string(body))
	}

	// Parser la réponse complète pour voir la structure
	var orderResponse map[string]interface{}
	if err := json.Unmarshal(body, &orderResponse); err != nil {
		return nil, fmt.Errorf("erreur lors du parsing de la réponse PayPal: %v - body: %s", err, string(body))
	}

	// Extraire l'ID de la commande
	orderID, ok := orderResponse["id"].(string)
	if !ok || orderID == "" {
		return nil, fmt.Errorf("ID de commande manquant ou invalide dans la réponse: %s", string(body))
	}

	log.Printf("PayPal CreateOrder - Order ID créé: %s", orderID)

	return &CreateOrderResponse{ID: orderID}, nil
}

// CaptureOrder capture un paiement PayPal
func CaptureOrder(orderIDParam string) (*CaptureOrderResponse, error) {
	accessToken, err := getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("erreur lors de l'obtention du token: %v", err)
	}

	url := paypalBaseURL + "/v2/checkout/orders/" + orderIDParam + "/capture"
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lecture réponse: %v", err)
	}

	log.Printf("PayPal CaptureOrder - Status: %d, Body: %s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("erreur PayPal (status %d): %s", resp.StatusCode, string(body))
	}

	// Parser la réponse complète
	var captureResponse map[string]interface{}
	if err := json.Unmarshal(body, &captureResponse); err != nil {
		return nil, fmt.Errorf("erreur lors du parsing de la réponse PayPal: %v - body: %s", err, string(body))
	}

	// Extraire l'ID et le statut
	var orderID, status string
	if id, ok := captureResponse["id"].(string); ok {
		orderID = id
	}
	if st, ok := captureResponse["status"].(string); ok {
		status = st
	} else {
		status = "UNKNOWN"
	}

	if orderID == "" {
		return nil, fmt.Errorf("ID de commande manquant dans la réponse: %s", string(body))
	}

	log.Printf("PayPal CaptureOrder - Order ID: %s, Status: %s", orderID, status)

	return &CaptureOrderResponse{ID: orderID, Status: status}, nil
}

// buildPayPalItems construit les items au format PayPal
func buildPayPalItems(items []CartItem) []map[string]interface{} {
	paypalItems := make([]map[string]interface{}, len(items))
	for i, item := range items {
		paypalItems[i] = map[string]interface{}{
			"name": item.Name,
			"unit_amount": map[string]interface{}{
				"currency_code": "EUR",
				"value":         fmt.Sprintf("%.2f", item.Price),
			},
			"quantity": "1",
		}
	}
	return paypalItems
}

// IsConfigured vérifie si PayPal est configuré
func IsConfigured() bool {
	return paypalClientID != "" && paypalClientSecret != ""
}

// GetClientID retourne le client ID PayPal (pour le frontend)
func GetClientID() string {
	return paypalClientID
}
