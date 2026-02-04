package lineage

import (
	"strings"
	"testing"
)

func TestDockerInstallCommandContainsMessages(t *testing.T) {
	t.Parallel()

	command := dockerInstallCommand()
	expectedSnippets := []string{
		"install_docker_packages()",
		"docker_compose_available()",
		"docker_compose()",
		"ensure the build server runs as root",
		"set HETZNER_SERVER_IMAGE to a Debian/Ubuntu image",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(command, snippet) {
			t.Fatalf("expected docker install command to contain %q", snippet)
		}
	}
}
