package lineage

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultServerType     = "cx41"
	defaultServerImage    = "ubuntu-22.04"
	defaultServerName     = "lineageos-builder"
	defaultComposeFile    = "docker-compose.yml"
	defaultServiceName    = "build"
	defaultWorkingDir     = "lineageos-build"
	defaultSSHPort        = 22
	defaultTimeoutMins    = 360
	defaultArtifactDir    = "zips"
	defaultArtifactGlob   = "*.zip"
	defaultLocalArtifacts = "artifacts"
	defaultStateFile      = ".hetzner-server-state.json"
)

func LoadConfigFromEnv() (Config, error) {
	cfg := Config{
		HetznerToken:        os.Getenv("HETZNER_TOKEN"),
		ServerType:          envOrDefault("HETZNER_SERVER_TYPE", defaultServerType),
		ServerLocation:      os.Getenv("HETZNER_SERVER_LOCATION"),
		ServerImage:         envOrDefault("HETZNER_SERVER_IMAGE", defaultServerImage),
		ServerName:          envOrDefault("HETZNER_SERVER_NAME", defaultServerName),
		ServerUserDataPath:  os.Getenv("HETZNER_SERVER_USER_DATA"),
		BuildSourceDir:      os.Getenv("BUILD_SOURCE_DIR"),
		ComposeFile:         envOrDefault("BUILD_COMPOSE_FILE", defaultComposeFile),
		BuildServiceName:    os.Getenv("BUILD_SERVICE_NAME"),
		WorkingDir:          envOrDefault("BUILD_WORKDIR", defaultWorkingDir),
		ArtifactDir:         envOrDefault("ARTIFACT_DIR", defaultArtifactDir),
		ArtifactPattern:     envOrDefault("ARTIFACT_PATTERN", defaultArtifactGlob),
		LocalArtifactDir:    envOrDefault("LOCAL_ARTIFACT_DIR", defaultLocalArtifacts),
		SSHPort:             envToInt("HETZNER_SSH_PORT", defaultSSHPort),
		BuildTimeoutMinutes: envToInt("BUILD_TIMEOUT_MINUTES", defaultTimeoutMins),
		KeepServerOnFailure: envToBool("KEEP_SERVER_ON_FAILURE"),
		UserSSHKeys:         loadUserSSHKeys(),
		ServerStateFile:     envOrDefault("SERVER_STATE_FILE", defaultStateFile),
	}

	if cfg.HetznerToken == "" {
		return Config{}, fmt.Errorf("HETZNER_TOKEN is required")
	}
	if cfg.BuildSourceDir == "" {
		return Config{}, fmt.Errorf("BUILD_SOURCE_DIR is required")
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envToInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envToBool(key string) bool {
	value := strings.ToLower(os.Getenv(key))
	return value == "true" || value == "1" || value == "yes"
}

func loadUserSSHKeys() []string {
	// Check if explicitly provided via environment variable
	if keys := os.Getenv("USER_SSH_KEYS"); keys != "" {
		return strings.Split(keys, ",")
	}

	// Check if running in GitHub Actions
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		// Try to fetch GitHub user's SSH keys
		if keys := fetchGitHubSSHKeys(); len(keys) > 0 {
			return keys
		}
	}

	// Try to load from user's local SSH directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	var keys []string
	for _, keyFile := range []string{"id_rsa.pub", "id_ed25519.pub", "id_ecdsa.pub"} {
		keyPath := filepath.Join(sshDir, keyFile)
		if content, err := os.ReadFile(keyPath); err == nil {
			key := strings.TrimSpace(string(content))
			if key != "" {
				keys = append(keys, key)
			}
		}
	}

	return keys
}

func fetchGitHubSSHKeys() []string {
	// Get GitHub actor (username)
	actor := os.Getenv("GITHUB_ACTOR")
	if actor == "" {
		return nil
	}

	// Fetch SSH keys from GitHub API
	// Format: https://github.com/{username}.keys
	// We'll use a simple HTTP GET request
	return fetchHTTPKeys(fmt.Sprintf("https://github.com/%s.keys", actor))
}

func fetchHTTPKeys(url string) []string {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		log.Printf("failed to fetch SSH keys from %s: %v", url, err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("failed to fetch SSH keys from %s: status %d", url, resp.StatusCode)
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read SSH keys response: %v", err)
		return nil
	}

	content := strings.TrimSpace(string(body))
	if content == "" {
		return nil
	}

	lines := strings.Split(content, "\n")
	var keys []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			keys = append(keys, line)
		}
	}

	return keys
}
