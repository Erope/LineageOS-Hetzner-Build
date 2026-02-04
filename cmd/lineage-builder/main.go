package main

import (
	"context"
	"log"
	"os"

	"github.com/Erope/LineageOS-Hetzner-Build/internal/lineage"
)

func main() {
	log.Printf("lineage builder starting")
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
