package cmd

import (
	"os"
	"os/signal"
	"product-service/config"
	"product-service/internal/adapter/message"
	"product-service/internal/adapter/repository"
	"product-service/internal/adapter/search"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Running background workers (Search Indexing & Stock Update)",
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

		mode := strings.ToLower(strings.TrimSpace(cfg.SearchEngine))
		if mode != "none" && mode != "off" && mode != "postgres" {
			go func() {
				baseDelay := 5 * time.Second
				maxDelay := 12 * time.Hour
				attempt := 0

				for {
					searchEngine, err := search.NewSearchEngine(cfg)
					if err == nil {
						log.Infof("Search engine (%s) connected, starting consumers...", searchEngine.EngineName())

						close(stopStockConsumer)

						searchProductRepo := repository.NewProductRepository(dbConn.DB, searchEngine)
						go message.StartIndexingConsumer(rabbitMQClient, searchEngine, queueEsName, productEvent)
						go message.StartStockUpdateConsumer(rabbitMQClient, searchProductRepo, queueStockName, searchEngine, make(chan struct{}))
						return
					}

					shift := attempt
					if shift > 16 {
						shift = 16
					}
					delay := baseDelay * time.Duration(1<<shift)
					if delay > maxDelay {
						delay = maxDelay
					}
					attempt++
					log.Warnf("Search engine not ready (attempt %d): %v, retrying in %v", attempt, err, delay)
					time.Sleep(delay)
				}
			}()
			log.Infof("Worker started. Stock consumer running, waiting for search engine...")
		} else {
			log.Infof("Worker started in PostgreSQL-only mode (SEARCH_ENGINE=%s). Stock consumer running.", mode)
		}

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit

		log.Info("Workers shutting down...")
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)
}
