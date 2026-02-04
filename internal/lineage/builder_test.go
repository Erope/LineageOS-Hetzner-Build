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
		"curl is required to install Docker; set HETZNER_SERVER_IMAGE to an image that includes curl",
		"GET_DOCKER_SHA256",
		"warning: executing get.docker.com installer",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(command, snippet) {
			t.Fatalf("expected docker install command to contain %q", snippet)
		}
	}
}

func TestRunComposeUsesDockerComposePlugin(t *testing.T) {
	t.Parallel()

	ssh := &SSHClient{}
	builder := NewBuilder(ssh, Config{
		WorkingDir:  "/tmp/build",
		ComposeFile: "docker-compose.yml",
	})
	command := builder.buildComposeCommand()
	expectedSnippets := []string{
		"docker compose version",
		"docker compose -f 'docker-compose.yml' pull",
		"docker compose -f 'docker-compose.yml' up --build --abort-on-container-exit --exit-code-from build",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(command, snippet) {
			t.Fatalf("expected compose command to contain %q", snippet)
		}
	}
}
