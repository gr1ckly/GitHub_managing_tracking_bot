package gorm

import (
	"context"

	gormio "gorm.io/gorm"
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
		err = tx.Table("tokens").
			Select("tokens.*").
			Joins("INNER JOIN users ON users.id = tokens.user_id").
			Where("users.chat_id = ?", chatId).
			Order("tokens.created_at DESC").
			Limit(1).
			First(&token).Error
		return err
	})
	if err != nil {
		return "", err
	}
	return token.Token, nil
}
