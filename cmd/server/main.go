package main

import (
	"log"
	"net/http"

	"srs-backend/internal/config"
	"srs-backend/internal/handlers"
	"srs-backend/internal/services"
)

func main() {
	// Inicializar configuración
	cfg := config.New()

	log.Printf("🆔 Server ID: %s", cfg.ServerID)
	log.Printf("🌐 Server IP: %s", cfg.ServerIP)

	// Inicializar servicios
	supabaseService := services.NewSupabaseService(cfg.SupabaseURL, cfg.SupabaseKey)
	thumbnailService := services.NewThumbnailService()

	// ✅ Registrar servidor en BD
	if err := supabaseService.RegisterServer(cfg.ServerID, cfg.ServerIP); err != nil {
		log.Printf("⚠️ Error registrando servidor: %v", err)
	} else {
		log.Printf("✅ Servidor %s registrado en base de datos", cfg.ServerID)
	}

	// ✅ CORREGIDO: Pasar serverID y serverIP
	metricsCollector := services.NewMetricsCollector(supabaseService, cfg.ServerID, cfg.ServerIP)

	// Iniciar recolector de métricas en background
	go metricsCollector.Start()

	// Inicializar handlers
	publishHandler := handlers.NewPublishHandler(supabaseService, thumbnailService)
	unpublishHandler := handlers.NewUnpublishHandler(supabaseService, thumbnailService)
	// Cambio: handler para sesiones on_play/on_stop (Firma: Cursor)
	sessionsHandler := handlers.NewSessionsHandler(supabaseService, cfg.ServerID, cfg.ServerIP)
	forwardHandler := handlers.NewForwardHandler(cfg.TargetForwardURL)
	statsHandler := handlers.NewStatsHandler()
	clientsHandler := handlers.NewClientsHandler()
	performanceHandler := handlers.NewPerformanceHandler()
	summaryHandler := handlers.NewSummaryHandler()

	// Registrar rutas
	http.HandleFunc("/api/v1/publish", publishHandler.Handle)
	http.HandleFunc("/api/v1/unpublish", unpublishHandler.Handle)
	// Cambio: ruta para callbacks de sesiones (Firma: Cursor)
	http.HandleFunc("/api/v1/sessions", sessionsHandler.Handle)
	http.HandleFunc("/api/v1/forward", forwardHandler.Handle)
	http.HandleFunc("/api/v1/stats", statsHandler.Handle)
	http.HandleFunc("/api/v1/clients", clientsHandler.Handle)
	http.HandleFunc("/api/v1/performance", performanceHandler.Handle)
	http.HandleFunc("/api/v1/summary", summaryHandler.Handle)

	port := cfg.Port
	log.Printf("🚀 Backend Go iniciado en puerto %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}