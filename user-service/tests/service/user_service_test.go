package service

import (
	"context"
	"errors"
	"testing"
	"user-service/config"
	"user-service/internal/core/domain/entity"
	service2 "user-service/internal/core/service"
	mocks2 "user-service/tests/mocks"
	"user-service/utils/conv"
	"user-service/utils/message"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupUserServiceTest() (
	*mocks2.UserRepositoryInterface,
	*mocks2.JwtServiceInterface,
	*mocks2.VerificationTokenRepositoryInterface,
	service2.UserServiceInterface,
	*config.Config,
) {
	mockRepo := new(mocks2.UserRepositoryInterface)
	mockJwt := new(mocks2.JwtServiceInterface)
	mockRepoToken := new(mocks2.VerificationTokenRepositoryInterface)

	cfg := &config.Config{
		App: config.App{
			UrlFrontEnd:       "http://localhost:3000",
			UrlForgotPassword: "http://localhost:3000",
		},
		ExchangeName: config.ExchangeName{
			UserEvent: "user_event",
		},
	}

	userService := service2.NewUserService(mockRepo, cfg, mockJwt, nil, nil, mockRepoToken)
	return mockRepo, mockJwt, mockRepoToken, userService, cfg
}

func TestSignIn(t *testing.T) {
	mockRepo, mockJwt, _, userService, _ := setupUserServiceTest()
	ctx := context.Background()
	hashedPass, _ := conv.HashPassword("password123")

	t.Run("success", func(t *testing.T) {
		req := entity.UserEntity{Email: "test@example.com", Password: "password123"}
		user := &entity.UserEntity{ID: 1, Email: "test@example.com", Password: hashedPass, RoleName: "Admin", Name: "Test"}

		mockRepo.On("GetUserByEmail", ctx, req.Email).Return(user, nil).Once()
		mockJwt.On("GenerateToken", user.ID, user.RoleName).Return("fake-token", nil).Once()

		resUser, token, err := userService.SignIn(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, "fake-token", token)
		assert.Equal(t, user, resUser)
		mockRepo.AssertExpectations(t)
		mockJwt.AssertExpectations(t)
	})

	t.Run("user not found", func(t *testing.T) {
		req := entity.UserEntity{Email: "notfound@example.com", Password: "password123"}
		mockRepo.On("GetUserByEmail", ctx, req.Email).Return(nil, message.ErrUserNotFound).Once()

		resUser, token, err := userService.SignIn(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, message.ErrInvalidCredential, err)
		assert.Empty(t, token)
		assert.Nil(t, resUser)
	})

	t.Run("wrong password", func(t *testing.T) {
		req := entity.UserEntity{Email: "test@example.com", Password: "wrongpassword"}
		user := &entity.UserEntity{ID: 1, Email: "test@example.com", Password: hashedPass}
		mockRepo.On("GetUserByEmail", ctx, req.Email).Return(user, nil).Once()

		resUser, token, err := userService.SignIn(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, message.ErrInvalidCredential, err)
		assert.Empty(t, token)
		assert.Nil(t, resUser)
	})

	t.Run("jwt error", func(t *testing.T) {
		req := entity.UserEntity{Email: "test@example.com", Password: "password123"}
		user := &entity.UserEntity{ID: 1, Email: "test@example.com", Password: hashedPass, RoleName: "Admin"}

		mockRepo.On("GetUserByEmail", ctx, req.Email).Return(user, nil).Once()
		mockJwt.On("GenerateToken", user.ID, user.RoleName).Return("", errors.New("jwt err")).Once()

		_, _, err := userService.SignIn(ctx, req)
		assert.Error(t, err)
	})
}

func TestSignOut(t *testing.T) {
	_, _, _, userService, _ := setupUserServiceTest()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		err := userService.SignOut(ctx, 1)
		assert.NoError(t, err)
	})
}

