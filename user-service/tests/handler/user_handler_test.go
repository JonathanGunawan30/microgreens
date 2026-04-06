package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"user-service/config"
	"user-service/internal/adapter/handler"
	"user-service/internal/adapter/handler/response"
	"user-service/internal/core/domain/entity"
	"user-service/tests/mocks"
	"user-service/utils/message"
	"user-service/utils/validator"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupUserHandlerTest() (*echo.Echo, *mocks.UserServiceInterface, handler.UserHandlerInterface) {
	e := echo.New()
	mockService := new(mocks.UserServiceInterface)
	cfg := &config.Config{
		App: config.App{
			JwtSecretKey: "secret",
		},
	}

	h := handler.NewUserHandler(e, mockService, cfg, &redis.Client{})
	return e, mockService, h
}

func TestSignIn(t *testing.T) {
	e, mockService, h := setupUserHandlerTest()
	e.Validator = validator.NewValidator()

	t.Run("Success", func(t *testing.T) {
		signInReq := `{"email": "test@example.com", "password": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/signin", bytes.NewBufferString(signInReq))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		expectedUser := &entity.UserEntity{ID: 1, Email: "test@example.com", Name: "Test"}
		mockService.On("SignIn", mock.Anything, mock.MatchedBy(func(req entity.UserEntity) bool {
			return req.Email == "test@example.com"
		})).Return(expectedUser, "fake-token", nil).Once()

		if assert.NoError(t, h.SignIn(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
			var resp response.DefaultResponse
			json.Unmarshal(rec.Body.Bytes(), &resp)
			assert.Equal(t, "Success", resp.Message)
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		signInReq := `{"email": "wrong@example.com", "password": "wrongpassword"}`
		req := httptest.NewRequest(http.MethodPost, "/signin", bytes.NewBufferString(signInReq))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockService.On("SignIn", mock.Anything, mock.Anything).Return(nil, "", message.ErrInvalidCredential).Once()

		if assert.NoError(t, h.SignIn(c)) {
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		}
	})
}

func TestCreateUserAccount(t *testing.T) {
	e, mockService, h := setupUserHandlerTest()
	e.Validator = validator.NewValidator()

	t.Run("Success", func(t *testing.T) {
		signUpReq := `{"name": "Test", "email": "test@example.com", "password": "password123", "password_confirmation": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(signUpReq))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockService.On("CreateUserAccount", mock.Anything, mock.MatchedBy(func(req entity.UserEntity) bool {
			return req.Email == "test@example.com"
		})).Return(nil).Once()

		if assert.NoError(t, h.CreateUserAccount(c)) {
			assert.Equal(t, http.StatusCreated, rec.Code)
		}
	})

	t.Run("Conflict - Email Exists", func(t *testing.T) {
		signUpReq := `{"name": "Test", "email": "exists@example.com", "password": "password123", "password_confirmation": "password123"}`
		req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(signUpReq))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockService.On("CreateUserAccount", mock.Anything, mock.Anything).Return(message.ErrEmailAlreadyExists).Once()

		if assert.NoError(t, h.CreateUserAccount(c)) {
			assert.Equal(t, http.StatusConflict, rec.Code)
		}
	})
}

func TestGetUserProfile(t *testing.T) {
	e, mockService, h := setupUserHandlerTest()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/profile", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", entity.JwtUserData{ID: 1})

		expectedUser := &entity.UserEntity{ID: 1, Name: "Test", Email: "test@example.com"}
		mockService.On("GetUserProfile", mock.Anything, int64(1)).Return(expectedUser, nil).Once()

		if assert.NoError(t, h.GetUserProfile(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("Unauthorized - No Context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/profile", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if assert.NoError(t, h.GetUserProfile(c)) {
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		}
	})
}

