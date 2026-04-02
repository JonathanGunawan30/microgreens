package service

import (
	"mime/multipart"
	"net/textproto"
	"testing"
	service2 "user-service/internal/core/service"
	"user-service/internal/mocks"
	"user-service/utils/message"

	"github.com/stretchr/testify/assert"
)

func TestUploadProfileImage(t *testing.T) {
	mockStorage := new(mocks.SupabaseInterface)
	imageSvc := service2.NewImageService(mockStorage)

	t.Run("File Too Large", func(t *testing.T) {
		file := &multipart.FileHeader{
			Size: 2 * 1024 * 1024, // 2MB
		}
		url, err := imageSvc.UploadProfileImage(file)
		assert.Error(t, err)
		assert.Equal(t, message.ErrFileTooLarge, err)
		assert.Empty(t, url)
	})

	t.Run("Invalid Mime Type", func(t *testing.T) {
		file := &multipart.FileHeader{
			Size: 500 * 1024,
			Header: textproto.MIMEHeader{
				"Content-Type": []string{"application/pdf"},
			},
		}
		url, err := imageSvc.UploadProfileImage(file)
		assert.Error(t, err)
		assert.Equal(t, message.ErrInvalidMime, err)
		assert.Empty(t, url)
	})
}
