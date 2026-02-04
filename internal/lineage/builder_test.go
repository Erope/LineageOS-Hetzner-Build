package lineage

import (
	"strings"
	"testing"
)

func TestDockerInstallCommandContainsExpectedContent(t *testing.T) {
	t.Parallel()

	command := dockerInstallCommand()
	expectedSnippets := []string{
		"install_docker_packages()",
		"docker_compose_available()",
		"ensure the build server runs as root",
		"curl is required to install Docker",
		"GET_DOCKER_SHA256",
		"warning: executing get.docker.com installer",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(command, snippet) {
			t.Fatalf("expected docker install command to contain %q", snippet)
		}
	}
}
