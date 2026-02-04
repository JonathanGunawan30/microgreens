package cmd

import (
	"os"
	"os/signal"
	"product-service/config"
	"product-service/internal/adapter/message"
	"product-service/internal/adapter/repository"
	"syscall"

	"github.com/labstack/gommon/log"
	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Running background workers (ES Indexing & Stock Update)",
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("Starting Worker Service...")
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

		dbConn, err := cfg.ConnectionPostgres()
		if err != nil {
			log.Fatalf("Failed to connect to Postgres: %v", err)
		}

		productRepository := repository.NewProductRepository(dbConn.DB, esClient)

		queueEsName := cfg.RabbitMQ.QueueEsIndexing
		queueStockName := cfg.RabbitMQ.QueueStockUpdate

		if queueEsName == "" || queueStockName == "" {
			log.Fatalf("Queue name are empty in .env!")
		}

		log.Infof("Depedencies ready. Spawning consumers...")

		go message.StartIndexingCustomer(rabbitMQClient, esClient, queueEsName)
		go message.StartStockUpdateConsumer(rabbitMQClient, productRepository, queueStockName)

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit

		log.Info("Workers shutting down...")
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)
}
