package lineage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func fetchGitHubPublicKeys(ctx context.Context, actor string) ([]string, error) {
	const githubAPITimeout = 10 * time.Second

	if actor == "" {
		return nil, fmt.Errorf("github actor is required")
	}
	escapedActor := url.PathEscape(actor)
	requestURL := fmt.Sprintf("https://github.com/%s.keys", escapedActor)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create github keys request: %w", err)
	}
	req.Header.Set("Accept", "text/plain")

	client := &http.Client{Timeout: githubAPITimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch github keys: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch github keys: unexpected status %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read github keys response: %w", err)
	}
	lines := strings.Split(string(body), "\n")
	keys := make([]string, 0, len(lines))
	for _, line := range lines {
		key := strings.TrimSpace(line)
		if key == "" {
			continue
		}
		keys = append(keys, key)
	}
	return keys, nil
}
