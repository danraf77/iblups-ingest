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
		fileName := getPersistentHash(channelID) + ".jpg"
		log.Printf("✅ Canal encontrado (ID: %s). Generando thumbnail: %s", channelID, fileName)

		updateData := map[string]interface{}{
			"is_on_live":  true,
			"last_status": "online",
			"cover":       fileName,
			"modified":    time.Now().Format(time.RFC3339),
		}
		
		client.From("channels_channel").Update(updateData, "", "").Eq("id", channelID).Execute()

		// Esperar 3 segundos para que el stream esté disponible
		time.Sleep(3 * time.Second)
		log.Printf("⏳ Stream disponible, iniciando captura vía HTTP-FLV...")

		// ✅ HTTP-FLV: Solo lectura, sin conflictos con OBS
		httpFlvURL := fmt.Sprintf("http://srs:8080/%s/%s.flv", appName, streamID)
		
		cmd := exec.Command("ffmpeg", 
			"-loglevel", "error",
			"-y",
			"-i", httpFlvURL,
			"-f", "image2", 
			"-vf", "fps=1/10,scale=480:-1", 
			"-update", "1",
			"/app/thumbnails/"+fileName)

		mu.Lock()
		activeProcesses[streamID] = cmd
		mu.Unlock()

		log.Printf("📸 FFmpeg iniciado para %s", fileName)
		if err := cmd.Run(); err != nil {
			log.Printf("❌ Error FFmpeg: %v", err)
		} else {
			log.Printf("✅ Thumbnail generado correctamente: %s", fileName)
		}
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