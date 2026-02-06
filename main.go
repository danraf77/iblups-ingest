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

		// Esperar a que el stream esté completamente disponible
		time.Sleep(5 * time.Second)
		log.Printf("⏳ Iniciando captura de thumbnail...")

		// ✅ IMPORTANTE: Agregar vhost para conectarse al stream correcto
		rtmpURL := fmt.Sprintf("rtmp://srs:1935/%s/%s?vhost=51.210.109.197", appName, streamID)
		outputPath := "/app/thumbnails/" + fileName
		
		log.Printf("🔗 RTMP URL: %s", rtmpURL)
		log.Printf("💾 Output: %s", outputPath)
		
		// ✅ Usar timeout para evitar bloqueos + capturar 1 frame
		cmd := exec.Command("sh", "-c", 
			fmt.Sprintf("timeout 10 ffmpeg -y -i '%s' -vframes 1 -q:v 2 '%s'", rtmpURL, outputPath))

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		mu.Lock()
		activeProcesses[streamID] = cmd
		mu.Unlock()

		log.Printf("📸 FFmpeg iniciado con timeout de 10s...")
		if err := cmd.Run(); err != nil {
			// timeout retorna exit code 124, verificar si el archivo se generó
			if _, statErr := os.Stat(outputPath); statErr == nil {
				log.Printf("✅ Thumbnail generado (con timeout): %s", fileName)
			} else {
				log.Printf("❌ Error FFmpeg: %v", err)
			}
		} else {
			log.Printf("✅ Thumbnail generado correctamente: %s", fileName)
		}
		
		// Limpiar del mapa después de capturar
		mu.Lock()
		delete(activeProcesses, streamID)
		mu.Unlock()
	}(cb.Stream, cb.App)
}

func handleUnpublish(w http.ResponseWriter, r *http.Request) {
	var cb SRSCallback
	json.NewDecoder(r.Body).Decode(&cb)
	log.Printf("🔻 Unpublish detectado: %s", cb.Stream)
	w.Write([]byte("0"))

	go func(streamID string) {
		// Actualizar estado en base de datos
		updateData := map[string]interface{}{
			"is_on_live": false, 
			"modified":   time.Now().Format(time.RFC3339),
		}
		client.From("channels_channel").Update(updateData, "", "").Eq("stream_id", streamID).Execute()
		log.Printf("✅ Canal actualizado como offline: %s", streamID)
	}(cb.Stream)
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