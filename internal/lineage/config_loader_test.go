package lineage

import "testing"

func TestLoadConfigFromEnvKeepServerOnFailure(t *testing.T) {
	t.Setenv("HETZNER_TOKEN", "token")
	t.Setenv("BUILD_SOURCE_DIR", "/tmp/src")
	t.Setenv("KEEP_SERVER_ON_FAILURE", "true")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.KeepServerOnFailure {
		t.Fatalf("expected KeepServerOnFailure to be true")
	}
}

func TestLoadConfigFromEnvGitHubActions(t *testing.T) {
	t.Setenv("HETZNER_TOKEN", "token")
	t.Setenv("BUILD_SOURCE_DIR", "/tmp/src")
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_ACTOR", "octocat")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.GitHubActions {
		t.Fatalf("expected GitHubActions to be true")
	}
	if cfg.GitHubActor != "octocat" {
		t.Fatalf("expected GitHubActor to be octocat, got %q", cfg.GitHubActor)
	}
}

func TestLoadConfigFromEnvDefaults(t *testing.T) {
	t.Setenv("HETZNER_TOKEN", "token")
	t.Setenv("BUILD_SOURCE_DIR", "/tmp/src")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("GITHUB_ACTOR", "")
	t.Setenv("KEEP_SERVER_ON_FAILURE", "")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.GitHubActions {
		t.Fatalf("expected GitHubActions to default to false")
	}
	if cfg.GitHubActor != "" {
		t.Fatalf("expected GitHubActor to default to empty")
	}
	if cfg.KeepServerOnFailure {
		t.Fatalf("expected KeepServerOnFailure to default to false")
	}
}
