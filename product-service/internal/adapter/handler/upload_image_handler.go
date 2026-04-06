package handler

import (
	"errors"
	"net/http"
	"product-service/config"
	"product-service/internal/adapter"
	"product-service/internal/adapter/handler/request"
	"product-service/internal/adapter/handler/response"
	"product-service/internal/core/service"
	"product-service/utils/message"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
)

type UploadImageInterface interface {
	UploadImage(c echo.Context) error
	RemoveImage(c echo.Context) error
}

type UploadImage struct {
	imageService service.ImageServiceInterface
	cfg          *config.Config
}

func NewUploadImage(e *echo.Echo, imageService service.ImageServiceInterface, cfg *config.Config, redisClient *redis.Client) UploadImageInterface {
	uploadImageHandler := &UploadImage{imageService: imageService, cfg: cfg}

	mid := adapter.NewMiddlewareAdapter(cfg, redisClient)

	adminGroup := e.Group("/admin", mid.CheckToken(cfg.App.JwtSecretKey))
	adminGroup.POST("/image-upload", uploadImageHandler.UploadImage)
	adminGroup.DELETE("/image", uploadImageHandler.RemoveImage)

	authGroup := e.Group("/auth", mid.CheckToken(cfg.App.JwtSecretKey))
	authGroup.POST("/image-upload", uploadImageHandler.UploadImage)
	authGroup.DELETE("/image", uploadImageHandler.RemoveImage)

	return uploadImageHandler
}

// UploadImage godoc
// @Summary Upload image (admin/auth)
// @Description Upload an image file to storage
// @Tags images
// @Accept mpfd
// @Produce json
// @Security BearerAuth
// @Param image formData file true "Image file to upload"
// @Success 200 {object} response.DefaultResponse{data=map[string]string} "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request - Invalid file or size"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /admin/image-upload [post]
// @Router /auth/image-upload [post]
func (u *UploadImage) UploadImage(c echo.Context) error {
	var resp = response.DefaultResponse{}
	file, err := c.FormFile("image")
	if err != nil {
		log.Errorf("[UploadImageHandler-1] UploadImage: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	url, err := u.imageService.UploadProfileImage(file)
	if err != nil {
		switch {
		case errors.Is(err, message.ErrInvalidMime):
			log.Errorf("[UploadImageHandler-2] UploadImage: %v", err)
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusBadRequest, resp)

		case errors.Is(err, message.ErrFileTooLarge):
			log.Errorf("[UploadImageHandler-2] UploadImage: %v", err)
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusBadRequest, resp)

		default:
			log.Errorf("[UploadImageHandler-2] UploadImage: %v", err)
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusInternalServerError, resp)
		}
	}

	resp.Message = "Success"
	resp.Data = map[string]string{"image_url": url}

	return c.JSON(http.StatusOK, resp)
}

// RemoveImage godoc
// @Summary Remove image (admin/auth)
// @Description Delete an image from storage by its URL
// @Tags images
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.RemoveImageRequest true "Remove Image Request"
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request - URL required"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /admin/image [delete]
// @Router /auth/image [delete]
func (u *UploadImage) RemoveImage(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		req  request.RemoveImageRequest
	)

	if err := c.Bind(&req); err != nil {
		log.Errorf("[UploadImageHandler-3] RemoveImage: %v", err)
		resp.Message = "invalid request body"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if req.ImageURL == "" {
		resp.Message = "image_url is required"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	err := u.imageService.DeleteProfileImage(req.ImageURL, u.cfg.Supabase.Bucket)
	if err != nil {
		log.Errorf("[UploadImageHandler-4] RemoveImage: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil

	return c.JSON(http.StatusOK, resp)
}
