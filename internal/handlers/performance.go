package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"srs-backend/internal/models"
)

type PerformanceHandler struct{}

func NewPerformanceHandler() *PerformanceHandler {
	return &PerformanceHandler{}
}

func (h *PerformanceHandler) Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Obtener rusage
	respRusage, err := http.Get("http://srs:1985/api/v1/rusages/")
	var cpuPercent float64 = 0
	var memoryMB int64 = 0

	if err == nil {
		defer respRusage.Body.Close()
		var rusageResponse struct {
			Data struct {
				Percent float64 `json:"percent"`
				MemKB   int64   `json:"mem_kbyte"`
			} `json:"data"`
		}
		json.NewDecoder(respRusage.Body).Decode(&rusageResponse)
		cpuPercent = rusageResponse.Data.Percent
		memoryMB = rusageResponse.Data.MemKB / 1024
	}

	// Obtener streams para contar conexiones
	respStreams, err := http.Get("http://srs:1985/api/v1/streams/")
	totalConnections := 0
	if err == nil {
		defer respStreams.Body.Close()
		var streamsResponse struct {
			Streams []struct {
				Clients int `json:"clients"`
			} `json:"streams"`
		}
		json.NewDecoder(respStreams.Body).Decode(&streamsResponse)
		for _, s := range streamsResponse.Streams {
			totalConnections += s.Clients
		}
	}

	perf := models.PerformanceStats{
		CPU:         cpuPercent,
		Memory:      memoryMB,
		Connections: totalConnections,
	}

	json.NewEncoder(w).Encode(perf)
	log.Printf("📊 Performance solicitado: CPU=%.1f%%, Mem=%dMB, Conn=%d", cpuPercent, memoryMB, totalConnections)
}