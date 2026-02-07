package services

import (
	"crypto/md5"
	"encoding/hex"
	"log"
	// Cambio: se agrega time para last_seen (Firma: Cursor)
	"time"

	"github.com/supabase-community/supabase-go"
)

type SupabaseService struct {
	client *supabase.Client
}

func NewSupabaseService(url, key string) *SupabaseService {
	client, _ := supabase.NewClient(url, key, nil)
	return &SupabaseService{client: client}
}

func (s *SupabaseService) GetClient() *supabase.Client {
	return s.client
}

func (s *SupabaseService) GetPersistentHash(id string) string {
	hash := md5.Sum([]byte(id))
	return hex.EncodeToString(hash[:])
}

// ✅ CORREGIDO: RegisterServer con 3 valores de retorno
func (s *SupabaseService) RegisterServer(serverID, serverIP string) error {
	serverData := map[string]interface{}{
		"server_id":   serverID,
		"server_ip":   serverIP,
		"server_name": "SRS Server " + serverID,
		"is_active":   true,
	}

	// Upsert: inserta o actualiza si ya existe
	// Cambio: prefijo de tabla actualizado a server_ingest_ (Firma: Cursor)
	_, _, err := s.client.From("server_ingest_srs_servers").
		Upsert(serverData, "server_id", "", "").
		Execute()

	if err != nil {
		log.Printf("❌ Error registrando servidor: %v", err)
		return err
	}

	return nil
}

// Cambio: actualizar last_seen de servidor registrado (Firma: Cursor)
func (s *SupabaseService) UpdateServerHeartbeat(serverID, serverIP string) error {
	updateData := map[string]interface{}{
		"server_ip": serverIP,
		"last_seen": time.Now().UTC(),
	}

	_, _, err := s.client.From("server_ingest_srs_servers").
		Update(updateData, "", "").
		Eq("server_id", serverID).
		Execute()
	if err != nil {
		log.Printf("❌ Error actualizando last_seen: %v", err)
		return err
	}

	return nil
}