package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/Erope/LineageOS-Hetzner-Build/internal/lineage"
)

func main() {
	cleanup := flag.Bool("cleanup", false, "cleanup server from saved state file")
	flag.Parse()

	if *cleanup {
		runCleanup()
		return
	}

	runBuild()
}

func runBuild() {
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

func runCleanup() {
	log.Printf("cleanup mode: destroying server from state file")

	stateFile := os.Getenv("SERVER_STATE_FILE")
	if stateFile == "" {
		stateFile = ".hetzner-server-state.json"
	}

	// Check if state file exists
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		log.Printf("no server state file found at %s", stateFile)
		log.Printf("nothing to cleanup")
		return
	}

	// Check KEEP_SERVER_ON_FAILURE setting
	keepOnFailure := os.Getenv("KEEP_SERVER_ON_FAILURE")
	if keepOnFailure == "true" || keepOnFailure == "1" || keepOnFailure == "yes" {
		log.Printf("WARNING: KEEP_SERVER_ON_FAILURE is set to true")
		log.Printf("This indicates you wanted to preserve the server for debugging")
		log.Printf("Are you sure you want to destroy it?")
		log.Printf("If yes, unset KEEP_SERVER_ON_FAILURE and run cleanup again")
		log.Printf("Example: KEEP_SERVER_ON_FAILURE=false go run ./cmd/lineage-builder --cleanup")
		os.Exit(1)
	}

	if err := lineage.CleanupServerFromState(stateFile); err != nil {
		log.Printf("cleanup failed: %v", err)
		os.Exit(1)
	}

	log.Printf("server cleanup completed successfully")
}
