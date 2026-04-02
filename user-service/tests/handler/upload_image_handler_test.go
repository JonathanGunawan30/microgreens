package handler_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"user-service/config"
	"user-service/internal/adapter/handler"
	"user-service/internal/mocks"
	"user-service/utils/message"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupUploadImageHandlerTest() (*echo.Echo, *mocks.ImageServiceInterface, handler.UploadImageInterface) {
	e := echo.New()
	mockService := new(mocks.ImageServiceInterface)
	cfg := &config.Config{
		App: config.App{
			JwtSecretKey: "secret",
		},
	}
	h := handler.NewUploadImage(e, mockService, cfg, &redis.Client{})
	return e, mockService, h
}

func TestUploadImage(t *testing.T) {
	e, mockService, h := setupUploadImageHandlerTest()

	t.Run("Success", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "test.jpg")
		part.Write([]byte("fake image content"))
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/auth/profile/image-upload", body)
		req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockService.On("UploadProfileImage", mock.Anything).Return("http://example.com/image.jpg", nil).Once()

		if assert.NoError(t, h.UploadImage(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("Missing File", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/profile/image-upload", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if assert.NoError(t, h.UploadImage(c)) {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("File Too Large", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "large.jpg")
		part.Write([]byte("large content"))
		writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/auth/profile/image-upload", body)
		req.Header.Set(echo.HeaderContentType, writer.FormDataContentType())
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		mockService.On("UploadProfileImage", mock.Anything).Return("", message.ErrFileTooLarge).Once()

		if assert.NoError(t, h.UploadImage(c)) {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		}
	})
}