func TestForgotPassword(t *testing.T) {
	e, mockService, h := setupUserHandlerTest()
	e.Validator = validator.NewValidator()

	t.Run("Success", func(t *testing.T) {
		reqBody := `{"email": "test@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewBufferString(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockService.On("ForgotPassword", mock.Anything, mock.MatchedBy(func(req entity.UserEntity) bool {
			return req.Email == "test@example.com"
		})).Return(nil).Once()

		if assert.NoError(t, h.ForgotPassword(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		reqBody := `{"email": "notfound@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewBufferString(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockService.On("ForgotPassword", mock.Anything, mock.Anything).Return(message.ErrUserNotFound).Once()

		if assert.NoError(t, h.ForgotPassword(c)) {
			assert.Equal(t, http.StatusNotFound, rec.Code)
		}
	})
}

func TestVerifyAccount(t *testing.T) {
	e, mockService, h := setupUserHandlerTest()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/verify-account?token=valid-token", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		expectedUser := &entity.UserEntity{ID: 1, Token: "new-access-token"}
		mockService.On("VerifyToken", mock.Anything, "valid-token").Return(expectedUser, nil).Once()

		if assert.NoError(t, h.VerifyAccount(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("Unauthorized - Expired", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/verify-account?token=expired", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockService.On("VerifyToken", mock.Anything, "expired").Return(nil, message.ErrTokenExpired).Once()

		if assert.NoError(t, h.VerifyAccount(c)) {
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		}
	})
}

func TestUpdatePassword(t *testing.T) {
	e, mockService, h := setupUserHandlerTest()
	e.Validator = validator.NewValidator()

	t.Run("Success", func(t *testing.T) {
		reqBody := `{"token": "valid-token", "password_new": "newpass123", "password_confirmation": "newpass123"}`
		req := httptest.NewRequest(http.MethodPut, "/update-password", bytes.NewBufferString(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockService.On("UpdatePassword", mock.Anything, mock.MatchedBy(func(req entity.UserEntity) bool {
			return req.Token == "valid-token" && req.Password == "newpass123"
		})).Return(nil).Once()

		if assert.NoError(t, h.UpdatePassword(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})
}

func TestUpdateProfilePassword(t *testing.T) {
	e, mockService, h := setupUserHandlerTest()
	e.Validator = validator.NewValidator()

	t.Run("Success", func(t *testing.T) {
		reqBody := `{"current_password": "oldpass123", "new_password": "newpass123", "confirm_password": "newpass123"}`
		req := httptest.NewRequest(http.MethodPatch, "/auth/profile/password", bytes.NewBufferString(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", entity.JwtUserData{ID: 1})

		mockService.On("UpdateProfilePassword", mock.Anything, int64(1), "oldpass123", "newpass123").Return(nil).Once()

		if assert.NoError(t, h.UpdateProfilePassword(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})
}

func TestUpdateDataUser(t *testing.T) {
	e, mockService, h := setupUserHandlerTest()
	e.Validator = validator.NewValidator()

	t.Run("Success", func(t *testing.T) {
		reqBody := `{"name": "Updated", "email": "up@example.com", "phone": "12345678901", "address": "Address", "lat": "1.0", "lng": "2.0", "photo": "photo.jpg"}`
		req := httptest.NewRequest(http.MethodPut, "/auth/profile", bytes.NewBufferString(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", entity.JwtUserData{ID: 1})

		mockService.On("UpdateDataUser", mock.Anything, mock.MatchedBy(func(req entity.UserEntity) bool {
			return req.ID == 1 && req.Name == "Updated"
		})).Return(nil).Once()

		if assert.NoError(t, h.UpdateDataUser(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})
}

func TestGetAllCustomers(t *testing.T) {
	e, mockService, h := setupUserHandlerTest()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/customers", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", entity.JwtUserData{ID: 1})

		customers := []entity.UserEntity{{ID: 2, Name: "Customer 1"}}
		mockService.On("GetAllCustomers", mock.Anything, mock.Anything).Return(customers, int64(1), int64(1), nil).Once()

		if assert.NoError(t, h.GetAllCustomers(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})
}

func TestGetCustomerByID(t *testing.T) {
	e, mockService, h := setupUserHandlerTest()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/customers/2", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", entity.JwtUserData{ID: 1})
		c.SetParamNames("id")
		c.SetParamValues("2")

		customer := &entity.UserEntity{ID: 2, Name: "Customer 1"}
		mockService.On("GetCustomerByID", mock.Anything, int64(2)).Return(customer, nil).Once()

		if assert.NoError(t, h.GetCustomerByID(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})
}

func TestCreateCustomer(t *testing.T) {
	e, mockService, h := setupUserHandlerTest()
	e.Validator = validator.NewValidator()

	t.Run("Success", func(t *testing.T) {
		reqBody := `{"name": "Customer", "email": "cust@example.com", "password": "password123", "phone": "12345678901", "address": "Address", "lat": "1.0", "lng": "2.0", "photo": "photo.jpg"}`
		req := httptest.NewRequest(http.MethodPost, "/admin/customers", bytes.NewBufferString(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", entity.JwtUserData{ID: 1})

		mockService.On("CreateCustomer", mock.Anything, mock.MatchedBy(func(req entity.UserEntity) bool {
			return req.Email == "cust@example.com"
		})).Return(nil).Once()

		if assert.NoError(t, h.CreateCustomer(c)) {
			assert.Equal(t, http.StatusCreated, rec.Code)
		}
	})
}

func TestUpdateCustomer(t *testing.T) {
	e, mockService, h := setupUserHandlerTest()
	e.Validator = validator.NewValidator()

	t.Run("Success", func(t *testing.T) {
		reqBody := `{"name": "Customer Up", "email": "cust@example.com", "phone": "12345678901", "address": "Address", "lat": "1.0", "lng": "2.0", "photo": "photo.jpg"}`
		req := httptest.NewRequest(http.MethodPut, "/admin/customers/2", bytes.NewBufferString(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", entity.JwtUserData{ID: 1})
		c.SetParamNames("id")
		c.SetParamValues("2")

		mockService.On("UpdateCustomer", mock.Anything, mock.MatchedBy(func(req entity.UserEntity) bool {
			return req.ID == 2 && req.Name == "Customer Up"
		})).Return(nil).Once()

		if assert.NoError(t, h.UpdateCustomer(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})
}

func TestDeleteCustomer(t *testing.T) {
	e, mockService, h := setupUserHandlerTest()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/admin/customers/2", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", entity.JwtUserData{ID: 1})
		c.SetParamNames("id")
		c.SetParamValues("2")

		mockService.On("DeleteCustomer", mock.Anything, int64(2)).Return(nil).Once()

		if assert.NoError(t, h.DeleteCustomer(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})
}
