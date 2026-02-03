package lineage

import (
	"fmt"
	"net/url"
	"os"
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
	defaultArtifactDir    = "out/target/product"
	defaultArtifactGlob   = "*.zip"
	defaultLocalArtifacts = "artifacts"
	defaultGitHost        = "github.com"
)

func LoadConfigFromEnv() (Config, error) {
	cfg := Config{
		HetznerToken:        os.Getenv("HETZNER_TOKEN"),
		ServerType:          envOrDefault("HETZNER_SERVER_TYPE", defaultServerType),
		ServerLocation:      os.Getenv("HETZNER_SERVER_LOCATION"),
		ServerImage:         envOrDefault("HETZNER_SERVER_IMAGE", defaultServerImage),
		ServerName:          envOrDefault("HETZNER_SERVER_NAME", defaultServerName),
		ServerUserDataPath:  os.Getenv("HETZNER_SERVER_USER_DATA"),
		BuildRepoURL:        os.Getenv("BUILD_REPO_URL"),
		BuildRepoRef:        os.Getenv("BUILD_REPO_REF"),
		BuildRepoSHA:        os.Getenv("GITHUB_SHA"),
		ComposeFile:         envOrDefault("BUILD_COMPOSE_FILE", defaultComposeFile),
		WorkingDir:          envOrDefault("BUILD_WORKDIR", defaultWorkingDir),
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
	if cfg.GitHubToken == "" {
		return Config{}, fmt.Errorf("GITHUB_TOKEN is required")
	}
	host, owner, name, err := parseGitHubRepo(cfg.BuildRepoURL)
	if err != nil {
		return Config{}, err
	}
	cfg.BuildRepoHost = host
	cfg.BuildRepoOwner = owner
	cfg.BuildRepoName = name

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

func parseGitHubRepo(repoURL string) (string, string, string, error) {
	if repoURL == "" {
		return "", "", "", fmt.Errorf("BUILD_REPO_URL is empty")
	}
	if strings.HasPrefix(repoURL, "git@") {
		trimmed := strings.TrimPrefix(repoURL, "git@")
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			return "", "", "", fmt.Errorf("invalid BUILD_REPO_URL: %s", repoURL)
		}
		host := parts[0]
		path := strings.TrimSuffix(parts[1], ".git")
		pathParts := strings.Split(strings.Trim(path, "/"), "/")
		if !isPublicGitHubHost(host) || len(pathParts) != 2 {
			return "", "", "", fmt.Errorf("invalid BUILD_REPO_URL: %s", repoURL)
		}
		return host, pathParts[0], pathParts[1], nil
	}

	normalized := repoURL
	if !strings.Contains(normalized, "://") {
		normalized = "https://" + normalized
	}
	parsed, err := url.Parse(normalized)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid BUILD_REPO_URL: %s", repoURL)
	}
	host := parsed.Host
	path := strings.TrimSuffix(parsed.Path, ".git")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if !isPublicGitHubHost(host) || len(pathParts) != 2 {
		return "", "", "", fmt.Errorf("invalid BUILD_REPO_URL: %s", repoURL)
	}
	return host, pathParts[0], pathParts[1], nil
}

func isPublicGitHubHost(host string) bool {
	return host == defaultGitHost
}
