package lineage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type BuildResult struct {
	Artifacts []string
	Logs      string
}

type Builder struct {
	ssh              *SSHClient
	workDir          string
	compose          string
	artifactDir      string
	artifactPattern  string
	localArtifactDir string
	logs             []string
}

func NewBuilder(ssh *SSHClient, cfg Config) *Builder {
	return &Builder{
		ssh:              ssh,
		workDir:          cfg.WorkingDir,
		compose:          cfg.ComposeFile,
		artifactDir:      cfg.ArtifactDir,
		artifactPattern:  cfg.ArtifactPattern,
		localArtifactDir: cfg.LocalArtifactDir,
	}
}

func (b *Builder) Run(ctx context.Context) (BuildResult, error) {
	if err := b.prepare(ctx); err != nil {
		return BuildResult{Logs: b.joinLogs()}, err
	}
	artifacts, err := b.collectArtifacts(ctx)
	if err != nil {
		return BuildResult{Logs: b.joinLogs()}, err
	}
	return BuildResult{Artifacts: artifacts, Logs: b.joinLogs()}, nil
}

func (b *Builder) prepare(ctx context.Context) error {
	commands := []string{
		"set -euo pipefail",
	}
	commands = append(commands, fmt.Sprintf("cd %s", shellQuote(b.workDir)))
	commands = append(commands, "docker compose version || docker-compose --version")
	commands = append(commands, fmt.Sprintf("docker compose -f %s pull", shellQuote(b.compose)))
	commands = append(commands, fmt.Sprintf("docker compose -f %s up --build --abort-on-container-exit --exit-code-from build", shellQuote(b.compose)))
	command := strings.Join(commands, " && ")
	return b.runCommand(ctx, command)
}

func (b *Builder) StageRepository(ctx context.Context, archivePath string) error {
	file, err := os.Open(filepath.Clean(archivePath))
	if err != nil {
		return fmt.Errorf("open repository archive: %w", err)
	}
	defer file.Close()

	remoteArchive := fmt.Sprintf("/tmp/lineage-repo-%d.tar.gz", time.Now().UnixNano())
	if err := b.ssh.Upload(ctx, remoteArchive, file, 0o600); err != nil {
		return fmt.Errorf("upload repository archive: %w", err)
	}

	commands := []string{
		"set -euo pipefail",
		fmt.Sprintf("rm -rf %s", shellQuote(b.workDir)),
		fmt.Sprintf("mkdir -p %s", shellQuote(b.workDir)),
		fmt.Sprintf("tar -xzf %s -C %s", shellQuote(remoteArchive), shellQuote(b.workDir)),
		fmt.Sprintf("rm -f %s", shellQuote(remoteArchive)),
	}
	command := strings.Join(commands, " && ")
	return b.runCommand(ctx, command)
}

func (b *Builder) collectArtifacts(ctx context.Context) ([]string, error) {
	command := fmt.Sprintf("cd %s && find %s -maxdepth 2 -type f -name %s -print", shellQuote(b.workDir), shellQuote(b.artifactDir), shellQuote(b.artifactPattern))
	stdout, _, err := b.ssh.Run(ctx, command)
	b.logs = append(b.logs, stdout)
	if err != nil {
		return nil, fmt.Errorf("list artifacts: %w", err)
	}
	files := strings.Fields(strings.TrimSpace(stdout))
	if len(files) == 0 {
		return nil, fmt.Errorf("no artifacts matched %s/%s", b.artifactDir, b.artifactPattern)
	}
	return files, nil
}

func (b *Builder) DownloadArtifacts(ctx context.Context, files []string) ([]string, error) {
	if b.localArtifactDir == "" {
		return nil, fmt.Errorf("LOCAL_ARTIFACT_DIR is required")
	}
	if err := os.MkdirAll(b.localArtifactDir, 0o755); err != nil {
		return nil, fmt.Errorf("create artifact dir: %w", err)
	}
	localPaths := make([]string, 0, len(files))
	for _, remote := range files {
		filename := filepath.Base(remote)
		localPath := filepath.Join(b.localArtifactDir, filename)
		if err := b.ssh.Download(ctx, remote, localPath); err != nil {
			return nil, fmt.Errorf("download artifact %s: %w", remote, err)
		}
		localPaths = append(localPaths, localPath)
	}
	return localPaths, nil
}

func (b *Builder) SaveRemoteLogs(ctx context.Context) (string, error) {
	command := fmt.Sprintf("cd %s && docker compose -f %s logs --no-color", shellQuote(b.workDir), shellQuote(b.compose))
	stdout, _, err := b.ssh.Run(ctx, command)
	b.logs = append(b.logs, stdout)
	return stdout, err
}

func (b *Builder) joinLogs() string {
	return strings.Join(b.logs, "\n")
}

func (b *Builder) runCommand(ctx context.Context, command string) error {
	stdout, stderr, err := b.ssh.Run(ctx, command)
	b.logs = append(b.logs, stdout)
	if stderr != "" {
		b.logs = append(b.logs, stderr)
	}
	if err != nil {
		return fmt.Errorf("remote command failed: %w", err)
	}
	return nil
}
