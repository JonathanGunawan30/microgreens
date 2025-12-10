package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
	"user-service/config"
	publisher "user-service/internal/adapter/message"
	"user-service/internal/adapter/repository"
	"user-service/internal/core/domain/entity"
	"user-service/utils"
	"user-service/utils/conv"
	"user-service/utils/message"

	"github.com/google/uuid"
	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
)

type UserServiceInterface interface {
	SignIn(ctx context.Context, req entity.UserEntity) (*entity.UserEntity, string, error)
	CreateUserAccount(ctx context.Context, user entity.UserEntity) error
	ForgotPassword(ctx context.Context, user entity.UserEntity) error
	VerifyToken(ctx context.Context, token string) (*entity.UserEntity, error)
	UpdatePassword(ctx context.Context, user entity.UserEntity) error
	GetUserProfile(ctx context.Context, userID int64) (*entity.UserEntity, error)
	UpdateDataUser(ctx context.Context, user entity.UserEntity) error

	GetAllCustomers(ctx context.Context, query entity.QueryStringCustomer) ([]entity.UserEntity, int64, int64, error)
	GetCustomerByID(ctx context.Context, userID int64) (*entity.UserEntity, error)
	CreateCustomer(ctx context.Context, user entity.UserEntity) error
	UpdateCustomer(ctx context.Context, user entity.UserEntity) error
	DeleteCustomer(ctx context.Context, customerID int64) error
}

type userService struct {
	repo       repository.UserRepositoryInterface
	cfg        *config.Config
	jwtService JwtServiceInterface
	redis      *redis.Client
	repoToken  repository.VerificationTokenRepositoryInterface
}

func NewUserService(repo repository.UserRepositoryInterface, cfg *config.Config, jwtService JwtServiceInterface, redis *redis.Client, repositoryInterface repository.VerificationTokenRepositoryInterface) UserServiceInterface {
	return &userService{
		repo:       repo,
		cfg:        cfg,
		jwtService: jwtService,
		redis:      redis,
		repoToken:  repositoryInterface,
	}
}

func (u *userService) SignIn(ctx context.Context, req entity.UserEntity) (*entity.UserEntity, string, error) {
	user, err := u.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, message.ErrUserNotFound) {
			return nil, "", message.ErrInvalidCredential
		}
		log.Errorf("[UserService-1] SignIn: %v", err)
		return nil, "", err
	}

	if checkPass := conv.CheckPasswordHash(req.Password, user.Password); !checkPass {
		log.Errorf("[UserService-2] SignIn: %v", err)
		return nil, "", message.ErrInvalidCredential
	}

	token, err := u.jwtService.GenerateToken(user.ID)
	if err != nil {
		log.Errorf("[UserService-3] SignIn: %v", err)
		return nil, "", err
	}

	sessionData := map[string]any{
		"user_id":    user.ID,
		"name":       user.Name,
		"email":      user.Email,
		"logged_in":  true,
		"created_at": time.Now().String(),
		"token":      token,
		"role_name":  user.RoleName,
	}

	redisClient := u.redis

	sessionKey := "session:" + strconv.FormatInt(user.ID, 10)

	err = redisClient.HSet(ctx, sessionKey, sessionData).Err()
	if err != nil {
		log.Errorf("[UserService-4] SignIn: %v", err)
		return nil, "", err
	}

	redisClient.Expire(ctx, sessionKey, time.Hour*24)

	return user, token, nil

}

func (u *userService) CreateUserAccount(ctx context.Context, user entity.UserEntity) error {
	hashPassword, err := conv.HashPassword(user.Password)
	if err != nil {
		log.Errorf("[UserService-1] CreateUserAccount: %v", err)
		return err
	}
	user.Password = hashPassword
	token := uuid.New().String()
	user.Token = token

	err = u.repo.CreateUserAccount(ctx, user)
	if err != nil {
		log.Errorf("[UserService-2] CreateUserAccount: %v", err)
		return err
	}

	urlVerify := fmt.Sprintf("http://localhost:8080/verify?token=%s", token)
	messageParam := fmt.Sprintf("Please verify your email address by clicking this link: %s", urlVerify)
	err = publisher.PublishMessage(user.Email, messageParam, "email_verification")
	if err != nil {
		log.Errorf("[UserService-3] CreateUserAccount: %v", err)
		return err
	}

	return nil
}

func (u *userService) ForgotPassword(ctx context.Context, user entity.UserEntity) error {
	email, err := u.repo.GetUserByEmail(ctx, user.Email)
	if err != nil {
		log.Errorf("[UserService-1] ForgotPassword: %v", err)
		return err
	}

	token := uuid.New().String()
	reqEntity := entity.VerificationTokenEntity{
		UserID:    email.ID,
		Token:     token,
		TokenType: "forgot_password",
		ExpiresAt: time.Now().Add(time.Hour * 24),
	}

	err = u.repoToken.CreateVerificationToken(ctx, reqEntity)
	if err != nil {
		log.Errorf("[UserService-2] ForgotPassword: %v", err)
		return err
	}

	urlForgot := fmt.Sprintf("%s/forgot-password?token=%s", u.cfg.App.UrlForgotPassword, token)
	messageParam := fmt.Sprintf("Please reset your password by clicking this link: %s", urlForgot)
	err = publisher.PublishMessage(user.Email, messageParam, "forgot_password")
	if err != nil {
		log.Errorf("[UserService-3] ForgotPassword: %v", err)
		return err
	}

	return nil
}

