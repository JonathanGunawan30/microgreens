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

func TestProductHandler_CreateProduct(t *testing.T) {
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	mockSvc := new(svcMocks.ProductServiceInterface)
	cfg := &config.Config{
		App: config.App{
			JwtSecretKey: "secret",
		},
	}
	h := handler2.NewProductHandler(e, mockSvc, cfg, nil)

	t.Run("success 201", func(t *testing.T) {
		reqBody := request.ProductRequest{
			Name:         "Product A",
			CategorySlug: "cat-a",
			Unit:         "kg",
			Variant:      1,
			Description:  "Desc",
			Status:       "active",
			VariantDetail: []request.ProductDetailRequest{
				{
					Stock:        10,
					Image:        "http://img.com/a.png",
					Weight:       100,
					SalePrice:    1000,
					RegulerPrice: 1200,
				},
			},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/admin/products", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockSvc.On("CreateProduct", mock.Anything, mock.MatchedBy(func(p entity.ProductEntity) bool {
			return p.Name == "Product A"
		})).Return(nil).Once()

		if assert.NoError(t, h.CreateProduct(c)) {
			assert.Equal(t, http.StatusCreated, rec.Code)
		}
	})

	t.Run("validation error 422", func(t *testing.T) {
		reqBody := request.ProductRequest{
			Name: "", // missing name
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/admin/products", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if assert.NoError(t, h.CreateProduct(c)) {
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		}
	})
}

func TestProductHandler_GetProductByID(t *testing.T) {
	e := echo.New()
	mockSvc := new(svcMocks.ProductServiceInterface)
	cfg := &config.Config{
		App: config.App{
			JwtSecretKey: "secret",
		},
	}
	h := handler2.NewProductHandler(e, mockSvc, cfg, nil)

	t.Run("success 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/products/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		mockSvc.On("GetProductByID", mock.Anything, int64(1)).Return(&entity.ProductEntity{ID: 1, Name: "Prod"}, nil).Once()

		if assert.NoError(t, h.GetProductByID(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("not found 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/products/2", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("2")

		mockSvc.On("GetProductByID", mock.Anything, int64(2)).Return(nil, message.ErrProductNotFound).Once()

		if assert.NoError(t, h.GetProductByID(c)) {
			assert.Equal(t, http.StatusNotFound, rec.Code)
		}
	})
}

func TestProductHandler_UpdateProduct(t *testing.T) {
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	mockSvc := new(svcMocks.ProductServiceInterface)
	cfg := &config.Config{
		App: config.App{
			JwtSecretKey: "secret",
		},
	}
	h := handler2.NewProductHandler(e, mockSvc, cfg, nil)

	t.Run("success 200", func(t *testing.T) {
		reqBody := request.ProductRequest{
			Name:         "Updated Product",
			CategorySlug: "cat-a",
			Unit:         "kg",
			Variant:      1,
			Description:  "Desc",
			Status:       "active",
			VariantDetail: []request.ProductDetailRequest{
				{
					Stock:        10,
					Image:        "http://img.com/a.png",
					Weight:       100,
					SalePrice:    1000,
					RegulerPrice: 1200,
				},
			},
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPut, "/admin/products/1", bytes.NewBuffer(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		mockSvc.On("UpdateProduct", mock.Anything, mock.Anything).Return(nil).Once()

		if assert.NoError(t, h.UpdateProduct(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})
}

func TestProductHandler_DeleteProductByID(t *testing.T) {
	e := echo.New()
	mockSvc := new(svcMocks.ProductServiceInterface)
	cfg := &config.Config{
		App: config.App{
			JwtSecretKey: "secret",
		},
	}
	h := handler2.NewProductHandler(e, mockSvc, cfg, nil)

	t.Run("success 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/admin/products/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		mockSvc.On("DeleteProductByID", mock.Anything, int64(1)).Return(nil).Once()

		if assert.NoError(t, h.DeleteProductByID(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("internal server error 500", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/admin/products/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")

		mockSvc.On("DeleteProductByID", mock.Anything, int64(1)).Return(errors.New("error")).Once()

		if assert.NoError(t, h.DeleteProductByID(c)) {
			assert.Equal(t, http.StatusInternalServerError, rec.Code)
		}
	})
}
