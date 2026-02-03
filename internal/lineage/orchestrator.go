package lineage

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Orchestrator struct {
	hetznerClient *HetznerClient
	githubClient  *GitHubReleaseClient
	cfg           Config
}

func NewOrchestrator(cfg Config) *Orchestrator {
	return &Orchestrator{
		hetznerClient: NewHetznerClient(cfg.HetznerToken),
		githubClient:  NewGitHubReleaseClient(cfg.GitHubToken),
		cfg:           cfg,
	}
}

func (o *Orchestrator) Run(ctx context.Context) error {
	server, err := o.hetznerClient.CreateServer(ctx, o.cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := o.hetznerClient.DeleteServer(context.Background(), server.ID); err != nil {
			log.Printf("failed to delete server %d: %v", server.ID, err)
		}
		if err := o.hetznerClient.DeleteSSHKey(context.Background(), server.SSHKeyID); err != nil {
			log.Printf("failed to delete ssh key %d: %v", server.SSHKeyID, err)
		}
	}()

	if err := o.hetznerClient.WaitForServer(ctx, server.ID); err != nil {
		return fmt.Errorf("wait for server: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", server.IP, server.SSHPort)
	if err := waitForPort(ctx, addr, 5*time.Minute); err != nil {
		return fmt.Errorf("wait for ssh: %w", err)
	}

	knownHostsPath, err := ensureKnownHosts(server.IP, server.SSHPort, o.cfg.LocalArtifactDir)
	if err != nil {
		return err
	}
	sshClient, err := NewSSHClient(addr, server.SSHUser, server.SSHKey, knownHostsPath, 30*time.Second)
	if err != nil {
		return err
	}

	builder := NewBuilder(sshClient, o.cfg)
	buildCtx, cancel := context.WithTimeout(ctx, time.Duration(o.cfg.BuildTimeoutMinutes)*time.Minute)
	defer cancel()

	result, err := builder.Run(buildCtx)
	if err != nil {
		logs, logErr := builder.SaveRemoteLogs(ctx)
		if logErr == nil {
			_ = saveLogs(o.cfg, sanitizeLog(logs))
		}
		return fmt.Errorf("build failed: %w", err)
	}

	artifacts, err := builder.DownloadArtifacts(ctx, result.Artifacts)
	if err != nil {
		return err
	}

	if err := o.githubClient.UploadArtifacts(ctx, o.cfg, artifacts); err != nil {
		return err
	}

	return nil
}

func saveLogs(cfg Config, logs string) error {
	if cfg.LocalArtifactDir == "" {
		return nil
	}
	if err := os.MkdirAll(cfg.LocalArtifactDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(cfg.LocalArtifactDir, "build.log")
	return os.WriteFile(path, []byte(logs), 0o600)
}

func sanitizeLog(input string) string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(github_token\s*=\s*)\S+`),
		regexp.MustCompile(`(?i)(hetzner_token\s*=\s*)\S+`),
		regexp.MustCompile(`(?i)(authorization:\s*(bearer|token)\s+)\S+`),
		regexp.MustCompile(`(?i)(token\s*=\s*)\S+`),
	}
	lines := strings.Split(input, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		redacted := line
		for _, pattern := range patterns {
			redacted = pattern.ReplaceAllString(redacted, "${1}REDACTED")
		}
		filtered = append(filtered, redacted)
	}
	return strings.Join(filtered, "\n")
}
