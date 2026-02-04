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
	cfg           Config
}

func NewOrchestrator(cfg Config) *Orchestrator {
	return &Orchestrator{
		hetznerClient: NewHetznerClient(cfg.HetznerToken),
		cfg:           cfg,
	}
}

func (o *Orchestrator) Run(ctx context.Context) error {
	progress := newStageLogger(7)
	progress.Step("prepare source archive")
	archivePath, cleanup, err := PrepareRepositoryArchive(ctx, o.cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	progress.Step("create Hetzner server")
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

	progress.Step("wait for server to be running")
	if err := o.hetznerClient.WaitForServer(ctx, server.ID); err != nil {
		return fmt.Errorf("wait for server: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", server.IP, server.SSHPort)
	progress.Step("wait for SSH to become available")
	if err := waitForPort(ctx, addr, 5*time.Minute); err != nil {
		return fmt.Errorf("wait for ssh: %w", err)
	}

	// known_hosts is deferred until rescue exit so we can tolerate the rescue host key.
	sshClient, err := NewSSHClient(addr, server.SSHUser, server.SSHKey, "", 30*time.Second)
	if err != nil {
		return err
	}
	// Rescue mode can persist several minutes while the instance reboots into the final OS.
	const rescueExitTimeout = 8 * time.Minute
	if err := waitForRescueExit(ctx, sshClient, rescueExitTimeout); err != nil {
		return err
	}
	knownHostsPath, err := ensureKnownHosts(server.IP, server.SSHPort, o.cfg.LocalArtifactDir)
	if err != nil {
		return err
	}
	sshClient.KnownHosts = knownHostsPath
	builder := NewBuilder(sshClient, o.cfg)
	buildCtx, cancel := context.WithTimeout(ctx, time.Duration(o.cfg.BuildTimeoutMinutes)*time.Minute)
	defer cancel()

	progress.Step("stage source on server")
	if err := builder.StageSource(buildCtx, archivePath); err != nil {
		return err
	}
	progress.Step("run build on server")
	result, err := builder.Run(buildCtx)
	if err != nil {
		log.Printf("build failed, collecting remote logs")
		logs, logErr := builder.SaveRemoteLogs(ctx)
		if logErr == nil {
			_ = saveLogs(o.cfg, sanitizeLog(logs))
		}
		return fmt.Errorf("build failed: %w", err)
	}

	progress.Step("download artifacts")
	artifacts, err := builder.DownloadArtifacts(ctx, result.Artifacts)
	if err != nil {
		return err
	}
	log.Printf("downloaded %d artifacts", len(artifacts))

	return nil
}

type stageLogger struct {
	total    int
	current  int
	barWidth int
}

func newStageLogger(total int) *stageLogger {
	return &stageLogger{
		total:    total,
		barWidth: 20,
	}
}

func (s *stageLogger) Step(message string) {
	if s.total <= 0 {
		log.Printf("%s", message)
		return
	}
	if s.current < s.total {
		s.current++
	}
	filled := s.current * s.barWidth / s.total
	if filled > s.barWidth {
		filled = s.barWidth
	}
	bar := fmt.Sprintf("[%s%s]", strings.Repeat("#", filled), strings.Repeat("-", s.barWidth-filled))
	percent := s.current * 100 / s.total
	log.Printf("%s %d/%d %3d%% %s", bar, s.current, s.total, percent, message)
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

func waitForRescueExit(ctx context.Context, sshClient *SSHClient, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		hostname, _, hostErr := sshClient.Run(ctx, "hostname")
		if hostErr != nil {
			if time.Now().After(deadline) {
				return hostErr
			}
			if err := sleepWithContext(ctx, 10*time.Second); err != nil {
				return err
			}
			continue
		}
		// Expect Linux with coreutils df output in Hetzner rescue/ubuntu images.
		rootFs, _, rootErr := sshClient.Run(ctx, "df -T /")
		if rootErr == nil {
			if !isRescueHostname(hostname) && !isRescueRootFilesystem(rootFs) {
				return nil
			}
		}
		if time.Now().After(deadline) {
			if hostErr != nil {
				return hostErr
			}
			if rootErr != nil {
				return rootErr
			}
			return fmt.Errorf("timeout waiting for rescue system to exit")
		}
		if err := sleepWithContext(ctx, 10*time.Second); err != nil {
			return err
		}
	}
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
