package services

import (
	"log"
	"os"
	"os/exec"
	"sync"
	"time"
)

type ThumbnailService struct {
	activeProcesses map[string]*time.Ticker
	mu              sync.Mutex
}

func NewThumbnailService() *ThumbnailService {
	return &ThumbnailService{
		activeProcesses: make(map[string]*time.Ticker),
	}
}

func (s *ThumbnailService) StartCapture(streamID, appName, fileName, rtmpURL, outputPath string) {
	// Captura inicial
	time.Sleep(5 * time.Second)
	log.Printf("📸 Capturando thumbnail inicial...")
	s.captureThumbnail(rtmpURL, outputPath, fileName)

	// Ticker: Captura cada 2 minutos
	ticker := time.NewTicker(2 * time.Minute)

	s.mu.Lock()
	s.activeProcesses[streamID] = ticker
	s.mu.Unlock()

	log.Printf("⏰ Thumbnail se actualizará cada 2 minutos para %s", streamID)

	// Loop que captura periódicamente
	go func() {
		for range ticker.C {
			log.Printf("🔄 Actualizando thumbnail para %s", streamID)
			s.captureThumbnail(rtmpURL, outputPath, fileName)
		}
	}()
}

func (s *ThumbnailService) StopCapture(streamID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ticker, ok := s.activeProcesses[streamID]; ok {
		ticker.Stop()
		delete(s.activeProcesses, streamID)
		log.Printf("🛑 Ticker detenido para %s", streamID)
	}
}

func (s *ThumbnailService) captureThumbnail(rtmpURL, outputPath, fileName string) {
	cmd := exec.Command("ffmpeg",
		"-y",
		"-i", rtmpURL,
		"-vframes", "1",
		"-q:v", "2",
		outputPath)

	if err := cmd.Run(); err != nil {
		if _, statErr := os.Stat(outputPath); statErr == nil {
			log.Printf("✅ Thumbnail actualizado: %s", fileName)
		} else {
			log.Printf("❌ Error FFmpeg: %v", err)
		}
	} else {
		log.Printf("✅ Thumbnail generado: %s", fileName)
	}
}