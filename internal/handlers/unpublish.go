package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"srs-backend/internal/models"
	"srs-backend/internal/services"
)

type UnpublishHandler struct {
	supabase  *services.SupabaseService
	thumbnail *services.ThumbnailService
}

func NewUnpublishHandler(supabase *services.SupabaseService, thumbnail *services.ThumbnailService) *UnpublishHandler {
	return &UnpublishHandler{
		supabase:  supabase,
		thumbnail: thumbnail,
	}
}

func (h *UnpublishHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var cb models.SRSCallback
	json.NewDecoder(r.Body).Decode(&cb)
	log.Printf("🔻 Unpublish detectado: %s", cb.Stream)
	w.Write([]byte("0"))

	go h.processUnpublish(cb)
}

func (h *UnpublishHandler) processUnpublish(cb models.SRSCallback) {
	// Detener captura de thumbnails
	h.thumbnail.StopCapture(cb.Stream)

	// Actualizar base de datos
	client := h.supabase.GetClient()
	updateData := map[string]interface{}{
		"is_on_live": false,
		"modified":   time.Now().Format(time.RFC3339),
	}

	client.From("channels_channel").Update(updateData, "", "").Eq("stream_id", cb.Stream).Execute()
	log.Printf("✅ Canal actualizado como offline: %s", cb.Stream)
}