package lineage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWaitForStableKnownHosts(t *testing.T) {
	tmpDir := t.TempDir()
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+originalPath)
	t.Setenv("TMPDIR", tmpDir)

	script := `#!/bin/sh
FILE="$TMPDIR/scan_count"
count=0
if [ -f "$FILE" ]; then
  count=$(cat "$FILE")
fi
count=$((count + 1))
echo "$count" > "$FILE"
if [ "$count" -le 2 ]; then
  echo "192.0.2.1 ssh-ed25519 KEYONE"
else
  echo "192.0.2.1 ssh-ed25519 KEYTWO"
fi
`
	if err := os.WriteFile(filepath.Join(tmpDir, "ssh-keyscan"), []byte(script), 0o700); err != nil {
		t.Fatalf("write ssh-keyscan stub: %v", err)
	}

	originalInterval := hostKeyStabilityInterval
	hostKeyStabilityInterval = 1 * time.Millisecond
	defer func() {
		hostKeyStabilityInterval = originalInterval
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	path, err := waitForStableKnownHosts(ctx, "192.0.2.1", 22, tmpDir, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("waitForStableKnownHosts: %v", err)
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("read known hosts: %v", err)
	}
	if string(data) != "192.0.2.1 ssh-ed25519 KEYTWO" {
		t.Fatalf("unexpected known hosts content: %q", string(data))
	}
}

func TestWaitForStableKnownHostsTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+originalPath)
	t.Setenv("TMPDIR", tmpDir)

	script := `#!/bin/sh
FILE="$TMPDIR/scan_count_timeout"
count=0
if [ -f "$FILE" ]; then
  count=$(cat "$FILE")
fi
count=$((count + 1))
echo "$count" > "$FILE"
if [ $((count % 2)) -eq 0 ]; then
  echo "192.0.2.1 ssh-ed25519 KEYA"
else
  echo "192.0.2.1 ssh-ed25519 KEYB"
fi
`
	if err := os.WriteFile(filepath.Join(tmpDir, "ssh-keyscan"), []byte(script), 0o700); err != nil {
		t.Fatalf("write ssh-keyscan stub: %v", err)
	}

	originalInterval := hostKeyStabilityInterval
	hostKeyStabilityInterval = 1 * time.Millisecond
	defer func() {
		hostKeyStabilityInterval = originalInterval
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := waitForStableKnownHosts(ctx, "192.0.2.1", 22, tmpDir, 20*time.Millisecond)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
}

func TestScanHostKeyInvalidHost(t *testing.T) {
	t.Parallel()

	if _, err := scanHostKey("invalid host", 22); err == nil {
		t.Fatalf("expected invalid host error")
	}
}

func TestWriteKnownHosts(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path, err := writeKnownHosts("192.0.2.1 ssh-ed25519 KEY", tmpDir)
	if err != nil {
		t.Fatalf("writeKnownHosts: %v", err)
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatalf("read known hosts: %v", err)
	}
	if string(data) != "192.0.2.1 ssh-ed25519 KEY" {
		t.Fatalf("unexpected known hosts content: %q", string(data))
	}
}
