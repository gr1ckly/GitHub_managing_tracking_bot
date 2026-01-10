package dto

import (
	"time"

	"github.com/google/go-github/github"
)

type ChangingDTO struct {
	ChatId    string    `json:"chat_id"`
	Link      string    `json:"link"`
	Author    string    `json:"author"`
	Title     string    `json:"title"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ConvertRepositoryCommitToDTO(commit *github.RepositoryCommit, chatId string) *ChangingDTO {
	return &ChangingDTO{
		ChatId:    chatId,
		Link:      commit.GetHTMLURL(),
		Author:    commit.GetAuthor().GetLogin(),
		Title:     commit.GetCommit().GetMessage(),
		UpdatedAt: commit.GetCommit().GetCommitter().GetDate(),
	}
}
