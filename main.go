package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sitewatch/cmd/server" 
	"sitewatch/internal/config"
	"sitewatch/internal/logger"
	"sitewatch/internal/middleware"
	"sitewatch/internal/services/ping"
	"sitewatch/internal/models"
)

func main() {
	// Initialize structured logging first
	logger.InitDefault()
	log := logger.Default().WithComponent("main")
	
	log.Info("🚀 Starting SiteWatch")

	// Initialize application state
	config.GlobalAppState = &config.AppState{
		SiteStatus: make(map[string]*models.SiteStatus),
		StartTime:  time.Now(),
		ResultChan: make(chan models.PingResult, 100),
	}
	appState := config.GlobalAppState

	// Load configuration
	if err := appState.LoadConfig(); err != nil {
		log.Error("Failed to load config", "error", err)
		os.Exit(1)
	}
	log.Info("✅ Configuration loaded")

	// Load sites
	if err := appState.LoadSites(); err != nil {
		log.Error("Failed to load sites", "error", err)
		os.Exit(1)
	}
	log.Info("✅ Sites loaded", "count", len(appState.Sites))

	// Initialize storage
	if err := appState.InitStorage(); err != nil {
		log.Error("Failed to initialize storage", "error", err)
		os.Exit(1)
	}

	// Initialize site status
	appState.InitializeSiteStatus()
	log.Info("✅ Application state initialized")

	// Start ping workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ping.StartPingWorkers(ctx, appState)
	log.Info("✅ Ping workers started")
	
	// Start metrics updater
	middleware.StartMetricsUpdater(30 * time.Second)
	log.Info("✅ Metrics updater started")

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start server
	srv := server.SetupFiberApp(appState)
	go func() {
		addr := fmt.Sprintf("%s:%d", appState.Config.Server.Host, appState.Config.Server.Port)
		log.Info("🌐 Server starting", "address", addr)
		if err := srv.Listen(addr); err != nil {
			log.Error("Server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	<-c
	log.Info("🛑 Shutdown signal received")

	// Cancel context to stop workers
	cancel()

	// Shutdown server gracefully
	log.Info("⏳ Shutting down server")
	if err := srv.Shutdown(); err != nil {
		log.Error("Server shutdown error", "error", err)
	}

	// Close storage backend
	if appState.Storage != nil {
		if storageImpl, ok := appState.Storage.(interface{ Close() error }); ok {
			if err := storageImpl.Close(); err != nil {
				log.Error("Storage close error", "error", err)
			} else {
				log.Info("✅ Storage closed")
			}
		}
	}

	// Close result channel
	if appState.ResultChan != nil {
		close(appState.ResultChan)
		log.Info("✅ Result channel closed")
	}

	log.Info("👋 SiteWatch stopped")
}
