package services

import (
	"crypto/md5"
	"encoding/hex"

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