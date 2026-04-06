package app

import (
	"context"
	"notification-service/config"
	"notification-service/internal/adapter/email"
	"notification-service/internal/adapter/handler"
	"notification-service/internal/adapter/message"
	"notification-service/internal/adapter/repository"
	"notification-service/internal/adapter/router"
	"notification-service/internal/core/service"
	"notification-service/utils/constant"
	"notification-service/utils/validator"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10/translations/en"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

func RunServer() {
	cfg := config.NewConfig()

	db, err := cfg.ConnectionPostgres()
	if err != nil {
		log.Fatalf("[RunServer-1] Failed to connect to database: %v", err)
	}

	redisClient := config.NewRedisClient(cfg)

	rabbitMQClient, err := cfg.NewRabbitMQClient()
	if err != nil {
		log.Fatalf("[RunServer - 2] Failed to connect to RabbbitMQ: %v", err)
	}

	defer rabbitMQClient.Close()

	smtp := email.NewSmtpEmail(cfg)

	notificationRepository := repository.NewNotificationRepository(db.DB)

	notificationService := service.NewNotificationService(smtp, notificationRepository, cfg)

	notificationHandler := handler.NewNotificationHandler(notificationService)

	go message.ConsumeMessage(rabbitMQClient, constant.NOTIF_EMAIL_VERIFICATION, notificationService)
	go message.ConsumeMessage(rabbitMQClient, constant.NOTIF_EMAIL_FORGOT_PASSWORD, notificationService)
	go message.ConsumeMessage(rabbitMQClient, constant.NOTIF_EMAIL_UPDATE_CUSTOMER, notificationService)
	go message.ConsumeMessage(rabbitMQClient, constant.NOTIF_EMAIL_CREATE_CUSTOMER, notificationService)
	go message.ConsumeMessage(rabbitMQClient, constant.NOTIF_EMAIL_UPDATE_STATUS_ORDER, notificationService)
	go message.ConsumeMessage(rabbitMQClient, constant.TypePush, notificationService)
	go message.OrderEmailNotificationConsumer(rabbitMQClient, constant.ORDER_EMAIL_QUEUE, cfg.ExchangeName.OrderEvent, notificationService)
	go message.OrderPushNotificationConsumer(rabbitMQClient, constant.ORDER_PUSH_QUEUE, cfg.ExchangeName.OrderEvent, notificationService)

	e := echo.New()
	e.Use(middleware.CORS())

	customValidator := validator.NewValidator()
	err = en.RegisterDefaultTranslations(customValidator.Validator, customValidator.Translator)
	if err != nil {
		log.Fatalf("[RunServer-4] Failed to register validator translations: %v", err)
	}
	e.Validator = customValidator

	e.GET("/api/check", func(c echo.Context) error {
		return c.String(200, "OK")
	})

	router.RegisterNotificationRoutes(e, notificationHandler, cfg, redisClient)
	handler.NewWebSocketHandler(e)

	go func() {
		if cfg.App.AppPort == "" {
			cfg.App.AppPort = os.Getenv("APP_PORT")
		}
		err = e.Start(":" + cfg.App.AppPort)
		if err != nil {
			log.Fatalf("[RunServer-5] Failed to start server: %v", err)
		}
	}()
	log.Info("[RunServer-4] Server is running on port ", cfg.App.AppPort)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	signal.Notify(quit, syscall.SIGTERM)
	<-quit

	log.Info("[RunServer-6] Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	e.Shutdown(ctx)
}
