package github

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type GithubClient struct {
	clientsMx sync.RWMutex
	clients   map[string]*github.Client
}

func NewGithubClient() *GithubClient {
	return &GithubClient{clients: make(map[string]*github.Client)}
}

func (c *GithubClient) CheckRepo(ctx context.Context, token string, link string) (bool, error) {
	currClient := c.getOrCreateClient(ctx, token)
	owner, repoName, err := c.getOwnerRepo(link)
	if err != nil {
		return false, err
	}
	_, resp, err := currClient.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (c *GithubClient) GetCommitsSince(ctx context.Context, token string, link string, lastTime time.Time) ([]*github.RepositoryCommit, error) {
	currClient := c.getOrCreateClient(ctx, token)
	owner, repoName, err := c.getOwnerRepo(link)
	if err != nil {
		return nil, err
	}
	commits, _, err := currClient.Repositories.ListCommits(ctx, owner, repoName, &github.CommitsListOptions{Since: lastTime})
	return commits, err
}

func (c *GithubClient) getOrCreateClient(ctx context.Context, token string) *github.Client {
	c.clientsMx.RLock()
	if client, ok := c.clients[token]; ok {
		c.clientsMx.RUnlock()
		return client
	}
	c.clientsMx.RUnlock()
	newClient := c.newClientWithToken(ctx, token)
	c.clientsMx.Lock()
	c.clients[token] = newClient
	c.clientsMx.Unlock()
	return newClient
}

func (c *GithubClient) newClientWithToken(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

func (c *GithubClient) getOwnerRepo(link string) (string, string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", "", err
	}
	path := strings.Trim(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("Invalid repository path: %v", link)
	}

	owner := parts[0]
	repo := strings.TrimSuffix(parts[1], ".git")
	if owner == "" {
		return "", "", fmt.Errorf("Empty owner: %v", link)
	}
	if repo == "" {
		return "", "", fmt.Errorf("Empty gorm: %v", link)
	}
	return owner, repo, nil
}
