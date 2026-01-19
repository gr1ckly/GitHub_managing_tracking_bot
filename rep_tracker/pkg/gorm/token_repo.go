package gorm

import (
	"context"

	gormio "gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormTokenRepo struct {
	gorm *gormio.DB
}

func NewGormTokenRepo(gorm *gormio.DB) *GormTokenRepo {
	return &GormTokenRepo{gorm: gorm}
}

func (r *GormTokenRepo) GetToken(ctx context.Context, chatId string) (string, error) {
	var token Token
	err := r.gorm.WithContext(ctx).Transaction(func(tx *gormio.DB) error {
		var err error
		token, err = gormio.G[Token](tx).
			Joins(clause.JoinTarget{Type: clause.InnerJoin, Table: "users"}, func(db gormio.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
				db.Where("users.id = tokens.user_id")
				return nil
			}).
			Where("users.chat_id = ?", chatId).
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
