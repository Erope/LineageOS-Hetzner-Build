package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/Erope/LineageOS-Hetzner-Build/internal/lineage"
)

func main() {
	cleanupFlag := flag.Bool("cleanup", false, "cleanup persisted server resources")
	flag.Parse()

	log.Printf("lineage builder starting")

	// Handle cleanup mode first, before full config validation
	if *cleanupFlag {
		log.Printf("running in cleanup mode")
		cfg := lineage.Config{
			HetznerToken:    os.Getenv("HETZNER_TOKEN"),
			ServerStatePath: envOrDefault("SERVER_STATE_PATH", ".hetzner-server-state.json"),
		}

		if cfg.HetznerToken == "" {
			log.Printf("configuration error: HETZNER_TOKEN is required")
			os.Exit(1)
		}

		if err := lineage.CleanupPersistedServer(context.Background(), cfg); err != nil {
			log.Printf("cleanup failed: %v", err)
			os.Exit(1)
		}
		log.Printf("cleanup completed successfully")
		return
	}

	cfg, err := lineage.LoadConfigFromEnv()
	if err != nil {
		log.Printf("configuration error: %v", err)
		os.Exit(1)
	}

	log.Printf("configuration loaded for source directory %s", cfg.BuildSourceDir)
	orchestrator := lineage.NewOrchestrator(cfg)
	if err := orchestrator.Run(context.Background()); err != nil {
		log.Printf("build failed: %v", err)
		os.Exit(1)
	}
	log.Printf("build completed successfully")
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
