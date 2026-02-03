package lineage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type BuildResult struct {
	Artifacts []string
	Logs      string
}

type Builder struct {
	ssh              *SSHClient
	workDir          string
	repoURL          string
	repoRef          string
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
		repoURL:          cfg.BuildRepoURL,
		repoRef:          cfg.BuildRepoRef,
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
		fmt.Sprintf("rm -rf %s", shellQuote(b.workDir)),
		fmt.Sprintf("git clone %s %s", shellQuote(b.repoURL), shellQuote(b.workDir)),
	}
	if b.repoRef != "" {
		commands = append(commands, fmt.Sprintf("cd %s", shellQuote(b.workDir)), fmt.Sprintf("git checkout %s", shellQuote(b.repoRef)))
	}
	commands = append(commands, fmt.Sprintf("cd %s", shellQuote(b.workDir)))
	commands = append(commands, "docker compose version || docker-compose --version")
	commands = append(commands, fmt.Sprintf("docker compose -f %s pull", shellQuote(b.compose)))
	commands = append(commands, fmt.Sprintf("docker compose -f %s up --build --abort-on-container-exit --exit-code-from build", shellQuote(b.compose)))
	command := strings.Join(commands, " && ")
	return b.runCommand(ctx, command)
}

func (b *Builder) collectArtifacts(ctx context.Context) ([]string, error) {
	command := fmt.Sprintf("cd %s && find %s -maxdepth 1 -type f -name %s -print", shellQuote(b.workDir), shellQuote(b.artifactDir), shellQuote(b.artifactPattern))
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
