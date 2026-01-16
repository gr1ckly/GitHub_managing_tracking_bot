package gorm

import "time"

type FileState string

const (
	FileStateAdded    FileState = "ADDED"
	FileStateModified FileState = "MODIFIED"
	FileStateDeleted  FileState = "DELETED"
)

type User struct {
	ID        int       `gorm:"column:id;primaryKey;autoIncrement"`
	ChatID    string    `gorm:"column:chat_id;unique;not null"`
	Username  *string   `gorm:"column:username"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`

	Tokens        []Token         `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	UserRepos     []UserRepo      `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Commits       []Commit        `gorm:"foreignKey:AuthorID;references:ID;constraint:OnDelete:SET NULL"`
	EditorSession []EditorSession `gorm:"foreignKey:ForUser;references:ID;constraint:OnDelete:SET NULL"`
	Notifications []Notification  `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

type Token struct {
	ID             int        `gorm:"column:id;primaryKey;autoIncrement"`
	UserID         int        `gorm:"column:user_id;unique;not null"`
	Token          string     `gorm:"column:token;size:256;unique;not null"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime"`
	LastValidateAt *time.Time `gorm:"column:last_validate_at"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

type Repo struct {
	ID      int       `gorm:"column:id;primaryKey;autoIncrement"`
	URL     string    `gorm:"column:url;unique;not null"`
	Owner   *string   `gorm:"column:owner"`
	Name    *string   `gorm:"column:name"`
	AddedAt time.Time `gorm:"column:added_at;autoCreateTime"`

	UserRepos     []UserRepo     `gorm:"foreignKey:RepoID;references:ID;constraint:OnDelete:CASCADE"`
	Branches      []Branch       `gorm:"foreignKey:RepoID;references:ID;constraint:OnDelete:CASCADE"`
	Files         []File         `gorm:"foreignKey:RepoID;references:ID;constraint:OnDelete:CASCADE"`
	Commits       []Commit       `gorm:"foreignKey:RepoID;references:ID;constraint:OnDelete:CASCADE"`
	Notifications []Notification `gorm:"foreignKey:RepoID;references:ID;constraint:OnDelete:CASCADE"`
}

type UserRepo struct {
	UserID  int       `gorm:"column:user_id;primaryKey"`
	RepoID  int       `gorm:"column:repo_id;primaryKey"`
	AddedAt time.Time `gorm:"column:added_at;autoCreateTime"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Repo Repo `gorm:"foreignKey:RepoID;references:ID;constraint:OnDelete:CASCADE"`
}

type Branch struct {
	ID           int    `gorm:"column:id;primaryKey;autoIncrement"`
	RepoID       int    `gorm:"column:repo_id;not null"`
	Name         string `gorm:"column:name;not null"`
	LastCommitID *int64 `gorm:"column:last_commit_id"`

	Repo       Repo     `gorm:"foreignKey:RepoID;references:ID;constraint:OnDelete:CASCADE"`
	LastCommit *Commit  `gorm:"foreignKey:LastCommitID;references:ID;constraint:OnDelete:SET NULL"`
	Commits    []Commit `gorm:"foreignKey:BranchID;references:ID;constraint:OnDelete:SET NULL"`
}

type File struct {
	ID         int       `gorm:"column:id;primaryKey;autoIncrement"`
	RepoID     int       `gorm:"column:repo_id;not null"`
	State      FileState `gorm:"column:state;type:file_state;not null"`
	Path       string    `gorm:"column:path;not null"`
	StorageKey *string   `gorm:"column:storage_key"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`

	Repo           Repo            `gorm:"foreignKey:RepoID;references:ID;constraint:OnDelete:CASCADE"`
	CommitFiles    []CommitFile    `gorm:"foreignKey:FileID;references:ID;constraint:OnDelete:CASCADE"`
	EditorSessions []EditorSession `gorm:"foreignKey:FileID;references:ID;constraint:OnDelete:CASCADE"`
}

type Commit struct {
	ID         int64     `gorm:"column:id;primaryKey;autoIncrement"`
	RepoID     int       `gorm:"column:repo_id;not null"`
	BranchID   *int      `gorm:"column:branch_id"`
	CommitHash *string   `gorm:"column:commit_hash"`
	AuthorID   *int      `gorm:"column:author_id"`
	Message    *string   `gorm:"column:message"`
	Pushing    *bool     `gorm:"column:pushing"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`

	Repo          Repo           `gorm:"foreignKey:RepoID;references:ID;constraint:OnDelete:CASCADE"`
	Branch        *Branch        `gorm:"foreignKey:BranchID;references:ID;constraint:OnDelete:SET NULL"`
	Author        *User          `gorm:"foreignKey:AuthorID;references:ID;constraint:OnDelete:SET NULL"`
	CommitFiles   []CommitFile   `gorm:"foreignKey:CommitID;references:ID;constraint:OnDelete:CASCADE"`
	Notifications []Notification `gorm:"foreignKey:LastCommit;references:ID;constraint:OnDelete:SET NULL"`
}

type CommitFile struct {
	CommitID int64 `gorm:"column:commit_id;primaryKey"`
	FileID   int   `gorm:"column:file_id;primaryKey"`

	Commit Commit `gorm:"foreignKey:CommitID;references:ID;constraint:OnDelete:CASCADE"`
	File   File   `gorm:"foreignKey:FileID;references:ID;constraint:OnDelete:CASCADE"`
}

type EditorSession struct {
	ID         int64      `gorm:"column:id;primaryKey;autoIncrement"`
	FileID     int        `gorm:"column:file_id;not null"`
	SessionURL string     `gorm:"column:session_url;not null"`
	ForUser    *int       `gorm:"column:for_user"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime"`
	ExpiresAt  *time.Time `gorm:"column:expires_at"`

	File File  `gorm:"foreignKey:FileID;references:ID;constraint:OnDelete:CASCADE"`
	User *User `gorm:"foreignKey:ForUser;references:ID;constraint:OnDelete:SET NULL"`
}

type Notification struct {
	ID         int       `gorm:"column:id;primaryKey;autoIncrement"`
	UserID     int       `gorm:"column:user_id;not null"`
	RepoID     int       `gorm:"column:repo_id;not null"`
	LastCommit *int64    `gorm:"column:last_commit"`
	Enabled    bool      `gorm:"column:enabled;default:true"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`

	User             User    `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Repo             Repo    `gorm:"foreignKey:RepoID;references:ID;constraint:OnDelete:CASCADE"`
	LastCommitEntity *Commit `gorm:"foreignKey:LastCommit;references:ID;constraint:OnDelete:SET NULL"`
}
