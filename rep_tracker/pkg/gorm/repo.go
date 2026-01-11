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
	"gorm.io/gorm/clause"
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
		repo, err := findRepo(ctx, tx, repoOwner, repoName)
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
			existing, err = gormio.G[Commit](tx).
				Where("repo_id = ? AND commit_hash IN ?", repo.ID, hashes).
				Find(ctx)
			if err != nil {
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
				user, err := gormio.G[User](tx).
					Where("username = ?", login).
					First(ctx)
				if err == nil {
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
			if err := gormio.G[Commit](tx).Create(ctx, &newCommit); err != nil {
				return err
			}
			known[c.GetSHA()] = newCommit
		}

		latest, err := gormio.G[Commit](tx).
			Where("repo_id = ?", repo.ID).
			Order("created_at DESC").
			First(ctx)
		if err != nil {
			return err
		}

		_, err = gormio.G[Notification](tx).
			Where("repo_id = ? AND enabled = ?", repo.ID, true).
			Update(ctx, "last_commit", latest.ID)
		return err
	})
}

func (r *GormGlobalRepo) GetCountTrackingRepos(ctx context.Context) (int, error) {
	var count int64
	err := r.gorm.WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		var err error
		count, err = gormio.G[Notification](tx).
			Where("enabled = ?", true).
			Count(ctx, "id")
		return err
	})
	return int(count), err
}

func (r *GormGlobalRepo) GetTrackingRepos(ctx context.Context, offset int, limit int) ([]*Notification, error) {
	var notifications []*Notification
	err := r.gorm.WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		items, err := gormio.G[Notification](tx).
			Where("notifications.enabled = ?", true).
			Joins(clause.JoinTarget{Type: clause.LeftJoin, Table: "users"}, func(db gormio.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
				db.Where("users.id = notifications.user_id")
				return nil
			}).
			Joins(clause.JoinTarget{Type: clause.LeftJoin, Table: "repos"}, func(db gormio.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
				db.Where("repos.id = notifications.repo_id")
				return nil
			}).
			Preload("User", func(db gormio.PreloadBuilder) error { return nil }).
			Preload("Repo", func(db gormio.PreloadBuilder) error { return nil }).
			Preload("LastCommitEntity", func(db gormio.PreloadBuilder) error { return nil }).
			Order("notifications.id").
			Offset(offset).
			Limit(limit).
			Find(ctx)
		if err != nil {
			return err
		}
		notifications = make([]*Notification, 0, len(items))
		for i := range items {
			notifications = append(notifications, &items[i])
		}
		return nil
	})
	return notifications, err
}

func (r *GormGlobalRepo) GetToken(ctx context.Context, userId int) (string, error) {
	var token Token
	err := r.gorm.WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		var err error
		token, err = gormio.G[Token](tx).
			Where("user_id = ?", userId).
			Order("created_at DESC").
			Limit(1).
			First(ctx)
		return err
	})
	if err != nil {
		return "", err
	}
	return token.Token, nil
}

func findRepo(ctx context.Context, db *gormio.DB, owner string, name string) (*Repo, error) {
	repo, err := gormio.G[Repo](db).
		Where("owner = ? AND name = ?", owner, name).
		First(ctx)
	if err == nil {
		return &repo, nil
	}
	if !errors.Is(err, gormio.ErrRecordNotFound) {
		return nil, err
	}
	urlValue := fmt.Sprintf("https://github.com/%s/%s", owner, name)
	repo, err = gormio.G[Repo](db).
		Where("url = ? OR url = ?", urlValue, urlValue+".git").
		First(ctx)
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
