package lineage

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultServerType     = "cx41"
	defaultServerImage    = "ubuntu-22.04"
	defaultServerName     = "lineageos-builder"
	defaultComposeFile    = "docker-compose.yml"
	defaultWorkingDir     = "lineageos-build"
	defaultSSHPort        = 22
	defaultTimeoutMins    = 360
	defaultArtifactDir    = "zips"
	defaultArtifactGlob   = "*.zip"
	defaultLocalArtifacts = "artifacts"
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
		WorkingDir:          envOrDefault("BUILD_WORKDIR", defaultWorkingDir),
		ArtifactDir:         envOrDefault("ARTIFACT_DIR", defaultArtifactDir),
		ArtifactPattern:     envOrDefault("ARTIFACT_PATTERN", defaultArtifactGlob),
		LocalArtifactDir:    envOrDefault("LOCAL_ARTIFACT_DIR", defaultLocalArtifacts),
		SSHPort:             envToInt("HETZNER_SSH_PORT", defaultSSHPort),
		BuildTimeoutMinutes: envToInt("BUILD_TIMEOUT_MINUTES", defaultTimeoutMins),
	}

	if cfg.HetznerToken == "" {
		return Config{}, fmt.Errorf("HETZNER_TOKEN is required")
	}
	if cfg.BuildSourceDir == "" {
		return Config{}, fmt.Errorf("BUILD_SOURCE_DIR is required")
	}
	buildSourceDir, err := normalizeBuildSourceDir(cfg.BuildSourceDir, os.Getenv("GITHUB_WORKSPACE"))
	if err != nil {
		return Config{}, err
	}
	cfg.BuildSourceDir = buildSourceDir
	composePath, err := normalizeComposeFilePath(cfg.BuildSourceDir, cfg.ComposeFile)
	if err != nil {
		return Config{}, err
	}
	cfg.ComposeFile = composePath
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

func normalizeComposeFilePath(buildSourceDir, composeFile string) (string, error) {
	if composeFile == "" {
		return "", nil
	}
	cleaned := filepath.Clean(composeFile)
	if cleaned == "." || cleaned == ".." {
		return "", fmt.Errorf("BUILD_COMPOSE_FILE must point to a file")
	}
	buildAbs, err := filepath.Abs(buildSourceDir)
	if err != nil {
		return "", fmt.Errorf("resolve BUILD_SOURCE_DIR: %w", err)
	}
	composeAbs := cleaned
	if !filepath.IsAbs(cleaned) {
		composeAbs = filepath.Join(buildAbs, cleaned)
	}
	rel, err := filepath.Rel(buildAbs, composeAbs)
	if err != nil {
		return "", fmt.Errorf("resolve BUILD_COMPOSE_FILE: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("BUILD_COMPOSE_FILE must be within BUILD_SOURCE_DIR")
	}
	return rel, nil
}

func normalizeBuildSourceDir(buildSourceDir, workspace string) (string, error) {
	cleaned := filepath.Clean(buildSourceDir)
	if filepath.IsAbs(cleaned) {
		return cleaned, nil
	}
	if workspace != "" {
		return filepath.Join(workspace, cleaned), nil
	}
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("resolve BUILD_SOURCE_DIR: %w", err)
	}
	return abs, nil
}
