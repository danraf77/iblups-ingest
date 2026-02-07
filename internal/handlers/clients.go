package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"srs-backend/internal/models"
)

type ClientsHandler struct{}

func NewClientsHandler() *ClientsHandler {
	return &ClientsHandler{}
}

func (h *ClientsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	resp, err := http.Get("http://srs:1985/api/v1/clients/")
	if err != nil {
		log.Printf("❌ Error obteniendo clientes: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	var srsClientsResponse struct {
		Code    int `json:"code"`
		Clients []struct {
			ID        string `json:"id"`
			IP        string `json:"ip"`
			Type      string `json:"type"`
			Stream    string `json:"stream"`
			App       string `json:"app"`
			Alive     int64  `json:"alive"`
			SendBytes int64  `json:"send_bytes"`
			RecvBytes int64  `json:"recv_bytes"`
		} `json:"clients"`
	}

	json.NewDecoder(resp.Body).Decode(&srsClientsResponse)

	clients := []models.ClientInfo{}
	for _, c := range srsClientsResponse.Clients {
		clients = append(clients, models.ClientInfo{
			ID:        c.ID,
			IP:        c.IP,
			Type:      c.Type,
			Stream:    c.Stream,
			App:       c.App,
			Alive:     c.Alive,
			SendBytes: c.SendBytes,
			RecvBytes: c.RecvBytes,
		})
	}

	response := map[string]interface{}{
		"total":   len(clients),
		"clients": clients,
	}

	json.NewEncoder(w).Encode(response)
	log.Printf("📊 Clientes solicitados: %d conectados", len(clients))
}