func TestCreateUserAccount(t *testing.T) {
	mockRepo, _, _, userService, _ := setupUserServiceTest()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		req := entity.UserEntity{Email: "new@example.com", Password: "password123"}
		mockRepo.On("CreateUserAccount", ctx, mock.MatchedBy(func(u entity.UserEntity) bool {
			return u.Email == "new@example.com" && u.Token != ""
		})).Return(int64(1), nil).Once()

		err := userService.CreateUserAccount(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("repo error", func(t *testing.T) {
		req := entity.UserEntity{Email: "fail@example.com", Password: "password123"}
		mockRepo.On("CreateUserAccount", ctx, mock.Anything).Return(int64(0), errors.New("db error")).Once()

		err := userService.CreateUserAccount(ctx, req)
		assert.Error(t, err)
	})
}

func TestForgotPassword(t *testing.T) {
	mockRepo, _, mockRepoToken, userService, _ := setupUserServiceTest()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		req := entity.UserEntity{Email: "test@example.com"}
		user := &entity.UserEntity{ID: 1, Email: "test@example.com"}

		mockRepo.On("GetUserByEmail", ctx, req.Email).Return(user, nil).Once()
		mockRepoToken.On("CreateVerificationToken", ctx, mock.MatchedBy(func(vt entity.VerificationTokenEntity) bool {
			return vt.UserID == 1 && vt.TokenType == "forgot_password"
		})).Return(nil).Once()

		err := userService.ForgotPassword(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("user not found", func(t *testing.T) {
		req := entity.UserEntity{Email: "notfound@example.com"}
		mockRepo.On("GetUserByEmail", ctx, req.Email).Return(nil, message.ErrUserNotFound).Once()

		err := userService.ForgotPassword(ctx, req)
		assert.Error(t, err)
	})

	t.Run("token creation error", func(t *testing.T) {
		req := entity.UserEntity{Email: "test@example.com"}
		user := &entity.UserEntity{ID: 1, Email: "test@example.com"}

		mockRepo.On("GetUserByEmail", ctx, req.Email).Return(user, nil).Once()
		mockRepoToken.On("CreateVerificationToken", ctx, mock.Anything).Return(errors.New("db error")).Once()

		err := userService.ForgotPassword(ctx, req)
		assert.Error(t, err)
	})
}

func TestVerifyToken(t *testing.T) {
	mockRepo, mockJwt, mockRepoToken, userService, _ := setupUserServiceTest()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		token := "some-token"
		verifyToken := &entity.VerificationTokenEntity{UserID: 1, Token: token}
		userVerified := &entity.UserEntity{ID: 1, RoleName: "User"}

		mockRepoToken.On("GetDataByToken", ctx, token).Return(verifyToken, nil).Once()
		mockRepo.On("UpdateUserVerified", ctx, int64(1)).Return(userVerified, nil).Once()
		mockJwt.On("GenerateToken", int64(1), "User").Return("access-token", nil).Once()

		resUser, err := userService.VerifyToken(ctx, token)
		assert.NoError(t, err)
		assert.NotNil(t, resUser)
		assert.Equal(t, "access-token", resUser.Token)
	})

	t.Run("invalid token", func(t *testing.T) {
		mockRepoToken.On("GetDataByToken", ctx, "invalid").Return(nil, errors.New("not found")).Once()

		resUser, err := userService.VerifyToken(ctx, "invalid")
		assert.Error(t, err)
		assert.Nil(t, resUser)
	})

	t.Run("update failed", func(t *testing.T) {
		token := "some-token"
		verifyToken := &entity.VerificationTokenEntity{UserID: 1, Token: token}

		mockRepoToken.On("GetDataByToken", ctx, token).Return(verifyToken, nil).Once()
		mockRepo.On("UpdateUserVerified", ctx, int64(1)).Return(nil, errors.New("db err")).Once()

		resUser, err := userService.VerifyToken(ctx, token)
		assert.Error(t, err)
		assert.Nil(t, resUser)
	})
}

func TestUpdatePassword(t *testing.T) {
	mockRepo, _, mockRepoToken, userService, _ := setupUserServiceTest()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		req := entity.UserEntity{Token: "valid-token", Password: "newpassword123"}
		vt := &entity.VerificationTokenEntity{UserID: 1, TokenType: "forgot_password"}

		mockRepoToken.On("GetDataByToken", ctx, "valid-token").Return(vt, nil).Once()
		mockRepo.On("UpdatePasswordByID", ctx, mock.MatchedBy(func(u entity.UserEntity) bool {
			return u.ID == 1
		})).Return(nil).Once()

		err := userService.UpdatePassword(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("invalid token type", func(t *testing.T) {
		req := entity.UserEntity{Token: "valid-token", Password: "newpassword123"}
		vt := &entity.VerificationTokenEntity{UserID: 1, TokenType: "verify_account"}

		mockRepoToken.On("GetDataByToken", ctx, "valid-token").Return(vt, nil).Once()

		err := userService.UpdatePassword(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, message.ErrTokenNotFound, err)
	})
}

func TestGetUserProfile(t *testing.T) {
	mockRepo, _, _, userService, _ := setupUserServiceTest()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		expectedUser := &entity.UserEntity{ID: 1, Name: "Test User"}
		mockRepo.On("GetUserByID", ctx, int64(1)).Return(expectedUser, nil).Once()

		user, err := userService.GetUserProfile(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, expectedUser, user)
		mockRepo.AssertExpectations(t)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.On("GetUserByID", ctx, int64(2)).Return(nil, errors.New("user not found")).Once()

		user, err := userService.GetUserProfile(ctx, 2)

		assert.Error(t, err)
		assert.Nil(t, user)
		mockRepo.AssertExpectations(t)
	})
}

