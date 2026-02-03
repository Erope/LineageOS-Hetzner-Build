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
	defaultWorkingDir     = "lineageos-build"
	defaultSSHUser        = "root"
	defaultSSHPort        = 22
	defaultTimeoutMins    = 360
	defaultArtifactDir    = "out/target/product"
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
		InstanceSSHUser:     envOrDefault("HETZNER_SSH_USER", defaultSSHUser),
		InstanceSSHKeyPath:  os.Getenv("HETZNER_SSH_KEY"),
		KnownHostsPath:      os.Getenv("HETZNER_KNOWN_HOSTS"),
		BuildRepoURL:        os.Getenv("BUILD_REPO_URL"),
		BuildRepoRef:        os.Getenv("BUILD_REPO_REF"),
		ComposeFile:         envOrDefault("BUILD_COMPOSE_FILE", defaultComposeFile),
		WorkingDir:          envOrDefault("BUILD_WORKDIR", defaultWorkingDir),
		ReleaseRepoOwner:    os.Getenv("RELEASE_REPO_OWNER"),
		ReleaseRepoName:     os.Getenv("RELEASE_REPO_NAME"),
		ReleaseTag:          os.Getenv("RELEASE_TAG"),
		ReleaseName:         os.Getenv("RELEASE_NAME"),
		ReleaseNotes:        os.Getenv("RELEASE_NOTES"),
		GitHubToken:         os.Getenv("GITHUB_TOKEN"),
		ArtifactDir:         envOrDefault("ARTIFACT_DIR", defaultArtifactDir),
		ArtifactPattern:     envOrDefault("ARTIFACT_PATTERN", defaultArtifactGlob),
		LocalArtifactDir:    envOrDefault("LOCAL_ARTIFACT_DIR", defaultLocalArtifacts),
		SSHPort:             envToInt("HETZNER_SSH_PORT", defaultSSHPort),
		BuildTimeoutMinutes: envToInt("BUILD_TIMEOUT_MINUTES", defaultTimeoutMins),
	}

	if cfg.HetznerToken == "" {
		return Config{}, fmt.Errorf("HETZNER_TOKEN is required")
	}
	if cfg.BuildRepoURL == "" {
		return Config{}, fmt.Errorf("BUILD_REPO_URL is required")
	}
	if cfg.InstanceSSHKeyPath == "" {
		return Config{}, fmt.Errorf("HETZNER_SSH_KEY is required")
	}
	if cfg.KnownHostsPath == "" {
		return Config{}, fmt.Errorf("HETZNER_KNOWN_HOSTS is required")
	}
	if cfg.GitHubToken == "" {
		return Config{}, fmt.Errorf("GITHUB_TOKEN is required")
	}
	if cfg.ReleaseRepoOwner == "" || cfg.ReleaseRepoName == "" || cfg.ReleaseTag == "" {
		return Config{}, fmt.Errorf("RELEASE_REPO_OWNER, RELEASE_REPO_NAME, and RELEASE_TAG are required")
	}
	if cfg.ReleaseName == "" {
		cfg.ReleaseName = cfg.ReleaseTag
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
