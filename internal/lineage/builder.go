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
	compose          string
	artifactDir      string
	artifactPattern  string
	localArtifactDir string
	logs             []string
}

const commandLogPrefix = ">>>"

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
	if err := b.runCompose(ctx); err != nil {
		return BuildResult{Logs: b.joinLogs()}, err
	}
	artifacts, err := b.collectArtifacts(ctx)
	if err != nil {
		return BuildResult{Logs: b.joinLogs()}, err
	}
	return BuildResult{Artifacts: artifacts, Logs: b.joinLogs()}, nil
}

func (b *Builder) runCompose(ctx context.Context) error {
	commands := []string{
		"set -euo pipefail",
	}
	commands = append(commands, dockerInstallCommand())
	commands = append(commands, fmt.Sprintf("cd %s", shellQuote(b.workDir)))
	commands = append(commands, "docker compose version")
	commands = append(commands, fmt.Sprintf("docker compose -f %s pull", shellQuote(b.compose)))
	commands = append(commands, fmt.Sprintf("docker compose -f %s up --build --abort-on-container-exit --exit-code-from build", shellQuote(b.compose)))
	command := strings.Join(commands, " && ")
	return b.runCommand(ctx, command)
}

// dockerInstallCommand returns a shell script that ensures Docker and the
// Docker Compose plugin are installed before running the build commands.
// The script assumes a Debian/Ubuntu-based image with apt-get and root access.
func dockerInstallCommand() string {
	return strings.TrimSpace(`
install_docker_packages() {
  if [ "$(id -u)" -ne 0 ]; then
    echo 'root privileges are required to install Docker; ensure the build server runs as root' >&2
    exit 1
  fi
  if ! command -v curl >/dev/null 2>&1; then
    echo 'curl is required to install Docker; set HETZNER_SERVER_IMAGE to a Debian/Ubuntu image that includes curl' >&2
    exit 1
  fi
  curl -fsSL https://get.docker.com -o /tmp/get-docker.sh || { echo 'failed to download get.docker.com installer' >&2; exit 1; }
  if [ -n "${GET_DOCKER_SHA256:-}" ]; then
    if ! command -v sha256sum >/dev/null 2>&1; then
      echo 'sha256sum is required to verify GET_DOCKER_SHA256; install coreutils or unset GET_DOCKER_SHA256' >&2
      exit 1
    fi
    echo "${GET_DOCKER_SHA256}  /tmp/get-docker.sh" | sha256sum -c - >/dev/null 2>&1 || { echo 'get.docker.com checksum verification failed' >&2; exit 1; }
  else
    echo 'warning: executing get.docker.com installer without checksum verification; review the script for production use' >&2
  fi
  sh /tmp/get-docker.sh || { echo 'Docker install failed; check network connectivity and repository configuration' >&2; exit 1; }
  rm -f /tmp/get-docker.sh
}

docker_compose_available() {
  docker compose version >/dev/null 2>&1
}

if ! command -v docker >/dev/null 2>&1; then
  install_docker_packages
elif ! docker_compose_available; then
  install_docker_packages
fi`)
}

func (b *Builder) StageSource(ctx context.Context, archivePath string) error {
	file, err := os.Open(filepath.Clean(archivePath))
	if err != nil {
		return fmt.Errorf("open source archive: %w", err)
	}
	defer file.Close()

	suffix, err := randomSuffix()
	if err != nil {
		return err
	}
	remoteArchive := fmt.Sprintf("/tmp/lineage-repo-%s.tar.gz", suffix)
	if err := b.ssh.Upload(ctx, remoteArchive, file, 0o600); err != nil {
		return fmt.Errorf("upload source archive: %w", err)
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
	stdout, stderr, err := b.ssh.Run(ctx, command)
	b.logs = append(b.logs, stdout)
	if stderr != "" {
		b.logs = append(b.logs, stderr)
	}
	return joinLogParts(stdout, stderr), err
}

func (b *Builder) joinLogs() string {
	return strings.Join(b.logs, "\n")
}

// joinLogParts trims log values and joins the non-empty parts with newlines.
// It is used across the lineage package to normalize log output and is
// intentionally package-level for reuse.
func joinLogParts(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, "\n")
}

func (b *Builder) runCommand(ctx context.Context, command string) error {
	b.logs = append(b.logs, fmt.Sprintf("%s %s", commandLogPrefix, command))
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
