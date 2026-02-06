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
	Stream string `json:"stream"`
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
	json.NewDecoder(r.Body).Decode(&cb)
	w.Write([]byte("0"))

	go func(streamID string) {
		var results []struct{ ID string `json:"id"` }
		// Corregido: Select antes de Eq
		_, err := client.From("channels_channel").Select("id", "1", false).Eq("stream_id", streamID).ExecuteTo(&results)

		if err != nil || len(results) == 0 { return }
		channelID := results[0].ID
		fileName := getPersistentHash(channelID) + ".jpg"

		updateData := map[string]interface{}{
			"is_on_live":  true,
			"last_status": "online",
			"cover":       fileName,
			"modified":    time.Now().Format(time.RFC3339),
		}
		
		// Corregido: Update antes de Eq
		client.From("channels_channel").Update(updateData, "", "").Eq("id", channelID).Execute()

		cmd := exec.Command("ffmpeg", "-loglevel", "quiet", "-y",
			"-i", "rtmp://srs:1935/live/"+streamID,
			"-f", "image2", "-vf", "fps=1/10,scale=480:-1", "-update", "1",
			"/app/thumbnails/"+fileName)

		mu.Lock()
		activeProcesses[streamID] = cmd
		mu.Unlock()

		cmd.Run()
	}(cb.Stream)
}

func handleUnpublish(w http.ResponseWriter, r *http.Request) {
	var cb SRSCallback
	json.NewDecoder(r.Body).Decode(&cb)
	w.Write([]byte("0"))

	go func(streamID string) {
		mu.Lock()
		if cmd, ok := activeProcesses[streamID]; ok {
			cmd.Process.Kill()
			delete(activeProcesses, streamID)
		}
		mu.Unlock()

		updateData := map[string]interface{}{
			"is_on_live": false, 
			"modified":   time.Now().Format(time.RFC3339),
		}
		// Corregido: Update antes de Eq
		client.From("channels_channel").Update(updateData, "", "").Eq("stream_id", streamID).Execute()
	}(cb.Stream)
}

func handleForward(w http.ResponseWriter, r *http.Request) {
	var cb SRSCallback
	json.NewDecoder(r.Body).Decode(&cb)
	target := os.Getenv("TARGET_FORWARD_URL")
	resp := map[string]interface{}{"code": 0, "data": map[string]interface{}{"urls": []string{fmt.Sprintf("%s/%s", target, cb.Stream)}}}
	json.NewEncoder(w).Encode(resp)
}