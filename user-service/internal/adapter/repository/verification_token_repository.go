package repository

import (
	"context"
	"errors"
	"time"
	"user-service/internal/core/domain/entity"
	"user-service/internal/core/domain/model"
	"user-service/utils/message"

	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
)

type VerificationTokenRepositoryInterface interface {
	CreateVerificationToken(ctx context.Context, verificationToken entity.VerificationTokenEntity) error
	GetDataByToken(ctx context.Context, token string) (*entity.VerificationTokenEntity, error)
}

type verificationTokenRepository struct {
	db *gorm.DB
}

func NewVerificationTokenRepository(db *gorm.DB) VerificationTokenRepositoryInterface {
	return &verificationTokenRepository{db: db}
}

func (v *verificationTokenRepository) CreateVerificationToken(ctx context.Context, verificationToken entity.VerificationTokenEntity) error {
	modelVerificationToken := model.VerificationToken{
		UserID:    verificationToken.UserID,
		Token:     verificationToken.Token,
		TokenType: verificationToken.TokenType,
		ExpiresAt: verificationToken.ExpiresAt,
	}

	if err := v.db.WithContext(ctx).Create(&modelVerificationToken).Error; err != nil {
		log.Errorf("[VerificationTokenRepository-1] CreateVerificationToken: %v", err)
		return err
	}

	return nil

}

func (v *verificationTokenRepository) GetDataByToken(ctx context.Context, token string) (*entity.VerificationTokenEntity, error) {
	modelToken := model.VerificationToken{}

	if err := v.db.WithContext(ctx).Where("token = ?", token).First(&modelToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("[VerificationTokenRepository-2] GetDataByToken: Token not found")
			return nil, message.ErrTokenNotFound
		}
		log.Errorf("[VerificationTokenRepository-2] GetDataByToken: %v", err)
		return nil, err
	}

	currentTime := time.Now()

	if modelToken.ExpiresAt.Before(currentTime) {
		log.Error("[VerificationTokenRepository-3] GetDataByToken: token expired")
		return nil, message.ErrTokenExpired
	}

	return &entity.VerificationTokenEntity{
		ID:        modelToken.ID,
		UserID:    modelToken.UserID,
		Token:     modelToken.Token,
		TokenType: modelToken.TokenType,
		ExpiresAt: modelToken.ExpiresAt,
	}, nil

}
