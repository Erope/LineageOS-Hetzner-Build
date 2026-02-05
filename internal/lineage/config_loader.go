package lineage

import (
	"fmt"
	"os"
	"strconv"
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
	defaultServerStatePath = ".hetzner-server-state.json"
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
		KeepServerOnFailure: envToBool("KEEP_SERVER_ON_FAILURE", false),
		ServerStatePath:     envOrDefault("SERVER_STATE_PATH", defaultServerStatePath),
	}

	if cfg.HetznerToken == "" {
		return Config{}, fmt.Errorf("HETZNER_TOKEN is required")
	}
	if cfg.BuildSourceDir == "" {
		return Config{}, fmt.Errorf("BUILD_SOURCE_DIR is required")
	}

	return cfg, nil
}

// EnvOrDefault returns the value of the environment variable or a fallback if not set
func EnvOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// envOrDefault is a private alias for backward compatibility within this package
func envOrDefault(key, fallback string) string {
	return EnvOrDefault(key, fallback)
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

func envToBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
