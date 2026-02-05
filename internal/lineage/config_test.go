package lineage

import (
	"os"
	"testing"
)

func TestConfigLoadingWithNewEnvVars(t *testing.T) {
	t.Parallel()

	// Save original env vars
	origKeep := os.Getenv("KEEP_SERVER_ON_FAILURE")
	origKeys := os.Getenv("USER_SSH_KEYS")
	origState := os.Getenv("SERVER_STATE_FILE")
	origToken := os.Getenv("HETZNER_TOKEN")
	origSource := os.Getenv("BUILD_SOURCE_DIR")

	defer func() {
		os.Setenv("KEEP_SERVER_ON_FAILURE", origKeep)
		os.Setenv("USER_SSH_KEYS", origKeys)
		os.Setenv("SERVER_STATE_FILE", origState)
		os.Setenv("HETZNER_TOKEN", origToken)
		os.Setenv("BUILD_SOURCE_DIR", origSource)
	}()

	// Set test env vars
	os.Setenv("HETZNER_TOKEN", "test-token")
	os.Setenv("BUILD_SOURCE_DIR", "/tmp")
	os.Setenv("KEEP_SERVER_ON_FAILURE", "true")
	os.Setenv("USER_SSH_KEYS", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAATEST test@example.com,ssh-rsa AAAAB3NzaC1yc2ETEST test2@example.com")
	os.Setenv("SERVER_STATE_FILE", "/tmp/test-state.json")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv failed: %v", err)
	}

	if !cfg.KeepServerOnFailure {
		t.Errorf("KeepServerOnFailure should be true, got %v", cfg.KeepServerOnFailure)
	}

	if len(cfg.UserSSHKeys) != 2 {
		t.Errorf("Expected 2 SSH keys, got %d", len(cfg.UserSSHKeys))
	}

	if cfg.ServerStateFile != "/tmp/test-state.json" {
		t.Errorf("Expected ServerStateFile /tmp/test-state.json, got %s", cfg.ServerStateFile)
	}
}

func TestKeepServerOnFailureDefaults(t *testing.T) {
	t.Parallel()

	// Save original env vars
	origToken := os.Getenv("HETZNER_TOKEN")
	origSource := os.Getenv("BUILD_SOURCE_DIR")
	origKeep := os.Getenv("KEEP_SERVER_ON_FAILURE")

	defer func() {
		os.Setenv("HETZNER_TOKEN", origToken)
		os.Setenv("BUILD_SOURCE_DIR", origSource)
		os.Setenv("KEEP_SERVER_ON_FAILURE", origKeep)
	}()

	// Set minimal required vars
	os.Setenv("HETZNER_TOKEN", "test-token")
	os.Setenv("BUILD_SOURCE_DIR", "/tmp")
	os.Setenv("KEEP_SERVER_ON_FAILURE", "")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv failed: %v", err)
	}

	if cfg.KeepServerOnFailure {
		t.Errorf("KeepServerOnFailure should default to false, got true")
	}

	if cfg.ServerStateFile != defaultStateFile {
		t.Errorf("Expected default ServerStateFile %s, got %s", defaultStateFile, cfg.ServerStateFile)
	}
}
