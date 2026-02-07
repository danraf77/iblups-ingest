package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type MetricsCollector struct {
	supabase  *SupabaseService
	srsClient *SRSClient
}

func NewMetricsCollector(supabase *SupabaseService) *MetricsCollector {
	return &MetricsCollector{
		supabase:  supabase,
		srsClient: NewSRSClient(),
	}
}

func (m *MetricsCollector) Start() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("📊 Recolector de métricas iniciado (cada 30s)")

	for range ticker.C {
		go m.collectAndSaveMetrics()
	}
}

func (m *MetricsCollector) collectAndSaveMetrics() {
	client := m.supabase.GetClient()

	// 1. Obtener streams
	resp, err := http.Get("http://srs:1985/api/v1/streams/")
	if err != nil {
		log.Printf("❌ Error obteniendo streams para métricas: %v", err)
		return
	}
	defer resp.Body.Close()

	var srsStreamsResponse struct {
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

	// 2. Obtener recursos
	respRusage, _ := http.Get("http://srs:1985/api/v1/rusages/")
	var cpuPercent float64 = 0
	var memoryMB int64 = 0

	if respRusage != nil {
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

	// 3. Contar conexiones
	respClients, _ := http.Get("http://srs:1985/api/v1/clients/")
	publishers := 0
	players := 0
	totalConnections := 0

	if respClients != nil {
		defer respClients.Body.Close()
		var clientsResponse struct {
			Clients []struct {
				Type string `json:"type"`
			} `json:"clients"`
		}
		json.NewDecoder(respClients.Body).Decode(&clientsResponse)

		for _, c := range clientsResponse.Clients {
			totalConnections++
			if c.Type == "fmle-publish" || c.Type == "flash-publish" {
				publishers++
			} else {
				players++
			}
		}
	}

	// 4. Guardar métricas del servidor
	serverMetric := map[string]interface{}{
		"cpu_percent":       cpuPercent,
		"memory_mb":         memoryMB,
		"total_streams":     len(srsStreamsResponse.Streams),
		"total_connections": totalConnections,
		"publishers":        publishers,
		"players":           players,
	}

	_, err = client.From("server_metrics").Insert(serverMetric, false, "", "", "").Execute()
	if err != nil {
		log.Printf("❌ Error guardando server_metrics: %v", err)
	}

	// 5. Guardar métricas de streams
	for _, stream := range srsStreamsResponse.Streams {
		resolution := ""
		codec := ""
		if stream.Video != nil {
			resolution = fmt.Sprintf("%dx%d", stream.Video.Width, stream.Video.Height)
			codec = stream.Video.Codec
		}

		streamMetric := map[string]interface{}{
			"stream_id":     stream.ID,
			"stream_name":   stream.Name,
			"app":           stream.App,
			"clients":       stream.Clients,
			"recv_kbps":     stream.Kbps.RecvKbps,
			"send_kbps":     stream.Kbps.SendKbps,
			"is_publishing": stream.Publish.Active,
			"video_codec":   codec,
			"resolution":    resolution,
		}

		_, err = client.From("stream_metrics").Insert(streamMetric, false, "", "", "").Execute()
		if err != nil {
			log.Printf("❌ Error guardando stream_metrics: %v", err)
		}
	}

	// 6. Alertas
	if cpuPercent > 80 {
		alert := map[string]interface{}{
			"event_type": "high_cpu",
			"severity":   "warning",
			"message":    fmt.Sprintf("CPU usage alto: %.1f%%", cpuPercent),
			"metadata":   map[string]interface{}{"cpu": cpuPercent},
		}
		client.From("system_events").Insert(alert, false, "", "", "").Execute()
	}

	log.Printf("✅ Métricas guardadas: CPU=%.1f%%, Mem=%dMB, Streams=%d, Conn=%d",
		cpuPercent, memoryMB, len(srsStreamsResponse.Streams), totalConnections)
}