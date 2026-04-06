package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"payment-service/config"
	"payment-service/internal/adapter/handler"
	"payment-service/internal/core/domain/entity"
	"payment-service/tests/mocks"
	"payment-service/utils/message"
	"payment-service/utils/validator"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPaymentHandler_CreatePayment(t *testing.T) {
	e := echo.New()
	e.Validator = validator.NewValidator()
	cfg := &config.Config{}

	t.Run("success 201", func(t *testing.T) {
		mockSvc := new(mocks.PaymentServiceInterface)
		h := handler.NewPaymentHandler(mockSvc, e, cfg, nil)
		reqBody := map[string]any{
			"order_id":       1,
			"payment_method": "cod",
			"user_id":        1,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/payments", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		gatewayID := "TX-123"
		mockSvc.On("ProcessPayment", mock.Anything, mock.AnythingOfType("entity.PaymentEntity")).Return(&entity.PaymentEntity{PaymentGatewayID: &gatewayID}, nil)

		if assert.NoError(t, h.CreatePayment(c)) {
			assert.Equal(t, http.StatusCreated, rec.Code)
		}
	})

	t.Run("bad request 400", func(t *testing.T) {
		mockSvc := new(mocks.PaymentServiceInterface)
		h := handler.NewPaymentHandler(mockSvc, e, cfg, nil)
		req := httptest.NewRequest(http.MethodPost, "/auth/payments", bytes.NewReader([]byte("invalid json")))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if assert.NoError(t, h.CreatePayment(c)) {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("invalid payment method 422", func(t *testing.T) {
		mockSvc := new(mocks.PaymentServiceInterface)
		h := handler.NewPaymentHandler(mockSvc, e, cfg, nil)
		reqBody := map[string]any{
			"order_id":       1,
			"payment_method": "invalid",
			"user_id":        1,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/payments", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockSvc.On("ProcessPayment", mock.Anything, mock.Anything).Return(nil, message.ErrInvalidPaymentMethod)

		if assert.NoError(t, h.CreatePayment(c)) {
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		}
	})

	t.Run("validation error 422 (missing field)", func(t *testing.T) {
		mockSvc := new(mocks.PaymentServiceInterface)
		h := handler.NewPaymentHandler(mockSvc, e, cfg, nil)
		reqBody := map[string]any{
			"payment_method": "cod",
			"user_id":        1,
			// order_id missing
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/payments", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if assert.NoError(t, h.CreatePayment(c)) {
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		}
	})
}

func TestPaymentHandler_MidtransWebHook(t *testing.T) {
	e := echo.New()
	mockSvc := new(mocks.PaymentServiceInterface)
	h := handler.NewPaymentHandler(mockSvc, e, &config.Config{}, nil)

	t.Run("success settlement", func(t *testing.T) {
		payload := map[string]any{
			"order_id":           "ORDER-123",
			"status_code":        "200",
			"gross_amount":       "1000.00",
			"signature_key":      "valid_sig",
			"transaction_status": "settlement",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/payments/webhook", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockSvc.On("VerifyMidtransSignature", "ORDER-123", "200", "1000.00", "valid_sig").Return(true)
		mockSvc.On("UpdateStatusByOrderCode", mock.Anything, "ORDER-123", "Success").Return(nil)

		if assert.NoError(t, h.MidtransWebHook(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("invalid signature 403", func(t *testing.T) {
		payload := map[string]any{
			"order_id":      "ORDER-123",
			"signature_key": "invalid",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/payments/webhook", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockSvc.On("VerifyMidtransSignature", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false)

		if assert.NoError(t, h.MidtransWebHook(c)) {
			assert.Equal(t, http.StatusForbidden, rec.Code)
		}
	})
}

func TestPaymentHandler_GetAllPayments(t *testing.T) {
	e := echo.New()
	mockSvc := new(mocks.PaymentServiceInterface)
	h := handler.NewPaymentHandler(mockSvc, e, &config.Config{}, nil)

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/payments?page=1&limit=10", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", entity.JwtUserData{ID: 1, RoleName: "Customer"})

		mockSvc.On("GetAllPayments", mock.Anything, mock.Anything).Return([]entity.PaymentEntity{{ID: 1}}, int64(1), int64(1), nil)

		if assert.NoError(t, h.GetAllPayments(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("unauthorized 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/payments", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		// No user in context

		if assert.NoError(t, h.GetAllPayments(c)) {
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		}
	})
}

func TestPaymentHandler_GetPaymentDetail(t *testing.T) {
	e := echo.New()
	mockSvc := new(mocks.PaymentServiceInterface)
	h := handler.NewPaymentHandler(mockSvc, e, &config.Config{}, nil)

	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/payments/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")
		c.Set("user", entity.JwtUserData{ID: 1, RoleName: "Customer"})

		mockSvc.On("GetPaymentDetail", mock.Anything, int64(1), int64(1)).Return(&entity.PaymentEntity{ID: 1}, nil)

		if assert.NoError(t, h.GetPaymentDetail(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("not found 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/payments/99", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("99")
		c.Set("user", entity.JwtUserData{ID: 1, RoleName: "Customer"})

		mockSvc.On("GetPaymentDetail", mock.Anything, int64(99), int64(1)).Return(nil, errors.New("data not found"))

		if assert.NoError(t, h.GetPaymentDetail(c)) {
			assert.Equal(t, http.StatusNotFound, rec.Code)
		}
	})
}
