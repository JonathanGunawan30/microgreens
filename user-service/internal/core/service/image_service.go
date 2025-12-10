package service

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"time"
	"user-service/internal/adapter/storage"
	"user-service/utils/message"

	"github.com/google/uuid"
)

var allowedMime = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/jpg":  true,
	"image/webp": true,
}

type ImageServiceInterface interface {
	UploadProfileImage(file *multipart.FileHeader) (string, error)
}

type imageService struct {
	storage storage.SupabaseInterface
}

func NewImageService(storage storage.SupabaseInterface) ImageServiceInterface {
	return &imageService{
		storage: storage,
	}
}

func (i *imageService) UploadProfileImage(file *multipart.FileHeader) (string, error) {
	if file.Size > 1024*1024 {
		return "", message.ErrFileTooLarge
	}

	contentType := file.Header.Get("Content-Type")
	if !allowedMime[contentType] {
		return "", message.ErrInvalidMime
	}

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, src)
	if err != nil {
		return "", err
	}

	filename := fmt.Sprintf(
		"%s_%d%s",
		uuid.New().String(),
		time.Now().Unix(),
		filepath.Ext(file.Filename),
	)

	path := "uploads/" + filename

	return i.storage.UploadFile(path, buf, contentType)
}
