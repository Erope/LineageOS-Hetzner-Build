package lineage

import (
	"path/filepath"
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
		"warning: executing unverified installer script",
		"get.docker.com checksum verification failed",
		"docker compose plugin is required but not available; install Docker with get.docker.com which includes the compose plugin",
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
	composeCommand := builder.buildComposeCommand()
	expectedSnippets := []string{
		"docker compose version",
		"docker compose -f 'docker-compose.yml' pull",
		"docker compose -f 'docker-compose.yml' up --build --abort-on-container-exit --exit-code-from build",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(composeCommand, snippet) {
			t.Fatalf("expected compose command to contain %q", snippet)
		}
	}
}

func TestNormalizeComposeFilePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		buildSource string
		composeFile string
		expected    string
		expectErr   bool
	}{
		{
			name:        "empty compose file",
			buildSource: "/tmp/source",
			composeFile: "",
			expected:    "",
		},
		{
			name:        "default compose file",
			buildSource: "/tmp/source",
			composeFile: "docker-compose.yml",
			expected:    "docker-compose.yml",
		},
		{
			name:        "nested compose file",
			buildSource: "/tmp/source",
			composeFile: "compose/docker-compose.yml",
			expected:    filepath.Join("compose", "docker-compose.yml"),
		},
		{
			name:        "absolute compose file inside source dir",
			buildSource: "/tmp/source",
			composeFile: "/tmp/source/compose/docker-compose.yml",
			expected:    filepath.Join("compose", "docker-compose.yml"),
		},
		{
			name:        "compose file outside source dir",
			buildSource: "/tmp/source",
			composeFile: "/tmp/other/docker-compose.yml",
			expectErr:   true,
		},
		{
			name:        "compose file is dot",
			buildSource: "/tmp/source",
			composeFile: ".",
			expectErr:   true,
		},
		{
			name:        "compose file attempts escape",
			buildSource: "/tmp/source",
			composeFile: "../docker-compose.yml",
			expectErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeComposeFilePath(tc.buildSource, tc.composeFile)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error for %s", tc.name)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}
