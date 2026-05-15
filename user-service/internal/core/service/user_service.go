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
	SignOut(ctx context.Context, userID int64) error
	CreateUserAccount(ctx context.Context, user entity.UserEntity) error
	ForgotPassword(ctx context.Context, user entity.UserEntity) error
	VerifyToken(ctx context.Context, token string) (*entity.UserEntity, error)
	UpdatePassword(ctx context.Context, user entity.UserEntity) error
	GetUserProfile(ctx context.Context, userID int64) (*entity.UserEntity, error)
	UpdateProfilePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error
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
	rabbitmq   *config.RabbitMQClient
	repoToken  repository.VerificationTokenRepositoryInterface
}

func NewUserService(repo repository.UserRepositoryInterface, cfg *config.Config, jwtService JwtServiceInterface, redis *redis.Client, rabbitmq *config.RabbitMQClient, repositoryInterface repository.VerificationTokenRepositoryInterface) UserServiceInterface {
	return &userService{
		repo:       repo,
		cfg:        cfg,
		jwtService: jwtService,
		redis:      redis,
		rabbitmq:   rabbitmq,
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

	token, err := u.jwtService.GenerateToken(user.ID, user.RoleName)
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

	if u.redis != nil {
		redisClient := u.redis

		sessionKey := "session:" + strconv.FormatInt(user.ID, 10)

		err = redisClient.HSet(ctx, sessionKey, sessionData).Err()
		if err != nil {
			log.Errorf("[UserService-4] SignIn: %v", err)
			return nil, "", err
		}

		redisClient.Expire(ctx, sessionKey, time.Hour*24)
	}

	return user, token, nil

}

func (u *userService) SignOut(ctx context.Context, userID int64) error {
	if u.redis == nil {
		return nil
	}
	sessionKey := "session:" + strconv.FormatInt(userID, 10)

	err := u.redis.Del(ctx, sessionKey).Err()
	if err != nil {
		log.Errorf("[UserService] SignOut: Failed to delete redis key: %v", err)
		return err
	}
	return nil
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

	userID, err := u.repo.CreateUserAccount(ctx, user)
	if err != nil {
		log.Errorf("[UserService-2] CreateUserAccount: %v", err)
		return err
	}

	urlVerify := fmt.Sprintf("%s/auth/verify-account?token=%s", u.cfg.App.UrlFrontEnd, token)
	messageParam := fmt.Sprintf("Please verify your email address by clicking this link: %s", urlVerify)
	go func() {
		err = publisher.PublishMessage(u.rabbitmq, userID, user.Email, messageParam, utils.NOTIF_EMAIL_VERIFICATION, "Verify Your Account")
		if err != nil {
			log.Errorf("[UserService-3] CreateUserAccount: %v", err)
		}
	}()
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

	urlForgot := fmt.Sprintf("%s/auth/update-password?token=%s", u.cfg.App.UrlForgotPassword, token)
	messageParam := fmt.Sprintf("Please reset your password by clicking this link: %s", urlForgot)
	go func() {
		err = publisher.PublishMessage(u.rabbitmq, user.ID, user.Email, messageParam, utils.NOTIF_EMAIL_FORGOT_PASSWORD, "Reset Your Password")
		if err != nil {
			log.Errorf("[UserService-3] ForgotPassword: %v", err)
		}
	}()
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

	accessToken, err := u.jwtService.GenerateToken(updateUserVerified.ID, updateUserVerified.RoleName)
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
		"token":      accessToken,
		"role_name":  updateUserVerified.RoleName,
	}

	if u.redis != nil {
		redisClient := u.redis

		sessionKey := "session:" + strconv.FormatInt(updateUserVerified.ID, 10)

		err = redisClient.HSet(ctx, sessionKey, sessionData).Err()
		if err != nil {
			log.Errorf("[UserService-4] VerifyToken: %v", err)
			return nil, err
		}

		redisClient.Expire(ctx, sessionKey, time.Hour*24)
	}

	updateUserVerified.Token = accessToken

	go func(ue entity.UserEntity) {
		err := publisher.PublishUserEvent(u.rabbitmq, ue, u.cfg.ExchangeName.UserEvent)
		if err != nil {
			log.Errorf("[UserService-4] VerifyToken publish: %v", err)
		}
	}(*updateUserVerified)

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

func (u *userService) UpdateProfilePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error {
	oldHashedPassword, err := u.repo.GetUserHashedPasswordByID(ctx, userID)
	if err != nil {
		log.Errorf("[UserService - 1] UpdateProfilePassword, Failed to get hashed password: %v", err)
		return err
	}

	if !conv.CheckPasswordHash(currentPassword, oldHashedPassword) {
		return message.ErrWrongPassword
	}

	hashedPassword, err := conv.HashPassword(newPassword)
	if err != nil {
		log.Errorf("[UserService - 2] UpdateProfilePassword, Failed to hash password: %v", err)
		return err
	}

	user := entity.UserEntity{
		ID:       userID,
		Password: hashedPassword,
	}

	err = u.repo.UpdatePasswordByID(ctx, user)
	if err != nil {
		log.Errorf("[UserService - 3] UpdateProfilePassword: %v", err)
		return err
	}

	return nil
}

func (u *userService) UpdateDataUser(ctx context.Context, user entity.UserEntity) error {
	err := u.repo.UpdateDataUser(ctx, user)
	if err != nil {
		log.Errorf("[UserService] UpdateDataUser: %v", err)
		return err
	}

	go func(ue entity.UserEntity) {
		err := publisher.PublishUserEvent(u.rabbitmq, ue, u.cfg.ExchangeName.UserEvent)
		if err != nil {
			log.Errorf("[UserService] UpdateDataUser publish: %v", err)
		}
	}(user)

	return nil
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

	userID, err := u.repo.CreateCustomer(ctx, user)
	if err != nil {
		log.Errorf("[UserService-2] CreateCustomer: %v", err)
		return err
	}

	messageParam := fmt.Sprintf("Welcome to %s, your account has been created successfully. You can now login using email %s.", utils.APP_NAME, user.Email)
	go func() {
		err = publisher.PublishMessage(u.rabbitmq, userID, user.Email, messageParam, utils.NOTIF_EMAIL_CREATE_CUSTOMER, "Account has been created successfully")
		if err != nil {
			log.Errorf("[UserService-3] Email Failed (Create User): %v", err)
		}
	}()

	go func(ue entity.UserEntity) {
		ue.ID = userID
		err := publisher.PublishUserEvent(u.rabbitmq, ue, u.cfg.ExchangeName.UserEvent)
		if err != nil {
			log.Errorf("[UserService-4] CreateCustomer publish: %v", err)
		}
	}(user)

	return nil
}

func (u *userService) UpdateCustomer(ctx context.Context, user entity.UserEntity) error {
	err := u.repo.UpdateCustomer(ctx, user)
	if err != nil {
		log.Errorf("[UserService - 2] UpdateCustomer: %v", err)
		return err
	}

	go func(ue entity.UserEntity) {
		err := publisher.PublishUserEvent(u.rabbitmq, ue, u.cfg.ExchangeName.UserEvent)
		if err != nil {
			log.Errorf("[UserService - 4] UpdateCustomer publish: %v", err)
		}
	}(user)

	messageParam := fmt.Sprint("Your account has been updated successfully.")
	go func() {
		err = publisher.PublishMessage(u.rabbitmq, user.ID, user.Email, messageParam, utils.NOTIF_EMAIL_UPDATE_CUSTOMER, "Account updated successfully")
		if err != nil {
			log.Errorf("[UserService - 3] UpdateCustomer: %v", err)
		}
	}()
	return nil

}

func (u *userService) DeleteCustomer(ctx context.Context, customerID int64) error {
	return u.repo.DeleteCustomer(ctx, customerID)
}
