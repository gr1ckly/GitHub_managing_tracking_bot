package dto

import (
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
	return &ChangingDTO{
		Link:      commit.GetHTMLURL(),
		Author:    commit.GetAuthor().GetLogin(),
		Title:     commit.GetCommit().GetMessage(),
		UpdatedAt: commit.GetCommit().GetCommitter().GetDate(),
	}
}
