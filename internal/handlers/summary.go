package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"srs-backend/internal/models"
)

type SummaryHandler struct{}

func NewSummaryHandler() *SummaryHandler {
	return &SummaryHandler{}
}

func (h *SummaryHandler) Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Obtener clientes
	respClients, err := http.Get("http://srs:1985/api/v1/clients/")
	publishers := 0
	players := 0
	totalClients := 0

	if err == nil {
		defer respClients.Body.Close()
		var clientsResponse struct {
			Clients []struct {
				Type string `json:"type"`
			} `json:"clients"`
		}
		json.NewDecoder(respClients.Body).Decode(&clientsResponse)

		for _, c := range clientsResponse.Clients {
			totalClients++
			if c.Type == "fmle-publish" || c.Type == "flash-publish" {
				publishers++
			} else {
				players++
			}
		}
	}

	summary := models.ServerSummary{
		Version:      "6.0.184",
		PID:          1,
		Uptime:       time.Now().Unix(),
		Publishers:   publishers,
		Players:      players,
		TotalClients: totalClients,
	}

	json.NewEncoder(w).Encode(summary)
	log.Printf("📊 Summary solicitado: %d publishers, %d players", publishers, players)
}