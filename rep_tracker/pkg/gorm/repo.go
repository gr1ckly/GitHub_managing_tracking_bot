package gorm

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/go-github/github"
	gormio "gorm.io/gorm"
)

type GormGlobalRepo struct {
	gorm *gormio.DB
}

func NewGormGlobalRepo(gorm *gormio.DB) *GormGlobalRepo {
	return &GormGlobalRepo{gorm: gorm}
}

func (r *GormGlobalRepo) SaveCommitsAndUpdateNotification(ctx context.Context, commits ...*github.RepositoryCommit) error {
	if len(commits) == 0 {
		return nil
	}

	repoOwner, repoName, ok := parseOwnerRepo(commits[0])
	if !ok {
		return fmt.Errorf("unable to parse repo from commit")
	}

	return r.gorm.WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		repo, err := findRepo(tx, repoOwner, repoName)
		if err != nil {
			return err
		}

		hashes := make([]string, 0, len(commits))
		for _, c := range commits {
			if c == nil || c.GetSHA() == "" {
				continue
			}
			hashes = append(hashes, c.GetSHA())
		}

		existing := make([]Commit, 0, len(hashes))
		if len(hashes) > 0 {
			if err := tx.Where("repo_id = ? AND commit_hash IN ?", repo.ID, hashes).Find(&existing).Error; err != nil {
				return err
			}
		}

		known := make(map[string]Commit, len(existing))
		for _, c := range existing {
			if c.CommitHash != nil {
				known[*c.CommitHash] = c
			}
		}

		for _, c := range commits {
			if c == nil || c.GetSHA() == "" {
				continue
			}
			if _, ok := known[c.GetSHA()]; ok {
				continue
			}

			commitTime := getCommitTime(c)
			var authorID *int
			if login := getAuthorLogin(c); login != "" {
				var user User
				if err := tx.Where("username = ?", login).First(&user).Error; err == nil {
					authorID = &user.ID
				}
			}

			newCommit := Commit{
				RepoID:     repo.ID,
				CommitHash: ptrString(c.GetSHA()),
				AuthorID:   authorID,
				Message:    ptrString(c.GetCommit().GetMessage()),
				Pushing:    ptrBool(false),
				CreatedAt:  commitTime,
			}
			if err := tx.Create(&newCommit).Error; err != nil {
				return err
			}
			known[c.GetSHA()] = newCommit
		}

		var latest Commit
		if err := tx.Where("repo_id = ?", repo.ID).
			Order("created_at DESC").
			First(&latest).Error; err != nil {
			return err
		}

		return tx.Model(&Notification{}).
			Where("repo_id = ? AND enabled = ?", repo.ID, true).
			Update("last_commit", latest.ID).Error
	})
}

func (r *GormGlobalRepo) GetCountTrackingRepos(ctx context.Context) (int, error) {
	var count int64
	err := r.gorm.WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		return tx.Model(&Notification{}).
			Where("enabled = ?", true).
			Count(&count).Error
	})
	return int(count), err
}

func (r *GormGlobalRepo) GetTrackingRepos(ctx context.Context, offset int, limit int) ([]*Notification, error) {
	var notifications []*Notification
	err := r.gorm.WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		return tx.Where("notifications.enabled = ?", true).
			Joins("User").
			Joins("Repo").
			Preload("LastCommitEntity").
			Order("notifications.id").
			Offset(offset).
			Limit(limit).
			Find(&notifications).Error
	})
	return notifications, err
}

func (r *GormGlobalRepo) GetToken(ctx context.Context, userId int) (string, error) {
	var token Token
	err := r.gorm.WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		return tx.Where("user_id = ?", userId).
			Order("created_at DESC").
			Limit(1).
			First(&token).Error
	})
	if err != nil {
		return "", err
	}
	return token.Token, nil
}

func findRepo(db *gormio.DB, owner string, name string) (*Repo, error) {
	var repo Repo
	err := db.Where("owner = ? AND name = ?", owner, name).First(&repo).Error
	if err == nil {
		return &repo, nil
	}
	if !errors.Is(err, gormio.ErrRecordNotFound) {
		return nil, err
	}
	urlValue := fmt.Sprintf("https://github.com/%s/%s", owner, name)
	err = db.Where("url = ? OR url = ?", urlValue, urlValue+".git").First(&repo).Error
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func parseOwnerRepo(commit *github.RepositoryCommit) (string, string, bool) {
	if commit == nil {
		return "", "", false
	}
	if owner, repo, ok := parseOwnerRepoFromURL(commit.GetHTMLURL()); ok {
		return owner, repo, true
	}
	if owner, repo, ok := parseOwnerRepoFromURL(commit.GetURL()); ok {
		return owner, repo, true
	}
	return "", "", false
}

func parseOwnerRepoFromURL(raw string) (string, string, bool) {
	if raw == "" {
		return "", "", false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", false
	}
	path := strings.Trim(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", false
	}
	if parts[0] == "repos" && len(parts) >= 3 {
		return parts[1], strings.TrimSuffix(parts[2], ".git"), true
	}
	return parts[0], strings.TrimSuffix(parts[1], ".git"), true
}

func getCommitTime(commit *github.RepositoryCommit) time.Time {
	if commit == nil || commit.Commit == nil {
		return time.Now().UTC()
	}
	if commit.Commit.Committer != nil {
		if t := commit.Commit.Committer.GetDate(); !t.IsZero() {
			return t
		}
	}
	if commit.Commit.Author != nil {
		if t := commit.Commit.Author.GetDate(); !t.IsZero() {
			return t
		}
	}
	return time.Now().UTC()
}

func ptrString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func ptrBool(v bool) *bool {
	return &v
}

func getAuthorLogin(commit *github.RepositoryCommit) string {
	if commit == nil {
		return ""
	}
	if commit.Author != nil && commit.Author.GetLogin() != "" {
		return commit.Author.GetLogin()
	}
	return ""
}
