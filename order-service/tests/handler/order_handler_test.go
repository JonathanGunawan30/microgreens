package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"order-service/config"
	"order-service/internal/adapter/handler"
	"order-service/internal/adapter/handler/request"
	"order-service/internal/core/domain/entity"
	"order-service/mocks"
	"order-service/utils/message"
	"order-service/utils/validator"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupEcho() (*echo.Echo, *config.Config) {
	e := echo.New()
	e.Validator = validator.NewValidator()
	cfg := &config.Config{}
	cfg.App.LatitudeRef = "-6.200000"
	cfg.App.LongitudeRef = "106.816666"
	return e, cfg
}

func TestCreateOrder(t *testing.T) {
	e, cfg := setupEcho()
	mockSvc := new(mocks.OrderServiceInterface)
	h := handler.NewOrderHandler(mockSvc, e, cfg, nil)

	t.Run("success 201", func(t *testing.T) {
		reqBody := request.CreateOrderRequest{
			BuyerID:      1,
			OrderDate:    "2026-04-06",
			ShippingType: "Delivery",
			PaymentType:  "Credit Card",
			OrderTime:    "10:00",
			OrderDetails: []request.OrderDetailRequest{
				{ProductID: 1, Quantity: 2},
			},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/orders", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockSvc.On("CreateOrder", mock.Anything, mock.AnythingOfType("entity.OrderEntity")).Return(int64(1), nil).Once()

		if assert.NoError(t, h.CreateOrder(c)) {
			assert.Equal(t, http.StatusCreated, rec.Code)
			assert.Contains(t, rec.Body.String(), "order_id")
		}
	})

	t.Run("bind error 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/orders", bytes.NewBufferString("invalid json"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.CreateOrder(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("validation error 422", func(t *testing.T) {
		reqBody := request.CreateOrderRequest{
			BuyerID: 0, // Invalid
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/orders", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := h.CreateOrder(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("service error 500", func(t *testing.T) {
		reqBody := request.CreateOrderRequest{
			BuyerID:      1,
			OrderDate:    "2026-04-06",
			ShippingType: "Delivery",
			PaymentType:  "Credit Card",
			OrderTime:    "10:00",
			OrderDetails: []request.OrderDetailRequest{{ProductID: 1, Quantity: 2}},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/orders", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockSvc.On("CreateOrder", mock.Anything, mock.Anything).Return(int64(0), errors.New("db error")).Once()

		err := h.CreateOrder(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})
}

func TestGetOrderByID(t *testing.T) {
	e, cfg := setupEcho()
	mockSvc := new(mocks.OrderServiceInterface)
	h := handler.NewOrderHandler(mockSvc, e, cfg, nil)

	t.Run("success 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/orders/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		mockSvc.On("GetOrderByID", mock.Anything, int64(1)).Return(&entity.OrderEntity{ID: 1}, nil).Once()

		err := h.GetOrderByID(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("not found 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/orders/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		mockSvc.On("GetOrderByID", mock.Anything, int64(1)).Return(nil, message.ErrOrderNotFound).Once()

		err := h.GetOrderByID(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("invalid id 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/orders/abc", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")

		err := h.GetOrderByID(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

func TestUpdateStatus(t *testing.T) {
	e, cfg := setupEcho()
	mockSvc := new(mocks.OrderServiceInterface)
	h := handler.NewOrderHandler(mockSvc, e, cfg, nil)

	t.Run("success 200", func(t *testing.T) {
		reqBody := request.OrderUpdateStatusRequest{Status: "Completed"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPatch, "/admin/orders/1/status", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		mockSvc.On("UpdateStatusOrder", mock.Anything, mock.Anything).Return(nil).Once()

		err := h.UpdateStatus(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid transition 409", func(t *testing.T) {
		reqBody := request.OrderUpdateStatusRequest{Status: "Completed"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPatch, "/admin/orders/1/status", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		// Source code fixed: returns 409 for invalid status transition
		mockSvc.On("UpdateStatusOrder", mock.Anything, mock.Anything).Return(errors.New("invalid status transition")).Once()

		err := h.UpdateStatus(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusConflict, rec.Code)
	})
}

func TestGetAllCustomerOrders(t *testing.T) {
	e, cfg := setupEcho()
	mockSvc := new(mocks.OrderServiceInterface)
	h := handler.NewOrderHandler(mockSvc, e, cfg, nil)

	t.Run("success 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/orders", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockSvc.On("GetAllCustomerOrders", mock.Anything, mock.Anything, "token").Return([]entity.OrderEntity{{ID: 1}}, int64(1), int64(1), nil).Once()

		err := h.GetAllCustomerOrders(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("unauthorized 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/orders", nil)
		req.Header.Set(echo.HeaderAuthorization, "Bearer invalid-token")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Source code fixed: returns 401 for invalid token
		mockSvc.On("GetAllCustomerOrders", mock.Anything, mock.Anything, "invalid-token").Return(nil, int64(0), int64(0), errors.New("invalid token")).Once()

		err := h.GetAllCustomerOrders(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
