package cmd

import (
	"order-service/config"
	"order-service/internal/adapter/message"
	"order-service/internal/adapter/repository"
	"order-service/utils/helper"
	"os"
	"os/signal"
	"syscall"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/labstack/gommon/log"
	"github.com/spf13/cobra"
)

var workerOrderCmd = &cobra.Command{
	Use: "worker-order",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("Starting Worker Order Service...")
		cfg := config.NewConfig()

		rabbitMQClient, err := cfg.NewRabbitMQClient()
		if err != nil {
			log.Fatalf("Failed to connect to rabbitMQ: %v", err)
		}
		defer rabbitMQClient.Close()

		queueOrderName := cfg.PublisherName.OrderPublish
		eventOrderName := cfg.ExchangeName.OrderEvent

		if queueOrderName == "" {
			log.Fatalf("Queue/Publisher name are empty in .env!")
		}

		go helper.RetryElasticsearch(cfg, func(esClient *elasticsearch.TypedClient) {
			go message.StartOrderConsumer(rabbitMQClient, queueOrderName, eventOrderName, esClient)
		})

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		log.Info("Worker Order shutting down...")
	},
}

var workerPaymentCmd = &cobra.Command{
	Use: "worker-payment",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("Starting Worker Payment Service...")
		cfg := config.NewConfig()

		rabbitMQClient, err := cfg.NewRabbitMQClient()
		if err != nil {
			log.Fatalf("Failed to connect to rabbitMQ: %v", err)
		}
		defer rabbitMQClient.Close()

		db, err := cfg.ConnectionPostgres()
		if err != nil {
			log.Fatalf("Failed to connect to database postgres: %v", err)
		}

		orderRepository := repository.NewOrderRepository(db.DB)

		queueUpdatePaymentMethodDB := cfg.QueueName.UpdatePaymentMethodDB
		queueUpdatePaymentMethodES := cfg.QueueName.UpdatePaymentMethodES
		exchangeUpdatePaymentMethod := cfg.ExchangeName.PaymentEvent

		if queueUpdatePaymentMethodDB == "" || queueUpdatePaymentMethodES == "" || exchangeUpdatePaymentMethod == "" {
			log.Fatalf("Queue/Exchange name are empty in .env!")
		}

		go message.ConsumeUpdatePaymentMethodDB(rabbitMQClient, queueUpdatePaymentMethodDB, exchangeUpdatePaymentMethod, orderRepository)

		go helper.RetryElasticsearch(cfg, func(esClient *elasticsearch.TypedClient) {
			go message.ConsumeUpdatePaymentMethodES(rabbitMQClient, queueUpdatePaymentMethodES, exchangeUpdatePaymentMethod, esClient)
		})

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		log.Info("Worker Payment shutting down...")
	},
}

var workerStatusCmd = &cobra.Command{
	Use: "worker-update-status",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("Starting Worker Update Status Service...")
		cfg := config.NewConfig()

		rabbitMQClient, err := cfg.NewRabbitMQClient()
		if err != nil {
			log.Fatalf("Failed to connect to rabbitMQ: %v", err)
		}
		defer rabbitMQClient.Close()

		queueUpdateStatus := cfg.PublisherName.PublisherUpdateStatus

		if queueUpdateStatus == "" {
			log.Fatalf("Queue/Publisher name are empty in .env!")
		}

		go helper.RetryElasticsearch(cfg, func(esClient *elasticsearch.TypedClient) {
			go message.ConsumeUpdateStatus(rabbitMQClient, queueUpdateStatus, esClient)
		})

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		log.Info("Worker Update Status shutting down...")
	},
}

var workerUserSnapshot = &cobra.Command{
	Use:   "worker-user-snapshot",
	Short: "Running background worker for Upsert User Snapshot",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("Starting Worker User Snapshot...")
		cfg := config.NewConfig()

		rabbitMQClient, err := cfg.NewRabbitMQClient()
		if err != nil {
			log.Fatalf("Failed to connect to rabbitMQ: %v", err)
		}
		defer rabbitMQClient.Close()

		db, err := cfg.ConnectionPostgres()
		if err != nil {
			log.Fatalf("Failed to connect to database postgres: %v", err)
		}

		userSnapshotRepository := repository.NewUserSnapshotRepository(db.DB)

		queueUserSnapshot := cfg.QueueName.UserSnapshot

		exchangeUserSnapshot := cfg.ExchangeName.UserEvent

		if queueUserSnapshot == "" || exchangeUserSnapshot == "" {
			log.Fatalf("Queue/Exchange name are empty in .env!")
		}

		log.Infof("Dependencies ready. Spawning user snapshot consumers...")

		go message.ConsumeUserSnapshot(rabbitMQClient, queueUserSnapshot, exchangeUserSnapshot, userSnapshotRepository)

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		log.Info("Worker User Snapshot shutting down...")
	},
}

var workerProductSnapshot = &cobra.Command{
	Use:   "worker-product-snapshot",
	Short: "Running background worker for Upsert Product Snapshot",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("Starting Worker Product Snapshot...")
		cfg := config.NewConfig()

		rabbitMQClient, err := cfg.NewRabbitMQClient()
		if err != nil {
			log.Fatalf("Failed to connect to rabbitMQ: %v", err)
		}
		defer rabbitMQClient.Close()

		db, err := cfg.ConnectionPostgres()
		if err != nil {
			log.Fatalf("Failed to connect to database postgres: %v", err)
		}

		productSnapshotRepository := repository.NewProductSnapshotRepository(db.DB)

		queueProductSnapshot := cfg.QueueName.ProductSnapshot

		exchangeProductEvent := cfg.ExchangeName.ProductEvent

		if queueProductSnapshot == "" || exchangeProductEvent == "" {
			log.Fatalf("Queue/Exchange name are empty in .env!")
		}

		log.Infof("Dependencies ready. Spawning product snapshot  consumers...")

		go message.ConsumeProductSnapshot(rabbitMQClient, queueProductSnapshot, exchangeProductEvent, productSnapshotRepository)

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		log.Info("Worker Product Snapshot shutting down...")
	},
}

func init() {
	rootCmd.AddCommand(workerOrderCmd)
	rootCmd.AddCommand(workerPaymentCmd)
	rootCmd.AddCommand(workerStatusCmd)
	rootCmd.AddCommand(workerUserSnapshot)
	rootCmd.AddCommand(workerProductSnapshot)
}
