package storage

import (
	"io"
	"product-service/config"

	"github.com/labstack/gommon/log"
	"github.com/supabase-community/storage-go"
)

type SupabaseInterface interface {
	UploadFile(path string, file io.Reader, contentType string) (string, error)
}

type supabase struct {
	cfg *config.Config
}

func NewSupabaseStorage(cfg *config.Config) SupabaseInterface {
	return &supabase{cfg: cfg}
}

func (s *supabase) UploadFile(path string, file io.Reader, contentType string) (string, error) {
	client := storage_go.NewClient(s.cfg.Supabase.URL, s.cfg.Supabase.Key, map[string]string{"Content-Type": contentType})

	_, err := client.UploadFile(s.cfg.Supabase.Bucket, path, file)
	if err != nil {
		log.Errorf("[UploadFile] Failed to upload file: %v", err)
		return "", err
	}

	url := client.GetPublicUrl(s.cfg.Supabase.Bucket, path)

	return url.SignedURL, nil
}
