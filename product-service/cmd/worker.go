package cmd

import (
	"os"
	"os/signal"
	"product-service/config"
	"product-service/internal/adapter/message"
	"product-service/internal/adapter/repository"
	"syscall"
	"time"

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

		dbConn, err := cfg.ConnectionPostgres()
		if err != nil {
			log.Fatalf("Failed to connect to Postgres: %v", err)
		}

		productEvent := cfg.ExchangeName.ProductEvent
		queueEsName := cfg.QueueName.ProductES
		queueStockName := cfg.RabbitMQ.QueueStockUpdate

		if queueEsName == "" || queueStockName == "" || productEvent == "" {
			log.Fatalf("Queue/Exchange name are empty in .env!")
		}

		productRepo := repository.NewProductRepository(dbConn.DB, nil)
		stopStockConsumer := make(chan struct{})
		go message.StartStockUpdateConsumer(rabbitMQClient, productRepo, queueStockName, nil, stopStockConsumer)

		go func() {
			baseDelay := 5 * time.Second
			maxDelay := 12 * time.Hour
			attempt := 0

			for {
				esClient, err := cfg.NewElasticsearchClient()
				if err == nil {
					log.Info("Elasticsearch connected, starting ES consumers...")

					close(stopStockConsumer)

					esProductRepo := repository.NewProductRepository(dbConn.DB, esClient)
					go message.StartIndexingConsumer(rabbitMQClient, esClient, queueEsName, productEvent)
					go message.StartStockUpdateConsumer(rabbitMQClient, esProductRepo, queueStockName, esClient, make(chan struct{}))
					return
				}

				attempt++
				delay := baseDelay * time.Duration(1<<attempt)
				if delay > maxDelay {
					delay = maxDelay
				}
				log.Warnf("Elasticsearch not ready (attempt %d), retrying in %v", attempt, delay)
				time.Sleep(delay)
			}
		}()

		log.Infof("Worker started. Stock consumer running, waiting for Elasticsearch...")

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit

		log.Info("Workers shutting down...")
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)
}
