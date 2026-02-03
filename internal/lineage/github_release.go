package lineage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-github/v59/github"
	"golang.org/x/oauth2"
)

type GitHubReleaseClient struct {
	client *github.Client
}

func NewGitHubReleaseClient(token string) *GitHubReleaseClient {
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	return &GitHubReleaseClient{client: github.NewClient(oauth2.NewClient(context.Background(), src))}
}

func (gc *GitHubReleaseClient) UploadArtifacts(ctx context.Context, cfg Config, artifacts []string) error {
	release, err := gc.getOrCreateRelease(ctx, cfg)
	if err != nil {
		return err
	}

	for _, path := range artifacts {
		if err := gc.uploadAsset(ctx, cfg, release, path); err != nil {
			return err
		}
	}
	return nil
}

func (gc *GitHubReleaseClient) getOrCreateRelease(ctx context.Context, cfg Config) (*github.RepositoryRelease, error) {
	release, _, err := gc.client.Repositories.GetReleaseByTag(ctx, cfg.ReleaseRepoOwner, cfg.ReleaseRepoName, cfg.ReleaseTag)
	if err == nil {
		return release, nil
	}

	releaseRequest := &github.RepositoryRelease{
		TagName: github.String(cfg.ReleaseTag),
		Name:    github.String(cfg.ReleaseName),
		Body:    github.String(cfg.ReleaseNotes),
	}
	release, _, err = gc.client.Repositories.CreateRelease(ctx, cfg.ReleaseRepoOwner, cfg.ReleaseRepoName, releaseRequest)
	if err != nil {
		return nil, fmt.Errorf("create release: %w", err)
	}
	return release, nil
}

func (gc *GitHubReleaseClient) uploadAsset(ctx context.Context, cfg Config, release *github.RepositoryRelease, path string) error {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("open artifact: %w", err)
	}
	defer file.Close()

	_, _, err = gc.client.Repositories.UploadReleaseAsset(ctx, cfg.ReleaseRepoOwner, cfg.ReleaseRepoName, release.GetID(), &github.UploadOptions{
		Name: filepath.Base(path),
	}, file)
	if err != nil {
		return fmt.Errorf("upload release asset: %w", err)
	}
	return nil
}
