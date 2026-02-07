package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"srs-backend/internal/models"
)

type StatsHandler struct{}

func NewStatsHandler() *StatsHandler {
	return &StatsHandler{}
}

func (h *StatsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 1. Obtener streams del SRS
	resp, err := http.Get("http://srs:1985/api/v1/streams/")
	if err != nil {
		log.Printf("❌ Error obteniendo streams: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	var srsStreamsResponse struct {
		Code    int    `json:"code"`
		Server  string `json:"server"`
		Streams []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			App     string `json:"app"`
			Clients int    `json:"clients"`
			Kbps    struct {
				RecvKbps int `json:"recv_30s"`
				SendKbps int `json:"send_30s"`
			} `json:"kbps"`
			Publish struct {
				Active bool `json:"active"`
			} `json:"publish"`
			Video *struct {
				Codec  string `json:"codec"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"video"`
		} `json:"streams"`
	}

	json.NewDecoder(resp.Body).Decode(&srsStreamsResponse)

	// 2. Obtener uso de recursos
	respRusage, err := http.Get("http://srs:1985/api/v1/rusages/")
	var cpuPercent float64 = 0
	var memoryMB int64 = 0

	if err == nil {
		defer respRusage.Body.Close()
		var rusageResponse struct {
			Data struct {
				OK      bool    `json:"ok"`
				Percent float64 `json:"percent"`
				MemKB   int64   `json:"mem_kbyte"`
			} `json:"data"`
		}
		json.NewDecoder(respRusage.Body).Decode(&rusageResponse)
		cpuPercent = rusageResponse.Data.Percent
		memoryMB = rusageResponse.Data.MemKB / 1024
	}

	// 3. Construir respuesta
	streams := []models.StreamInfo{}
	totalConnections := 0

	for _, s := range srsStreamsResponse.Streams {
		videoCodec := ""
		width := 0
		height := 0
		if s.Video != nil {
			videoCodec = s.Video.Codec
			width = s.Video.Width
			height = s.Video.Height
		}

		streams = append(streams, models.StreamInfo{
			ID:         s.ID,
			Name:       s.Name,
			App:        s.App,
			Clients:    s.Clients,
			RecvKbps:   s.Kbps.RecvKbps,
			SendKbps:   s.Kbps.SendKbps,
			IsPublish:  s.Publish.Active,
			VideoCodec: videoCodec,
			Width:      width,
			Height:     height,
		})

		totalConnections += s.Clients
	}

	stats := models.SRSStats{
		Server: models.ServerStats{
			Uptime:       time.Now().Unix(),
			Connections:  totalConnections,
			TotalStreams: len(streams),
			Version:      "6.0.184",
		},
		Streams: streams,
		Resources: models.ResourceStats{
			CPU:    cpuPercent,
			Memory: memoryMB,
		},
	}

	json.NewEncoder(w).Encode(stats)
	log.Printf("📊 Stats solicitadas: %d streams, %d conexiones", len(streams), totalConnections)
}