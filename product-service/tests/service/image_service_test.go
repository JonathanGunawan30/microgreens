package service

import (
	"mime/multipart"
	"net/textproto"
	service2 "product-service/internal/core/service"
	storageMocks "product-service/tests/mocks/storage"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImageService_UploadProfileImage(t *testing.T) {
	mockStorage := new(storageMocks.SupabaseInterface)
	svc := service2.NewImageService(mockStorage)

	t.Run("invalid mime", func(t *testing.T) {
		file := &multipart.FileHeader{
			Filename: "test.txt",
			Header:   make(textproto.MIMEHeader),
			Size:     100,
		}
		file.Header.Set("Content-Type", "text/plain")

		url, err := svc.UploadProfileImage(file)

		assert.Error(t, err)
		assert.Empty(t, url)
	})
}
