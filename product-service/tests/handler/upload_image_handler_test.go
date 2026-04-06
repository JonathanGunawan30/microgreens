package handler

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"product-service/config"
	handler2 "product-service/internal/adapter/handler"
	svcMocks "product-service/tests/mocks/service"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUploadImage_UploadImage(t *testing.T) {
	e := echo.New()
	mockSvc := new(svcMocks.ImageServiceInterface)
	cfg := &config.Config{
		App: config.App{
			JwtSecretKey: "secret",
		},
	}
	h := handler2.NewUploadImage(e, mockSvc, cfg, nil)

	t.Run("success 200", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "test.png")
		part.Write([]byte("fake image content"))
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/admin/image-upload", body)
		req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockSvc.On("UploadProfileImage", mock.Anything).Return("http://img.com/a.png", nil).Once()

		if assert.NoError(t, h.UploadImage(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("missing file 400", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/admin/image-upload", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if assert.NoError(t, h.UploadImage(c)) {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		}
	})
}
