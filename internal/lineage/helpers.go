package lineage

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func shellQuote(value string) string {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(value, "'", "'\\''"))
}

func ensureKnownHosts(host string, port int, baseDir string) (string, error) {
	output, err := scanHostKey(host, port)
	if err != nil {
		return "", err
	}
	return writeKnownHosts(output, baseDir)
}

func scanHostKey(host string, port int) (string, error) {
	if !isValidHost(host) {
		return "", fmt.Errorf("invalid host: %s", host)
	}
	// Hostname is validated by isValidHost to prevent injection in ssh-keyscan.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh-keyscan", "-t", "ed25519", "-p", fmt.Sprintf("%d", port), host)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh-keyscan failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return "", fmt.Errorf("ssh-keyscan returned no data for %s", host)
	}
	return output, nil
}

func writeKnownHosts(output string, baseDir string) (string, error) {
	if baseDir == "" {
		baseDir = os.TempDir()
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", fmt.Errorf("create known hosts dir: %w", err)
	}
	path := filepath.Join(baseDir, "known_hosts")
	if err := os.WriteFile(path, []byte(output), 0o600); err != nil {
		return "", fmt.Errorf("write known hosts: %w", err)
	}
	return path, nil
}

func isValidHost(host string) bool {
	if host == "" {
		return false
	}
	if ip := net.ParseIP(host); ip != nil {
		return true
	}
	hasInvalidPrefix := strings.HasPrefix(host, "-") || strings.HasPrefix(host, ".")
	hasInvalidSuffix := strings.HasSuffix(host, "-") || strings.HasSuffix(host, ".")
	hasConsecutiveDots := strings.Contains(host, "..")
	if hasInvalidPrefix || hasInvalidSuffix || hasConsecutiveDots {
		return false
	}
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if label == "" || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, r := range label {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
				continue
			}
			return false
		}
	}
	return true
}

func randomSuffix() (string, error) {
	randomBytes := make([]byte, 6)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate random suffix: %w", err)
	}
	return hex.EncodeToString(randomBytes), nil
}

func isRescueHostname(hostname string) bool {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	return hostname == "rescue" || strings.HasPrefix(hostname, "rescue-")
}

func isRescueRootFilesystem(output string) bool {
	output = strings.ToLower(output)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[len(fields)-1] != "/" {
			continue
		}
		fsType := fields[1]
		if fsType == "tmpfs" || fsType == "ramfs" {
			return true
		}
	}
	return false
}
