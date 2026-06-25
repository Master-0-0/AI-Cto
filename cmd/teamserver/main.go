package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"c2/internal/teamserver"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "Path to configuration file")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[C2] ")

	// Load configuration
	cfg, err := teamserver.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create server
	srv, err := teamserver.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Save config (in case secrets were auto-generated)
	if err := cfg.Save(*configPath); err != nil {
		log.Printf("[WARN] Failed to save config: %v", err)
	}

	// Set up signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Run server
	if err := srv.Run(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
