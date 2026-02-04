package lineage

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func PrepareRepositoryArchive(ctx context.Context, cfg Config) (string, func(), error) {
	if cfg.BuildRepoURL == "" {
		return "", nil, fmt.Errorf("BUILD_REPO_URL is required")
	}
	baseDir := cfg.LocalArtifactDir
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create local artifact dir: %w", err)
	}
	stagingDir, err := os.MkdirTemp(baseDir, "repo-staging-")
	if err != nil {
		return "", nil, fmt.Errorf("create staging dir: %w", err)
	}
	archivePath := filepath.Join(baseDir, fmt.Sprintf("lineage-repo-%d.tar.gz", time.Now().UnixNano()))
	cleanup := func() {
		_ = os.RemoveAll(stagingDir)
		_ = os.Remove(archivePath)
	}
	if err := cloneRepository(ctx, cfg, stagingDir); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := createRepoArchive(ctx, stagingDir, archivePath); err != nil {
		cleanup()
		return "", nil, err
	}
	return archivePath, cleanup, nil
}

func cloneRepository(ctx context.Context, cfg Config, dest string) error {
	token := strings.TrimSpace(cfg.BuildRepoToken)
	repoURL := cfg.BuildRepoURL
	var args []string
	if token == "" {
		args = []string{"clone", repoURL, dest}
	} else {
		normalized, err := normalizeRepoURL(repoURL)
		if err != nil {
			return err
		}
		header := buildAuthHeader(token)
		args = []string{"-c", fmt.Sprintf("http.extraheader=%s", header), "clone", normalized, dest}
	}
	if err := runLocalCommand(ctx, "git", args...); err != nil {
		return err
	}
	if cfg.BuildRepoRef != "" {
		if err := runLocalCommand(ctx, "git", "-C", dest, "checkout", cfg.BuildRepoRef); err != nil {
			return err
		}
	}
	return nil
}

func createRepoArchive(ctx context.Context, sourceDir, archivePath string) error {
	return runLocalCommand(ctx, "tar", "-czf", archivePath, "-C", sourceDir, ".")
}

func runLocalCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(output.String())
		if message != "" {
			return fmt.Errorf("%s failed: %w: %s", name, err, message)
		}
		return fmt.Errorf("%s failed: %w", name, err)
	}
	return nil
}

func normalizeRepoURL(repoURL string) (string, error) {
	if strings.HasPrefix(repoURL, "http://") || strings.HasPrefix(repoURL, "https://") {
		return repoURL, nil
	}
	if strings.HasPrefix(repoURL, "git@") {
		trimmed := strings.TrimPrefix(repoURL, "git@")
		segments := strings.SplitN(trimmed, ":", 2)
		if len(segments) != 2 {
			return "", fmt.Errorf("invalid BUILD_REPO_URL: %s", repoURL)
		}
		return fmt.Sprintf("https://%s/%s", segments[0], segments[1]), nil
	}
	return "https://" + strings.TrimPrefix(repoURL, "//"), nil
}

func buildAuthHeader(token string) string {
	payload := fmt.Sprintf("x-access-token:%s", strings.TrimSpace(token))
	encoded := base64.StdEncoding.EncodeToString([]byte(payload))
	return fmt.Sprintf("AUTHORIZATION: basic %s", encoded)
}
