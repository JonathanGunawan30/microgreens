package repository

import (
	"context"
	"errors"
	"payment-service/internal/core/domain/entity"
	"payment-service/internal/core/domain/model"
	"payment-service/utils/message"

	"gorm.io/gorm"
)

type UserSnapshotRepositoryInterface interface {
	Upsert(ctx context.Context, user entity.UserSnapshotEntity) error
	GetByUserID(ctx context.Context, userID int64) (*entity.UserSnapshotEntity, error)
}

type userSnapshotRepository struct {
	db *gorm.DB
}

func NewUserSnapshotRepository(db *gorm.DB) UserSnapshotRepositoryInterface {
	return &userSnapshotRepository{db: db}
}

func (u *userSnapshotRepository) Upsert(ctx context.Context, user entity.UserSnapshotEntity) error {
	return u.db.WithContext(ctx).
		Where("user_id = ?", user.UserID).
		Assign(model.UserSnapshot{
			UserID:  user.UserID,
			Name:    user.Name,
			Email:   user.Email,
			Address: user.Address,
		}).FirstOrCreate(&model.UserSnapshot{}).Error
}

func (u *userSnapshotRepository) GetByUserID(ctx context.Context, userID int64) (*entity.UserSnapshotEntity, error) {
	var m model.UserSnapshot
	if err := u.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, message.ErrUserNotFound
		}
		return nil, err
	}

	return &entity.UserSnapshotEntity{
		UserID:  m.UserID,
		Name:    m.Name,
		Email:   m.Email,
		Address: m.Address,
	}, nil
}
