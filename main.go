package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/supabase-community/supabase-go"
)

var (
	client          *supabase.Client
	activeProcesses = make(map[string]*exec.Cmd)
	mu              sync.Mutex
)

type SRSCallback struct {
	Action string `json:"action"`
	App    string `json:"app"`
	Stream string `json:"stream"`
	Param  string `json:"param"`
}

func getPersistentHash(id string) string {
	hash := md5.Sum([]byte(id))
	return hex.EncodeToString(hash[:])
}

func main() {
	client, _ = supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	http.HandleFunc("/api/v1/publish", handlePublish)
	http.HandleFunc("/api/v1/unpublish", handleUnpublish)
	http.HandleFunc("/api/v1/forward", handleForward)

	log.Println("🚀 Backend Go iniciado en puerto 3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func handlePublish(w http.ResponseWriter, r *http.Request) {
	var cb SRSCallback
	if err := json.NewDecoder(r.Body).Decode(&cb); err != nil {
		log.Printf("❌ Error decode: %v", err)
		w.Write([]byte("1"))
		return
	}
	
	log.Printf("📢 Publish detectado: App=%s, Stream=%s", cb.App, cb.Stream)
	w.Write([]byte("0"))

	go func(streamID string, appName string) {
		var results []struct{ ID string `json:"id"` }
		_, err := client.From("channels_channel").Select("id", "1", false).Eq("stream_id", streamID).ExecuteTo(&results)

		if err != nil || len(results) == 0 {
			log.Printf("⚠️ Canal no encontrado en Supabase para stream_id: %s", streamID)
			return
		}

		channelID := results[0].ID
		finalFileName := getPersistentHash(channelID) + ".jpg"
		srsSnapshotPath := fmt.Sprintf("/snapshots/%s/%s.jpg", appName, streamID)
		finalPath := "/app/thumbnails/" + finalFileName
		
		log.Printf("✅ Canal encontrado (ID: %s). Esperando snapshot de SRS...", channelID)

		// Actualizar DB inmediatamente
		updateData := map[string]interface{}{
			"is_on_live":  true,
			"last_status": "online",
			"cover":       finalFileName,
			"modified":    time.Now().Format(time.RFC3339),
		}
		client.From("channels_channel").Update(updateData, "", "").Eq("id", channelID).Execute()

		// Esperar a que SRS genere el snapshot (puede tardar unos segundos)
		maxRetries := 30 // 30 segundos máximo
		for i := 0; i < maxRetries; i++ {
			time.Sleep(1 * time.Second)
			
			if _, err := os.Stat(srsSnapshotPath); err == nil {
				// El snapshot existe, copiarlo
				log.Printf("📸 Snapshot encontrado, copiando a: %s", finalFileName)
				
				// Crear directorio si no existe
				os.MkdirAll(filepath.Dir(finalPath), 0755)
				
				// Copiar el archivo
				copyCmd := exec.Command("cp", srsSnapshotPath, finalPath)
				if err := copyCmd.Run(); err != nil {
					log.Printf("❌ Error copiando snapshot: %v", err)
				} else {
					log.Printf("✅ Thumbnail generado exitosamente: %s", finalFileName)
				}
				return
			}
			
			if i%5 == 0 && i > 0 {
				log.Printf("⏳ Esperando snapshot... (%d/%d)", i, maxRetries)
			}
		}
		
		log.Printf("⚠️ Timeout esperando snapshot para %s", streamID)
	}(cb.Stream, cb.App)
}

func handleUnpublish(w http.ResponseWriter, r *http.Request) {
	var cb SRSCallback
	json.NewDecoder(r.Body).Decode(&cb)
	log.Printf("🔻 Unpublish detectado: %s", cb.Stream)
	w.Write([]byte("0"))

	go func(streamID string, appName string) {
		// Limpiar snapshot temporal de SRS
		srsSnapshotPath := fmt.Sprintf("/snapshots/%s/%s.jpg", appName, streamID)
		os.Remove(srsSnapshotPath)
		
		updateData := map[string]interface{}{
			"is_on_live": false, 
			"modified":   time.Now().Format(time.RFC3339),
		}
		client.From("channels_channel").Update(updateData, "", "").Eq("stream_id", streamID).Execute()
		log.Printf("✅ Canal actualizado como offline: %s", streamID)
	}(cb.Stream, cb.App)
}

func handleForward(w http.ResponseWriter, r *http.Request) {
	var cb SRSCallback
	if err := json.NewDecoder(r.Body).Decode(&cb); err != nil {
		log.Printf("❌ Error decode forward: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 1})
		return
	}
	
	target := os.Getenv("TARGET_FORWARD_URL")
	if target == "" {
		log.Printf("⚠️ TARGET_FORWARD_URL no configurado")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "urls": []string{}})
		return
	}
	
	resp := map[string]interface{}{
		"code": 0,
		"urls": []string{fmt.Sprintf("%s/%s/%s", target, cb.App, cb.Stream)},
	}
	
	log.Printf("➡️ Forwarding a: %s/%s/%s", target, cb.App, cb.Stream)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}