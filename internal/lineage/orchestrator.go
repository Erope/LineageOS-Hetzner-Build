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
	totalSteps := 7
	if o.cfg.GitHubActions {
		totalSteps++
	}
	progress := newStageLogger(totalSteps)
	var keepServer bool
	var injectedKeyIDs []int64
	progress.Step("prepare source archive")
	archivePath, cleanup, err := PrepareRepositoryArchive(ctx, o.cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	if o.cfg.GitHubActions {
		progress.Step("fetch GitHub SSH keys")
		if o.cfg.GitHubActor == "" {
			return fmt.Errorf("GITHUB_ACTOR is required when GITHUB_ACTIONS is set")
		}
		keys, err := fetchGitHubPublicKeys(ctx, o.cfg.GitHubActor)
		if err != nil {
			return err
		}
		if len(keys) == 0 {
			return fmt.Errorf("no public SSH keys found for GitHub actor %q; ensure SSH keys are configured at https://github.com/settings/keys", o.cfg.GitHubActor)
		}
		timestamp := time.Now().Unix()
		for i, key := range keys {
			name := fmt.Sprintf("lineage-builder-gh-%s-%d-%d", o.cfg.GitHubActor, timestamp, i)
			createdKey, err := o.hetznerClient.CreateSSHKey(ctx, name, key)
			if err != nil {
				return err
			}
			injectedKeyIDs = append(injectedKeyIDs, createdKey.ID)
		}
		log.Printf("registered %d GitHub SSH key(s) for actor %s", len(injectedKeyIDs), o.cfg.GitHubActor)
	}

	progress.Step("create Hetzner server")
	server, err := o.hetznerClient.CreateServer(ctx, o.cfg, injectedKeyIDs)
	if err != nil {
		return err
	}
	log.Printf("server created: id=%d name=%s ip=%s datacenter=%s", server.ID, server.Name, server.IP, server.Datacenter)
	if o.cfg.GitHubActions && len(injectedKeyIDs) > 0 {
		log.Printf("GitHub Actions SSH keys injected for actor %s", o.cfg.GitHubActor)
	}
	if o.cfg.KeepServerOnFailure {
		keepServer = true
	}
	defer func() {
		if keepServer {
			log.Printf("KEEP_SERVER_ON_FAILURE enabled; keeping server %d (ip=%s) for debugging", server.ID, server.IP)
			return
		}
		log.Printf("cleaning up server %d and ssh key %d", server.ID, server.SSHKeyID)
		if err := o.hetznerClient.DeleteServer(context.Background(), server.ID); err != nil {
			log.Printf("failed to delete server %d: %v", server.ID, err)
		}
		if err := o.hetznerClient.DeleteSSHKey(context.Background(), server.SSHKeyID); err != nil {
			log.Printf("failed to delete ssh key %d: %v", server.SSHKeyID, err)
		}
		for _, keyID := range injectedKeyIDs {
			if err := o.hetznerClient.DeleteSSHKey(context.Background(), keyID); err != nil {
				log.Printf("failed to delete ssh key %d: %v", keyID, err)
			}
		}
	}()

	progress.Step("wait for server to be running")
	log.Printf("waiting for server %d to reach running status...", server.ID)
	if err := o.hetznerClient.WaitForServer(ctx, server.ID); err != nil {
		return fmt.Errorf("wait for server: %w", err)
	}
	log.Printf("server %d is running", server.ID)

	addr := fmt.Sprintf("%s:%d", server.IP, server.SSHPort)
	progress.Step("wait for SSH to become available")
	log.Printf("waiting for SSH port on %s...", addr)
	if err := waitForPort(ctx, addr, 5*time.Minute); err != nil {
		return fmt.Errorf("wait for ssh: %w", err)
	}
	log.Printf("SSH port is open on %s", addr)

	sshClient, err := NewSSHClient(addr, server.SSHUser, server.SSHKey, 30*time.Second)
	if err != nil {
		return err
	}

	// Wait for rescue mode to exit and verify stable SSH connectivity.
	// The stability check requires the connection to be stable for stabilityDuration,
	// ensuring the OS has fully booted and won't reboot again.
	// 8 minutes is enough for Hetzner's rescue system to complete provisioning.
	// 2 minutes of stability is sufficient to confirm the OS is fully operational
	// and won't undergo additional reboots (based on observed Hetzner boot patterns).
	const rescueExitTimeout = 8 * time.Minute
	const stabilityDuration = 2 * time.Minute
	log.Printf("waiting for rescue system to exit and SSH to stabilize (stability duration: %v)...", stabilityDuration)
	if err := waitForStableSSH(ctx, sshClient, rescueExitTimeout, stabilityDuration); err != nil {
		return err
	}
	log.Printf("SSH connection is stable, system has exited rescue mode")

	builder := NewBuilder(sshClient, o.cfg)
	buildCtx, cancel := context.WithTimeout(ctx, time.Duration(o.cfg.BuildTimeoutMinutes)*time.Minute)
	defer cancel()

	progress.Step("stage source on server")
	log.Printf("uploading source archive to server...")
	if err := builder.StageSource(buildCtx, archivePath); err != nil {
		return err
	}
	log.Printf("source staged successfully")

	progress.Step("run build on server")
	log.Printf("starting build...")
	result, err := builder.Run(buildCtx)
	if err != nil {
		log.Printf("build failed, collecting remote logs")
		builderLogs := strings.TrimSpace(builder.joinLogs())
		if builderLogs != "" {
			log.Printf("remote command logs:\n%s", sanitizeLog(builderLogs))
		}
		remoteLogs, logErr := builder.SaveRemoteLogs(ctx)
		combinedLogs := builderLogs
		if logErr != nil {
			log.Printf("failed to collect remote logs: %v", logErr)
		} else {
			combinedLogs = joinLogParts(builderLogs, remoteLogs)
		}
		if combinedLogs != "" {
			_ = saveLogs(o.cfg, sanitizeLog(combinedLogs))
		}
		return fmt.Errorf("build failed: %w", err)
	}
	log.Printf("build completed successfully")

	progress.Step("download artifacts")
	artifacts, err := builder.DownloadArtifacts(ctx, result.Artifacts)
	if err != nil {
		return err
	}
	log.Printf("downloaded %d artifacts", len(artifacts))

	keepServer = false
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

// waitForStableSSH waits for the system to exit rescue mode and then verifies
// SSH connectivity is stable for the specified duration. This ensures:
// 1. The system is not in rescue mode
// 2. SSH connection is successful multiple times over the stability period
// 3. The system hostname remains consistent (not rebooting)
func waitForStableSSH(ctx context.Context, sshClient *SSHClient, timeout, stabilityDuration time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastHostname string
	var stableStart time.Time
	// 10 seconds between checks balances responsiveness with avoiding excessive
	// connection attempts. This interval is short enough to detect reboots quickly
	// but long enough to avoid overloading the server during boot.
	const checkInterval = 10 * time.Second

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for stable SSH connection")
		}

		// Try to connect and get hostname
		hostname, _, hostnameErr := sshClient.Run(ctx, "hostname")
		if hostnameErr != nil {
			log.Printf("SSH connection attempt failed: %v", hostnameErr)
			stableStart = time.Time{} // Reset stability timer
			if err := sleepWithContext(ctx, checkInterval); err != nil {
				return err
			}
			continue
		}
		hostname = strings.TrimSpace(hostname)
		log.Printf("SSH connected, hostname=%s", hostname)

		// Check if still in rescue mode
		if isRescueHostname(hostname) {
			log.Printf("system still in rescue mode (hostname=%s), waiting...", hostname)
			stableStart = time.Time{} // Reset stability timer
			if err := sleepWithContext(ctx, checkInterval); err != nil {
				return err
			}
			continue
		}

		// Check root filesystem
		rootFs, _, rootFsErr := sshClient.Run(ctx, "df -T /")
		if rootFsErr != nil {
			log.Printf("failed to check root filesystem: %v", rootFsErr)
			stableStart = time.Time{} // Reset stability timer
			if err := sleepWithContext(ctx, checkInterval); err != nil {
				return err
			}
			continue
		}
		if isRescueRootFilesystem(rootFs) {
			log.Printf("system has rescue root filesystem (tmpfs/ramfs), waiting...")
			stableStart = time.Time{} // Reset stability timer
			if err := sleepWithContext(ctx, checkInterval); err != nil {
				return err
			}
			continue
		}

		// System has exited rescue mode, start/continue stability check
		if stableStart.IsZero() || hostname != lastHostname {
			if lastHostname != "" && hostname != lastHostname {
				log.Printf("hostname changed from %s to %s, resetting stability timer", lastHostname, hostname)
			}
			stableStart = time.Now()
			lastHostname = hostname
			log.Printf("starting stability check for %v (hostname=%s)", stabilityDuration, hostname)
		}

		stableDuration := time.Since(stableStart)
		if stableDuration >= stabilityDuration {
			log.Printf("SSH connection stable for %v, proceeding", stableDuration)
			return nil
		}

		log.Printf("SSH stable for %v/%v", stableDuration.Truncate(time.Second), stabilityDuration)
		if err := sleepWithContext(ctx, checkInterval); err != nil {
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
