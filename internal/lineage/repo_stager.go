package lineage

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func PrepareRepositoryArchive(ctx context.Context, cfg Config) (string, func(), error) {
	if cfg.BuildSourceDir == "" {
		return "", nil, fmt.Errorf("BUILD_SOURCE_DIR is required")
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
	suffix, err := randomSuffix()
	if err != nil {
		_ = os.RemoveAll(stagingDir)
		return "", nil, err
	}
	archivePath := filepath.Join(baseDir, fmt.Sprintf("lineage-repo-%s.tar.gz", suffix))
	cleanup := func() {
		_ = os.RemoveAll(stagingDir)
		_ = os.Remove(archivePath)
	}
	if err := stageSourceDirectory(ctx, cfg.BuildSourceDir, stagingDir); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := createRepoArchive(ctx, stagingDir, archivePath); err != nil {
		cleanup()
		return "", nil, err
	}
	return archivePath, cleanup, nil
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

func stageSourceDirectory(ctx context.Context, sourceDir, dest string) error {
	if _, err := os.Stat(sourceDir); err != nil {
		return fmt.Errorf("check BUILD_SOURCE_DIR: %w", err)
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}
	return runLocalCommand(ctx, "cp", "-a", filepath.Join(sourceDir, "."), dest)
}
