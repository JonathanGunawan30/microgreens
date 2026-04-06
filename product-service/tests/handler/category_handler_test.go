package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"product-service/config"
	handler2 "product-service/internal/adapter/handler"
	"product-service/internal/adapter/handler/request"
	"product-service/internal/core/domain/entity"
	svcMocks "product-service/tests/mocks/service"
	"product-service/utils/message"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

func TestCategoryHandler_CreateAdminCategory(t *testing.T) {
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	mockSvc := new(svcMocks.CategoryServiceInterface)
	cfg := &config.Config{
		App: config.App{
			JwtSecretKey: "secret",
		},
	}
	h := handler2.NewCategoryHandler(e, mockSvc, cfg, nil)

	t.Run("success 201", func(t *testing.T) {
		reqBody := request.CreateCategoryRequest{
			Name:   "New Category",
			Icon:   "icon.png",
			Status: true,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/admin/categories", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockSvc.On("CreateCategory", mock.Anything, mock.MatchedBy(func(cat entity.CategoryEntity) bool {
			return cat.Name == "New Category"
		})).Return(nil).Once()

		if assert.NoError(t, h.CreateAdminCategory(c)) {
			assert.Equal(t, http.StatusCreated, rec.Code)
		}
	})

	t.Run("bad request 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/admin/categories", bytes.NewBufferString("invalid json"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if assert.NoError(t, h.CreateAdminCategory(c)) {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("validation error 422", func(t *testing.T) {
		reqBody := request.CreateCategoryRequest{
			Name: "", // Name is required
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/admin/categories", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if assert.NoError(t, h.CreateAdminCategory(c)) {
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		}
	})

	t.Run("conflict 409", func(t *testing.T) {
		reqBody := request.CreateCategoryRequest{
			Name:   "Existing Category",
			Icon:   "icon.png",
			Status: true,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/admin/categories", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockSvc.On("CreateCategory", mock.Anything, mock.Anything).Return(message.ErrCategoryAlreadyExists).Once()

		if assert.NoError(t, h.CreateAdminCategory(c)) {
			assert.Equal(t, http.StatusConflict, rec.Code)
		}
	})

	t.Run("internal server error 500", func(t *testing.T) {
		reqBody := request.CreateCategoryRequest{
			Name:   "Error Category",
			Icon:   "icon.png",
			Status: true,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/admin/categories", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockSvc.On("CreateCategory", mock.Anything, mock.Anything).Return(errors.New("error")).Once()

		if assert.NoError(t, h.CreateAdminCategory(c)) {
			assert.Equal(t, http.StatusInternalServerError, rec.Code)
		}
	})
}

func TestCategoryHandler_GetByIDAdminCategory(t *testing.T) {
	e := echo.New()
	mockSvc := new(svcMocks.CategoryServiceInterface)
	cfg := &config.Config{
		App: config.App{
			JwtSecretKey: "secret",
		},
	}
	h := handler2.NewCategoryHandler(e, mockSvc, cfg, nil)

	t.Run("success 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/categories/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		mockSvc.On("GetCategoryByID", mock.Anything, int64(1)).Return(&entity.CategoryEntity{ID: 1, Name: "Test"}, nil).Once()

		if assert.NoError(t, h.GetByIDAdminCategory(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("not found 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/categories/2", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("2")

		mockSvc.On("GetCategoryByID", mock.Anything, int64(2)).Return(nil, message.ErrCategoryNotFound).Once()

		if assert.NoError(t, h.GetByIDAdminCategory(c)) {
			assert.Equal(t, http.StatusNotFound, rec.Code)
		}
	})

	t.Run("bad request 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/categories/abc", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")

		if assert.NoError(t, h.GetByIDAdminCategory(c)) {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		}
	})
}

func TestCategoryHandler_UpdateAdminCategory(t *testing.T) {
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	mockSvc := new(svcMocks.CategoryServiceInterface)
	cfg := &config.Config{
		App: config.App{
			JwtSecretKey: "secret",
		},
	}
	h := handler2.NewCategoryHandler(e, mockSvc, cfg, nil)

	t.Run("success 200", func(t *testing.T) {
		reqBody := request.UpdateCategoryRequest{
			Name:   "Updated Category",
			Icon:   "icon.png",
			Status: true,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/admin/categories/1", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		mockSvc.On("UpdateCategory", mock.Anything, mock.Anything).Return(nil).Once()

		if assert.NoError(t, h.UpdateAdminCategory(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("validation error 422", func(t *testing.T) {
		reqBody := request.UpdateCategoryRequest{
			Name: "",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/admin/categories/1", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		if assert.NoError(t, h.UpdateAdminCategory(c)) {
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		}
	})

	t.Run("not found 404", func(t *testing.T) {
		reqBody := request.UpdateCategoryRequest{
			Name:   "Not Found Category",
			Icon:   "icon.png",
			Status: true,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/admin/categories/2", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("2")

		mockSvc.On("UpdateCategory", mock.Anything, mock.Anything).Return(message.ErrCategoryNotFound).Once()

		if assert.NoError(t, h.UpdateAdminCategory(c)) {
			assert.Equal(t, http.StatusNotFound, rec.Code)
		}
	})
}

func TestCategoryHandler_DeleteAdminCategory(t *testing.T) {
	e := echo.New()
	mockSvc := new(svcMocks.CategoryServiceInterface)
	cfg := &config.Config{
		App: config.App{
			JwtSecretKey: "secret",
		},
	}
	h := handler2.NewCategoryHandler(e, mockSvc, cfg, nil)

	t.Run("success 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/admin/categories/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		mockSvc.On("DeleteCategoryByID", mock.Anything, int64(1)).Return(nil).Once()

		if assert.NoError(t, h.DeleteAdminCategory(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("conflict 409", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/admin/categories/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		mockSvc.On("DeleteCategoryByID", mock.Anything, int64(1)).Return(message.ErrCategoryHasProducts).Once()

		if assert.NoError(t, h.DeleteAdminCategory(c)) {
			assert.Equal(t, http.StatusConflict, rec.Code)
		}
	})
}
