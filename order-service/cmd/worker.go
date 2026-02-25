package cmd

import (
	"order-service/config"
	"order-service/internal/adapter/message"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/gommon/log"
	"github.com/spf13/cobra"
)

var workerOrderCmd = &cobra.Command{
	Use:   "worker-order",
	Short: "Running background worker for Order Indexing",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("Starting Worker Order Service...")
		cfg := config.NewConfig()

		rabbitMQClient, err := cfg.NewRabbitMQClient()
		if err != nil {
			log.Fatalf("Failed to connect to rabbitMQ: %v", err)
		}
		defer rabbitMQClient.Close()

		esClient, err := cfg.NewElasticsearchClient()
		if err != nil {
			log.Fatalf("Failed to connect to Elasticsearch: %v", err)
		}

		queueOrderName := cfg.PublisherName.OrderPublish

		if queueOrderName == "" {
			log.Fatalf("Queue/Publisher name are empty in .env!")
		}

		log.Infof("Dependencies ready. Spawning Order consumers...")

		go message.StartOrderConsumer(rabbitMQClient, queueOrderName, esClient)

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		log.Info("Worker Order shutting down...")
	},
}

var workerPaymentCmd = &cobra.Command{
	Use:   "worker-payment",
	Short: "Running background worker for Payment Success",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("Starting Worker Payment Service...")
		cfg := config.NewConfig()

		rabbitMQClient, err := cfg.NewRabbitMQClient()
		if err != nil {
			log.Fatalf("Failed to connect to rabbitMQ: %v", err)
		}
		defer rabbitMQClient.Close()

		esClient, err := cfg.NewElasticsearchClient()
		if err != nil {
			log.Fatalf("Failed to connect to Elasticsearch: %v", err)
		}

		queuePaymentName := cfg.PublisherName.PublisherPaymentSuccess

		if queuePaymentName == "" {
			log.Fatalf("Queue/Publisher name are empty in .env!")
		}

		log.Infof("Dependencies ready. Spawning payment consumers...")

		go message.ConsumePaymentSuccess(rabbitMQClient, queuePaymentName, esClient)

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		log.Info("Worker Payment shutting down...")
	},
}

var workerStatusCmd = &cobra.Command{
	Use:   "worker-update-status",
	Short: "Running background worker for Update Status",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("Starting Worker Update Status Service...")
		cfg := config.NewConfig()

		rabbitMQClient, err := cfg.NewRabbitMQClient()
		if err != nil {
			log.Fatalf("Failed to connect to rabbitMQ: %v", err)
		}
		defer rabbitMQClient.Close()

		esClient, err := cfg.NewElasticsearchClient()
		if err != nil {
			log.Fatalf("Failed to connect to Elasticsearch: %v", err)
		}

		queueUpdateStatus := cfg.PublisherName.PublisherUpdateStatus

		if queueUpdateStatus == "" {
			log.Fatalf("Queue/Publisher name are empty in .env!")
		}

		log.Infof("Dependencies ready. Spawning payment consumers...")

		go message.ConsumeUpdateStatus(rabbitMQClient, queueUpdateStatus, esClient)

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		log.Info("Worker Update Status shutting down...")
	},
}

func init() {
	rootCmd.AddCommand(workerOrderCmd)
	rootCmd.AddCommand(workerPaymentCmd)
	rootCmd.AddCommand(workerStatusCmd)
}
