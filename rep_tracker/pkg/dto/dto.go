package dto

import (
	"strings"
	"time"

	"github.com/google/go-github/github"
)

type ChangingDTO struct {
	Link      string    `json:"link"`
	Author    string    `json:"author"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ConvertRepositoryCommitToDTO(commit *github.RepositoryCommit) *ChangingDTO {
	link := commit.GetHTMLURL()
	if link == "" {
		link = commit.GetURL()
	}
	if link == "" && commit.GetCommit() != nil {
		link = commit.GetCommit().GetURL()
	}
	return &ChangingDTO{
		Link:      normalizeGitHubLink(link),
		Author:    commit.GetAuthor().GetLogin(),
		Title:     commit.GetCommit().GetMessage(),
		UpdatedAt: commit.GetCommit().GetCommitter().GetDate(),
	}
}

func normalizeGitHubLink(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	normalized := strings.Replace(trimmed, "api.github.com/repos/", "github.com/", 1)
	normalized = strings.Replace(normalized, "/commits/", "/commit/", 1)
	return normalized
}
