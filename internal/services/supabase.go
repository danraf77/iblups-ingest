package services

import (
	"crypto/md5"
	"encoding/hex"
	"log"

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
	_, _, err := s.client.From("iblups_srs_servers").
		Upsert(serverData, "server_id", "", "").
		Execute()

	if err != nil {
		log.Printf("❌ Error registrando servidor: %v", err)
		return err
	}

	return nil
}