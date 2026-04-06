package router

import (
	"notification-service/config"
	"notification-service/internal/adapter"
	"notification-service/internal/adapter/handler"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/redis/go-redis/v9"
)

func RegisterNotificationRoutes(e *echo.Echo, handler *handler.NotificationHandler, cfg *config.Config, redis *redis.Client) {

	e.Use(middleware.Recover())
	mid := adapter.NewMiddlewareAdapter(cfg, redis)

	authGroup := e.Group("/auth", mid.CheckToken(cfg.App.JwtSecretKey))

	authGroup.GET("/notifications", handler.GetAll)
	authGroup.PATCH("/notifications/read-all", handler.ReadAll)
	authGroup.GET("/notifications/:id", handler.GetByID)
	authGroup.PATCH("/notifications/:id/read", handler.Read)
}