func TestUpdateProfilePassword(t *testing.T) {
	mockRepo, _, _, userService, _ := setupUserServiceTest()
	ctx := context.Background()
	hashedPass, _ := conv.HashPassword("oldpassword123")

	t.Run("success", func(t *testing.T) {
		mockRepo.On("GetUserHashedPasswordByID", ctx, int64(1)).Return(hashedPass, nil).Once()
		mockRepo.On("UpdatePasswordByID", ctx, mock.MatchedBy(func(u entity.UserEntity) bool {
			return u.ID == 1
		})).Return(nil).Once()

		err := userService.UpdateProfilePassword(ctx, 1, "oldpassword123", "newpassword123")
		assert.NoError(t, err)
	})

	t.Run("wrong old password", func(t *testing.T) {
		mockRepo.On("GetUserHashedPasswordByID", ctx, int64(1)).Return(hashedPass, nil).Once()

		err := userService.UpdateProfilePassword(ctx, 1, "wrongpassword", "newpassword123")
		assert.Error(t, err)
		assert.Equal(t, message.ErrWrongPassword, err)
	})
}

func TestUpdateDataUser(t *testing.T) {
	mockRepo, _, _, userService, _ := setupUserServiceTest()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		user := entity.UserEntity{ID: 1, Name: "Updated"}
		mockRepo.On("UpdateDataUser", ctx, user).Return(nil).Once()

		err := userService.UpdateDataUser(ctx, user)
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		user := entity.UserEntity{ID: 2}
		mockRepo.On("UpdateDataUser", ctx, user).Return(errors.New("db err")).Once()

		err := userService.UpdateDataUser(ctx, user)
		assert.Error(t, err)
	})
}

func TestGetAllCustomers(t *testing.T) {
	mockRepo, _, _, userService, _ := setupUserServiceTest()
	ctx := context.Background()
	query := entity.QueryStringCustomer{Search: "test"}

	t.Run("success", func(t *testing.T) {
		expectedUsers := []entity.UserEntity{{ID: 1, Name: "Customer 1"}}
		mockRepo.On("GetAllCustomers", ctx, query).Return(expectedUsers, int64(1), int64(1), nil).Once()

		users, totalItems, totalPages, err := userService.GetAllCustomers(ctx, query)

		assert.NoError(t, err)
		assert.Equal(t, expectedUsers, users)
		assert.Equal(t, int64(1), totalItems)
		assert.Equal(t, int64(1), totalPages)
	})
}

func TestGetCustomerByID(t *testing.T) {
	mockRepo, _, _, userService, _ := setupUserServiceTest()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		expectedUser := &entity.UserEntity{ID: 1, Name: "Customer 1"}
		mockRepo.On("GetCustomerByID", ctx, int64(1)).Return(expectedUser, nil).Once()

		user, err := userService.GetCustomerByID(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, expectedUser, user)
	})
}

func TestCreateCustomer(t *testing.T) {
	mockRepo, _, _, userService, _ := setupUserServiceTest()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		req := entity.UserEntity{Email: "cust@example.com", Password: "password123"}
		mockRepo.On("CreateCustomer", ctx, mock.MatchedBy(func(u entity.UserEntity) bool {
			return u.Email == "cust@example.com" && u.Password != "password123" // should be hashed
		})).Return(int64(1), nil).Once()

		err := userService.CreateCustomer(ctx, req)
		assert.NoError(t, err)
	})
}

func TestUpdateCustomer(t *testing.T) {
	mockRepo, _, _, userService, _ := setupUserServiceTest()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		req := entity.UserEntity{ID: 1, Name: "Updated"}
		mockRepo.On("UpdateCustomer", ctx, req).Return(nil).Once()

		err := userService.UpdateCustomer(ctx, req)
		assert.NoError(t, err)
	})
}

func TestDeleteCustomer(t *testing.T) {
	mockRepo, _, _, userService, _ := setupUserServiceTest()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockRepo.On("DeleteCustomer", ctx, int64(1)).Return(nil).Once()

		err := userService.DeleteCustomer(ctx, 1)
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.On("DeleteCustomer", ctx, int64(2)).Return(errors.New("delete failed")).Once()

		err := userService.DeleteCustomer(ctx, 2)
		assert.Error(t, err)
	})
}
