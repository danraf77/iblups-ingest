package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"srs-backend/internal/models"
	"srs-backend/internal/services"
)

type SessionsHandler struct {
	supabase *services.SupabaseService
	serverID string
	serverIP string
	mu       sync.Mutex
	// Cambio: map de sesiones activas por client_id (Firma: Cursor)
	activeSessions map[string]time.Time
}

// Cambio: handler para on_play/on_stop de SRS (Firma: Cursor)
func NewSessionsHandler(supabase *services.SupabaseService, serverID, serverIP string) *SessionsHandler {
	return &SessionsHandler{
		supabase:       supabase,
		serverID:       serverID,
		serverIP:       serverIP,
		activeSessions: make(map[string]time.Time),
	}
}

func (h *SessionsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var cb models.SRSCallback
	if err := json.NewDecoder(r.Body).Decode(&cb); err != nil {
		log.Printf("❌ Error decode sessions: %v", err)
		w.Write([]byte("1"))
		return
	}

	switch cb.Action {
	case "on_play":
		w.Write([]byte("0"))
		go h.processPlay(cb)
		return
	case "on_stop":
		w.Write([]byte("0"))
		go h.processStop(cb)
		return
	default:
		log.Printf("⚠️ Acción SRS no soportada en sessions: %s", cb.Action)
		w.Write([]byte("0"))
		return
	}
}

func (h *SessionsHandler) processPlay(cb models.SRSCallback) {
	client := h.supabase.GetClient()
	connectedAt := time.Now().UTC()

	insertData := map[string]interface{}{
		"server_id":    h.serverID,
		"server_ip":    h.serverIP,
		// Cambio: persistir client_id y stream_id si están presentes (Firma: Cursor)
		"client_id":    cb.ClientID,
		"client_ip":    cb.IP,
		"client_type":  "play",
		"stream_id":    cb.StreamID,
		"stream_name":  cb.Stream,
		"app":          cb.App,
		"connected_at": connectedAt,
	}

	// Cambio: insertar sesion de cliente on_play (Firma: Cursor)
	_, _, err := client.From("server_ingest_client_connections").Insert(insertData, false, "", "", "").Execute()
	if err != nil {
		log.Printf("❌ Error guardando server_ingest_client_connections: %v", err)
		return
	}

	if cb.ClientID != "" {
		h.mu.Lock()
		h.activeSessions[cb.ClientID] = connectedAt
		h.mu.Unlock()
	}
}

func (h *SessionsHandler) processStop(cb models.SRSCallback) {
	client := h.supabase.GetClient()
	disconnectedAt := time.Now().UTC()

	var connectedAt time.Time
	var ok bool

	if cb.ClientID != "" {
		h.mu.Lock()
		connectedAt, ok = h.activeSessions[cb.ClientID]
		delete(h.activeSessions, cb.ClientID)
		h.mu.Unlock()
	}

	durationSeconds := 0
	if ok {
		durationSeconds = int(disconnectedAt.Sub(connectedAt).Seconds())
	}

	updateData := map[string]interface{}{
		"disconnected_at":  disconnectedAt,
		"duration_seconds": durationSeconds,
	}

	// Cambio: cerrar sesion on_stop (Firma: Cursor)
	query := client.From("server_ingest_client_connections").
		Update(updateData, "", "").
		Eq("server_id", h.serverID).
		Eq("app", cb.App).
		Eq("stream_name", cb.Stream).
		Eq("client_ip", cb.IP).
		Eq("client_type", "play")

	if cb.ClientID != "" {
		// Cambio: cerrar sesión por client_id cuando exista (Firma: Cursor)
		query = query.Eq("client_id", cb.ClientID)
	} else if ok {
		query = query.Eq("connected_at", connectedAt)
	}

	if _, _, err := query.Execute(); err != nil {
		log.Printf("❌ Error cerrando sesion server_ingest_client_connections: %v", err)
	}
}
