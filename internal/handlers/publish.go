package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"srs-backend/internal/models"
	"srs-backend/internal/services"
)

type PublishHandler struct {
	supabase  *services.SupabaseService
	thumbnail *services.ThumbnailService
	// Cambio: guardar IP del servidor para fallback (Firma: Cursor)
	serverIP  string
}

func NewPublishHandler(supabase *services.SupabaseService, thumbnail *services.ThumbnailService, serverIP string) *PublishHandler {
	return &PublishHandler{
		supabase:  supabase,
		thumbnail: thumbnail,
		serverIP:  serverIP,
	}
}

func (h *PublishHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var cb models.SRSCallback
	if err := json.NewDecoder(r.Body).Decode(&cb); err != nil {
		log.Printf("❌ Error decode: %v", err)
		w.Write([]byte("1"))
		return
	}

	log.Printf("📢 Publish detectado: App=%s, Stream=%s", cb.App, cb.Stream)
	w.Write([]byte("0"))

	go h.processPublish(cb)
}

func (h *PublishHandler) processPublish(cb models.SRSCallback) {
	client := h.supabase.GetClient()

	var results []struct{ ID string `json:"id"` }
	_, err := client.From("channels_channel").Select("id", "1", false).Eq("stream_id", cb.Stream).ExecuteTo(&results)

	if err != nil || len(results) == 0 {
		log.Printf("⚠️ Canal no encontrado en Supabase para stream_id: %s", cb.Stream)
		return
	}

	channelID := results[0].ID
	fileName := h.supabase.GetPersistentHash(channelID) + ".jpg"
	log.Printf("✅ Canal encontrado (ID: %s). Generando thumbnail: %s", channelID, fileName)

	updateData := map[string]interface{}{
		"is_on_live":  true,
		"last_status": "online",
		"cover":       fileName,
		"modified":    time.Now().Format(time.RFC3339),
	}

	client.From("channels_channel").Update(updateData, "", "").Eq("id", channelID).Execute()

	// Cambio: usar vhost real del callback para evitar fallos de thumbnail (Firma: Cursor)
	vhost := cb.Vhost
	if vhost == "" {
		// Cambio: usar IP del server actual como fallback (Firma: Cursor)
		vhost = h.serverIP
	}
	rtmpURL := fmt.Sprintf("rtmp://srs:1935/%s/%s?vhost=%s", cb.App, cb.Stream, vhost)
	outputPath := "/app/thumbnails/" + fileName

	h.thumbnail.StartCapture(cb.Stream, cb.App, fileName, rtmpURL, outputPath)
}