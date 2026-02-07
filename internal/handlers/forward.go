package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"srs-backend/internal/models"
)

type ForwardHandler struct {
	targetURL string
}

func NewForwardHandler(targetURL string) *ForwardHandler {
	return &ForwardHandler{targetURL: targetURL}
}

func (h *ForwardHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var cb models.SRSCallback
	if err := json.NewDecoder(r.Body).Decode(&cb); err != nil {
		log.Printf("❌ Error decode forward: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 1})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if h.targetURL == "" {
		log.Printf("⚠️ TARGET_FORWARD_URL no configurado")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"urls": []string{},
			},
		})
		return
	}

	resp := map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"urls": []string{fmt.Sprintf("%s/%s/%s", h.targetURL, cb.App, cb.Stream)},
		},
	}

	log.Printf("➡️ Forwarding a: %s/%s/%s", h.targetURL, cb.App, cb.Stream)
	json.NewEncoder(w).Encode(resp)
}