package handler

import (
	"errors"
	"net/http"
	"user-service/config"
	"user-service/internal/adapter"
	"user-service/internal/adapter/handler/response"
	"user-service/internal/core/service"
	"user-service/utils/message"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
)

type UploadImageInterface interface {
	UploadImage(c echo.Context) error
}

type UploadImage struct {
	imageService service.ImageServiceInterface
}

func NewUploadImage(e *echo.Echo, imageService service.ImageServiceInterface, cfg *config.Config, redisClient *redis.Client) UploadImageInterface {
	uploadImageHandler := &UploadImage{imageService: imageService}

	mid := adapter.NewMiddlewareAdapter(cfg, redisClient)
	e.POST("/auth/profile/image-upload", uploadImageHandler.UploadImage, mid.CheckToken(cfg.App.JwtSecretKey))

	return uploadImageHandler
}

// UploadImage godoc
// @Summary Upload profile image
// @Description Upload a profile image to storage
// @Tags users
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param image formData file true "Image File"
// @Success 200 {object} response.DefaultResponse{data=map[string]string} "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /auth/profile/image-upload [post]
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
