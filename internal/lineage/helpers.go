package lineage

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

func shellQuote(value string) string {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(value, "'", "'\\''"))
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
	const (
		// df -T output fields: Filesystem, Type, 1K-blocks, Used, Available, Use%, Mounted on.
		dfTypeFieldIndex       = 1
		dfMountPointFieldIndex = 6
		dfExpectedFieldCount   = 7
	)
	output = strings.ToLower(output)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < dfExpectedFieldCount {
			continue
		}
		if fields[dfMountPointFieldIndex] != "/" {
			continue
		}
		fsType := fields[dfTypeFieldIndex]
		if fsType == "tmpfs" || fsType == "ramfs" {
			return true
		}
	}
	return false
}
