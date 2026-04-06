package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"product-service/config"
	handler2 "product-service/internal/adapter/handler"
	"product-service/internal/adapter/handler/request"
	"product-service/internal/adapter/handler/response"
	"product-service/internal/core/domain/entity"
	svcMocks "product-service/tests/mocks/service"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCartHandler_AddToCart(t *testing.T) {
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	mockSvc := new(svcMocks.CartServiceInterface)
	cfg := &config.Config{
		App: config.App{
			JwtSecretKey: "secret",
		},
	}
	h := handler2.NewCartHandler(e, mockSvc, cfg, nil)

	t.Run("success 201", func(t *testing.T) {
		reqBody := request.CartRequest{
			ProductID: 1,
			Quantity:  2,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/cart", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", entity.JwtUserData{ID: 100})

		mockSvc.On("AddToCart", mock.Anything, int64(100), int64(1), int64(2)).Return(nil).Once()

		if assert.NoError(t, h.AddToCart(c)) {
			assert.Equal(t, http.StatusCreated, rec.Code)
		}
	})
}

func TestCartHandler_GetCart(t *testing.T) {
	e := echo.New()
	mockSvc := new(svcMocks.CartServiceInterface)
	cfg := &config.Config{
		App: config.App{
			JwtSecretKey: "secret",
		},
	}
	h := handler2.NewCartHandler(e, mockSvc, cfg, nil)

	t.Run("success 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/cart", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", entity.JwtUserData{ID: 100})

		mockSvc.On("GetCart", mock.Anything, int64(100)).Return([]response.CartResponse{{ID: 1, Name: "Prod"}}, nil).Once()

		if assert.NoError(t, h.GetCart(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})
}
