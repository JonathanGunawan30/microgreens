package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
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

func setupRoleHandlerTest() (*echo.Echo, *mocks.RoleServiceInterface, handler.RoleHandlerInterface) {
	e := echo.New()
	mockService := new(mocks.RoleServiceInterface)
	cfg := &config.Config{
		App: config.App{
			JwtSecretKey: "secret",
		},
	}

	h := handler.NewRoleHandler(e, mockService, cfg, &redis.Client{})
	return e, mockService, h
}

func TestGetAllRole(t *testing.T) {
	e, mockService, h := setupRoleHandlerTest()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/roles?search=Admin", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		expectedRoles := []entity.RoleEntity{
			{ID: 1, Name: "Admin"},
		}
		mockService.On("GetAllRole", mock.Anything, "Admin").Return(expectedRoles, nil).Once()

		if assert.NoError(t, h.GetAllRole(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
			var resp response.DefaultResponse
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Equal(t, "Success", resp.Message)
		}
	})

	t.Run("Internal Server Error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/roles", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockService.On("GetAllRole", mock.Anything, "").Return(nil, errors.New("service error")).Once()

		if assert.NoError(t, h.GetAllRole(c)) {
			assert.Equal(t, http.StatusInternalServerError, rec.Code)
		}
	})
}

func TestGetRoleByID(t *testing.T) {
	e, mockService, h := setupRoleHandlerTest()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/roles/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		expectedRole := &entity.RoleEntity{ID: 1, Name: "Admin"}
		mockService.On("GetRoleByID", mock.Anything, int64(1)).Return(expectedRole, nil).Once()

		if assert.NoError(t, h.GetRoleByID(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("Bad Request - Invalid ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/roles/abc", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")

		if assert.NoError(t, h.GetRoleByID(c)) {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/roles/2", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("2")

		mockService.On("GetRoleByID", mock.Anything, int64(2)).Return(nil, message.ErrRoleNotFound).Once()

		if assert.NoError(t, h.GetRoleByID(c)) {
			assert.Equal(t, http.StatusNotFound, rec.Code)
		}
	})
}

func TestCreateRole(t *testing.T) {
	e, mockService, h := setupRoleHandlerTest()
	e.Validator = validator.NewValidator()

	t.Run("Success", func(t *testing.T) {
		roleReq := `{"name": "NewAdmin"}`
		req := httptest.NewRequest(http.MethodPost, "/admin/roles", bytes.NewBufferString(roleReq))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockService.On("CreateRole", mock.Anything, entity.RoleEntity{Name: "NewAdmin"}).Return(nil).Once()

		if assert.NoError(t, h.CreateRole(c)) {
			assert.Equal(t, http.StatusCreated, rec.Code)
		}
	})

	t.Run("Validation Error", func(t *testing.T) {
		roleReq := `{"name": ""}`
		req := httptest.NewRequest(http.MethodPost, "/admin/roles", bytes.NewBufferString(roleReq))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if assert.NoError(t, h.CreateRole(c)) {
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		}
	})
}

func TestUpdateRole(t *testing.T) {
	e, mockService, h := setupRoleHandlerTest()
	e.Validator = validator.NewValidator()

	t.Run("Success", func(t *testing.T) {
		roleReq := `{"name": "UpdatedAdmin"}`
		req := httptest.NewRequest(http.MethodPut, "/admin/roles/1", bytes.NewBufferString(roleReq))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		mockService.On("UpdateRole", mock.Anything, entity.RoleEntity{ID: 1, Name: "UpdatedAdmin"}).Return(nil).Once()

		if assert.NoError(t, h.UpdateRole(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		roleReq := `{"name": "UpdatedAdmin"}`
		req := httptest.NewRequest(http.MethodPut, "/admin/roles/2", bytes.NewBufferString(roleReq))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("2")

		mockService.On("UpdateRole", mock.Anything, entity.RoleEntity{ID: 2, Name: "UpdatedAdmin"}).Return(message.ErrRoleNotFound).Once()

		if assert.NoError(t, h.UpdateRole(c)) {
			assert.Equal(t, http.StatusNotFound, rec.Code)
		}
	})
}

func TestDeleteRoleByID(t *testing.T) {
	e, mockService, h := setupRoleHandlerTest()

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/admin/roles/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		mockService.On("DeleteRoleByID", mock.Anything, int64(1)).Return(nil).Once()

		if assert.NoError(t, h.DeleteRoleByID(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("Conflict - Role Associated", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/admin/roles/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		mockService.On("DeleteRoleByID", mock.Anything, int64(1)).Return(message.ErrRoleAssociated).Once()

		if assert.NoError(t, h.DeleteRoleByID(c)) {
			assert.Equal(t, http.StatusConflict, rec.Code)
		}
	})
}
