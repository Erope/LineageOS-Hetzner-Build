package lineage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// GitHubSSHKey represents a GitHub user's SSH key
type GitHubSSHKey struct {
	ID  int    `json:"id"`
	Key string `json:"key"`
}

// FetchGitHubUserSSHKeys fetches SSH public keys for a GitHub user
func FetchGitHubUserSSHKeys(ctx context.Context, username string) ([]string, error) {
	if username == "" {
		return nil, nil
	}

	url := fmt.Sprintf("https://api.github.com/users/%s/keys", username)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Use client without its own timeout, relying on context
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch github keys: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	// Limit response body to 1MB to prevent memory exhaustion
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var keys []GitHubSSHKey
	if err := json.Unmarshal(body, &keys); err != nil {
		return nil, fmt.Errorf("parse github keys: %w", err)
	}

	publicKeys := make([]string, 0, len(keys))
	for _, k := range keys {
		if k.Key != "" {
			publicKeys = append(publicKeys, strings.TrimSpace(k.Key))
		}
	}

	return publicKeys, nil
}

// GetGitHubActorSSHKeys retrieves SSH keys for the GitHub Actions actor if running in GitHub Actions
func GetGitHubActorSSHKeys(ctx context.Context) ([]string, error) {
	// Check if running in GitHub Actions
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		return nil, nil
	}

	// Get the GitHub actor (user who triggered the workflow)
	actor := os.Getenv("GITHUB_ACTOR")
	if actor == "" {
		return nil, nil
	}

	keys, err := FetchGitHubUserSSHKeys(ctx, actor)
	if err != nil {
		// Don't fail the build if we can't fetch keys, just log a warning
		return nil, fmt.Errorf("failed to fetch GitHub user SSH keys for %s: %w", actor, err)
	}

	return keys, nil
}