func (u *userService) VerifyToken(ctx context.Context, token string) (*entity.UserEntity, error) {
	verifyToken, err := u.repoToken.GetDataByToken(ctx, token)
	if err != nil {
		log.Errorf("[UserService-1] VerifyToken: %v", err)
		return nil, err
	}

	updateUserVerified, err := u.repo.UpdateUserVerified(ctx, verifyToken.UserID)
	if err != nil {
		log.Errorf("[UserService-2] VerifyToken: %v", err)
		return nil, err
	}

	accessToken, err := u.jwtService.GenerateToken(updateUserVerified.ID)
	if err != nil {
		log.Errorf("[UserService-3] VerifyToken: %v", err)
		return nil, err
	}

	sessionData := map[string]any{
		"user_id":    updateUserVerified.ID,
		"name":       updateUserVerified.Name,
		"email":      updateUserVerified.Email,
		"logged_in":  true,
		"created_at": time.Now().String(),
		"token":      token,
		"role_name":  updateUserVerified.RoleName,
	}

	redisClient := u.redis

	sessionKey := "session:" + strconv.FormatInt(updateUserVerified.ID, 10)

	err = redisClient.HSet(ctx, sessionKey, sessionData).Err()
	if err != nil {
		log.Errorf("[UserService-4] VerifyToken: %v", err)
		return nil, err
	}

	redisClient.Expire(ctx, sessionKey, time.Hour*24)

	updateUserVerified.Token = accessToken
	return updateUserVerified, nil
}

func (u *userService) UpdatePassword(ctx context.Context, user entity.UserEntity) error {
	token, err := u.repoToken.GetDataByToken(ctx, user.Token)
	if err != nil {
		log.Errorf("[UserService-1] UpdatePassword: %v", err)
		return err
	}

	if token.TokenType != "forgot_password" {
		log.Errorf("[UserService-2] UpdatePassword: %v", err)
		return message.ErrTokenNotFound
	}

	hashPassword, err := conv.HashPassword(user.Password)
	if err != nil {
		log.Errorf("[UserService-3] UpdatePassword: %v", err)
		return err
	}

	user.Password = hashPassword
	user.ID = token.UserID
	err = u.repo.UpdatePasswordByID(ctx, user)
	if err != nil {
		log.Errorf("[UserService-4] UpdatePassword: %v", err)
		return err
	}

	return nil
}

func (u *userService) GetUserProfile(ctx context.Context, userID int64) (*entity.UserEntity, error) {
	return u.repo.GetUserByID(ctx, userID)
}

func (u *userService) UpdateDataUser(ctx context.Context, user entity.UserEntity) error {
	return u.repo.UpdateDataUser(ctx, user)
}

func (u *userService) GetAllCustomers(ctx context.Context, query entity.QueryStringCustomer) ([]entity.UserEntity, int64, int64, error) {
	return u.repo.GetAllCustomers(ctx, query)
}

func (u *userService) GetCustomerByID(ctx context.Context, userID int64) (*entity.UserEntity, error) {
	return u.repo.GetCustomerByID(ctx, userID)
}

func (u *userService) CreateCustomer(ctx context.Context, user entity.UserEntity) error {
	password, err := conv.HashPassword(user.Password)
	if err != nil {
		log.Errorf("[UserService-1] CreateCustomer: %v", err)
		return err
	}

	user.Password = password

	err = u.repo.CreateCustomer(ctx, user)
	if err != nil {
		log.Errorf("[UserService-3] CreateCustomer: %v", err)
		return err
	}

	messageParam := fmt.Sprintf("Welcome to %s, your account has been created successfully. You can now login using email %s.", utils.APP_NAME, user.Email)

	err = publisher.PublishMessage(user.Email, messageParam, utils.NOTIF_EMAIL_CREATE_CUSTOMER)
	if err != nil {
		log.Warnf("[UserService-2] Email Failed (User Created): %v", err)
	}

	return nil
}

func (u *userService) UpdateCustomer(ctx context.Context, user entity.UserEntity) error {
	if user.Password != "" {
		password, err := conv.HashPassword(user.Password)
		if err != nil {
			log.Errorf("[UserService - 1] UpdateCustomer: %v", err)
			return err
		}
		user.Password = password
	}

	err := u.repo.UpdateCustomer(ctx, user)
	if err != nil {
		log.Errorf("[UserService - 2] UpdateCustomer: %v", err)
		return err
	}

	messageParam := fmt.Sprint("Your account has been updated successfully.")
	err = publisher.PublishMessage(user.Email, messageParam, utils.NOTIF_EMAIL_UPDATE_CUSTOMER)
	if err != nil {
		log.Warnf("[UserService - 3] UpdateCustomer: %v", err)
	}

	return nil

}

func (u *userService) DeleteCustomer(ctx context.Context, customerID int64) error {
	return u.repo.DeleteCustomer(ctx, customerID)
}
