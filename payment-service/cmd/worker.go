package cmd

import (
	"os"
	"os/signal"
	"payment-service/config"
	"payment-service/internal/adapter/message"
	"payment-service/internal/adapter/repository"
	"syscall"

	"github.com/labstack/gommon/log"
	"github.com/spf13/cobra"
)

var workerOrderSnapshot = &cobra.Command{
	Use:   "worker-order-snapshot",
	Short: "Running background worker for Upsert Order Snapshot",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("Starting Worker Order Snapshot...")
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

		ordersSnapshotRepository := repository.NewOrdersSnapshotRepository(db.DB)

		queueOrderSnapshot := cfg.QueueName.OrderSnapshot

		exchangeOrderEvent := cfg.ExchangeName.OrderEvent

		if queueOrderSnapshot == "" || exchangeOrderEvent == "" {
			log.Fatalf("Queue/Exchange name are empty in .env!")
		}

		log.Infof("Dependencies ready. Spawning order snapshot consumers...")

		go message.ConsumeOrderSnapshot(rabbitMQClient, queueOrderSnapshot, exchangeOrderEvent, ordersSnapshotRepository)

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		log.Info("Worker Order Snapshot shutting down...")
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

		usersSnapshotRepository := repository.NewUserSnapshotRepository(db.DB)

		queueUserSnapshot := cfg.QueueName.UserSnapshot

		exchangeUserEvent := cfg.ExchangeName.UserEvent

		if queueUserSnapshot == "" || exchangeUserEvent == "" {
			log.Fatalf("Queue/Exchange name are empty in .env!")
		}

		log.Infof("Dependencies ready. Spawning user snapshot consumers...")

		go message.ConsumeUserSnapshot(rabbitMQClient, queueUserSnapshot, exchangeUserEvent, usersSnapshotRepository)

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
		log.Info("Worker User Snapshot shutting down...")
	},
}

func init() {
	rootCmd.AddCommand(workerOrderSnapshot)
	rootCmd.AddCommand(workerUserSnapshot)
}
