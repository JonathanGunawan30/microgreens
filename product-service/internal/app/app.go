package app

import (
	"context"
	"os"
	"os/signal"
	"product-service/config"
	"product-service/internal/adapter/handler"
	"product-service/internal/adapter/repository"
	"product-service/internal/adapter/storage"
	"product-service/internal/core/service"
	"product-service/utils/validator"
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
	supabaseStorage := storage.NewSupabaseStorage(cfg)
	rabbitMQClient, err := cfg.NewRabbitMQClient()
	if err != nil {
		log.Fatalf("[RunServer - 2] Failed to connect to RabbbitMQ: %v", err)
	}
	elasticsearchClient, err := cfg.NewElasticsearchClient()
	if err != nil {
		log.Warnf("[RunServer - 3] Failed to connect to Elasticsearch: %v", err)
		elasticsearchClient = nil
	}

	defer rabbitMQClient.Close()

	categoryRepository := repository.NewCategoryRepository(db.DB)
	productRepository := repository.NewProductRepository(db.DB, elasticsearchClient)
	cartRepository := repository.NewCartRedisRepository(redisClient)

	categoryService := service.NewCategoryService(categoryRepository)
	productService := service.NewProductService(productRepository, rabbitMQClient, cfg.ExchangeName.ProductEvent, categoryRepository)
	imageService := service.NewImageService(supabaseStorage)
	cartService := service.NewCartService(cartRepository, productRepository)

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

	handler.NewCategoryHandler(e, categoryService, cfg, redisClient)
	handler.NewProductHandler(e, productService, cfg, redisClient)
	handler.NewUploadImage(e, imageService, cfg, redisClient)
	handler.NewCartHandler(e, cartService, cfg, redisClient)

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
