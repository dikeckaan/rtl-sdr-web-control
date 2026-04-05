package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kaandikec/sdr/internal/auth"
	"github.com/kaandikec/sdr/internal/config"
	"github.com/kaandikec/sdr/internal/db"
	"github.com/kaandikec/sdr/internal/docker"
	"github.com/kaandikec/sdr/internal/process"
	"github.com/kaandikec/sdr/internal/sdr"
	"github.com/kaandikec/sdr/internal/server"
	"github.com/kaandikec/sdr/internal/services"
	"github.com/kaandikec/sdr/internal/services/ais"
	"github.com/kaandikec/sdr/internal/services/airband"
	"github.com/kaandikec/sdr/internal/services/dockersvc"
	"github.com/kaandikec/sdr/internal/services/fmradio"
	"github.com/kaandikec/sdr/internal/services/gsm"
	"github.com/kaandikec/sdr/internal/services/hamradio"
	"github.com/kaandikec/sdr/internal/services/iss"
	"github.com/kaandikec/sdr/internal/services/pager"
	"github.com/kaandikec/sdr/internal/ws"
)

func main() {
	configPath := flag.String("config", "config.toml", "Path to configuration file")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("=== SDR Management Platform ===")

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Printf("[config] Loaded from %s", *configPath)
	log.Printf("[config] Station: %s (%.4f, %.4f)", cfg.Station.Name, cfg.Station.Latitude, cfg.Station.Longitude)

	// Open database
	database, err := db.Open(cfg.Server.DataDir)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()
	log.Printf("[db] Opened at %s/sdr.db", cfg.Server.DataDir)

	// Initialize components
	authMgr := auth.NewManager(cfg.Auth.SessionSecret)
	hub := ws.NewHub()
	procMgr := process.NewManager()
	deviceLock := sdr.NewDeviceLock()
	dockerClient := docker.NewClient()
	registry := services.NewRegistry(deviceLock)

	// Get base directory for service paths
	baseDir, _ := os.Getwd()

	// Register native services
	fmSvc := fmradio.New(cfg.FMRadio, cfg.SDR, procMgr)
	registry.Register(fmSvc)

	hamSvc := hamradio.New(cfg.Ham, cfg.SDR, procMgr)
	registry.Register(hamSvc)

	airbandSvc := airband.New(cfg.Airband, cfg.SDR, procMgr)
	registry.Register(airbandSvc)

	pagerSvc := pager.New(cfg.Pager, cfg.SDR, procMgr, database, hub)
	registry.Register(pagerSvc)

	aisSvc := ais.New(cfg.AIS, cfg.SDR, procMgr, database, hub)
	registry.Register(aisSvc)

	issSvc := iss.New(cfg.ISS, cfg.SDR, cfg.Station, procMgr, database, hub, cfg.Server.DataDir)
	registry.Register(issSvc)

	gsmSvc := gsm.New(cfg.GSM, cfg.SDR, procMgr, database)
	registry.Register(gsmSvc)

	// Register Docker services
	for id, dcfg := range cfg.Docker {
		svc := dockersvc.New(id, dcfg, dockerClient, baseDir)
		registry.Register(svc)
	}

	// Create HTTP server
	srv := server.New(cfg, database, authMgr, hub, registry)

	// Register service-specific routes
	srv.RegisterServiceRoutes(fmSvc)
	srv.RegisterServiceRoutes(hamSvc)
	srv.RegisterServiceRoutes(airbandSvc)
	srv.RegisterServiceRoutes(pagerSvc)
	srv.RegisterServiceRoutes(aisSvc)
	srv.RegisterServiceRoutes(issSvc)
	srv.RegisterServiceRoutes(gsmSvc)

	// Serve embedded frontend (or dev proxy)
	srv.ServeFrontend()

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Printf("Received %s, shutting down...", sig)
		procMgr.StopAll()
		os.Exit(0)
	}()

	// Start server
	log.Fatal(srv.Start())
}
