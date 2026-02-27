package app

import (
	"context"
	"os"
	"os/signal"
	"payment-service/config"
	"payment-service/internal/adapter"
	"payment-service/internal/adapter/handler"
	"payment-service/internal/adapter/repository"
	"payment-service/internal/core/service"
	"payment-service/utils/validator"
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

	paymentRepository := repository.NewPaymentRepository(db.DB)

	httpClient := adapter.NewHttpClient(cfg)
	midtransClient := adapter.NewMidtransClient(cfg)

	paymentService := service.NewPaymentService(paymentRepository, httpClient, midtransClient, rabbitMQClient, cfg)

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

	handler.NewPaymentHandler(paymentService, e, cfg, redisClient)

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
