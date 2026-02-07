package main

import (
	"log"
	"net/http"
	"os"

	"srs-backend/internal/config"
	"srs-backend/internal/handlers"
	"srs-backend/internal/services"
)

func main() {
	// Inicializar configuración
	cfg := config.New()

	// Inicializar servicios
	supabaseService := services.NewSupabaseService(cfg.SupabaseURL, cfg.SupabaseKey)
	thumbnailService := services.NewThumbnailService()
	metricsCollector := services.NewMetricsCollector(supabaseService)

	// Iniciar recolector de métricas en background
	go metricsCollector.Start()

	// Inicializar handlers
	publishHandler := handlers.NewPublishHandler(supabaseService, thumbnailService)
	unpublishHandler := handlers.NewUnpublishHandler(supabaseService, thumbnailService)
	forwardHandler := handlers.NewForwardHandler(cfg.TargetForwardURL)
	statsHandler := handlers.NewStatsHandler()
	clientsHandler := handlers.NewClientsHandler()
	performanceHandler := handlers.NewPerformanceHandler()
	summaryHandler := handlers.NewSummaryHandler()

	// Registrar rutas
	http.HandleFunc("/api/v1/publish", publishHandler.Handle)
	http.HandleFunc("/api/v1/unpublish", unpublishHandler.Handle)
	http.HandleFunc("/api/v1/forward", forwardHandler.Handle)
	http.HandleFunc("/api/v1/stats", statsHandler.Handle)
	http.HandleFunc("/api/v1/clients", clientsHandler.Handle)
	http.HandleFunc("/api/v1/performance", performanceHandler.Handle)
	http.HandleFunc("/api/v1/summary", summaryHandler.Handle)

	port := cfg.Port
	log.Printf("🚀 Backend Go iniciado en puerto %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}