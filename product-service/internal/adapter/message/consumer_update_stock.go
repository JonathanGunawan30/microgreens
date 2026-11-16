package message

import (
	"context"
	"encoding/json"
	"fmt"
	"product-service/config"
	"product-service/internal/adapter/repository"
	"product-service/internal/adapter/search"
	"product-service/internal/core/domain/entity"

	"github.com/labstack/gommon/log"
)

func StartStockUpdateConsumer(client *config.RabbitMQClient, repo repository.ProductRepositoryInterface, queueName string, searchEngine search.SearchEngineInterface, stop <-chan struct{}) {
	if client == nil {
		return
	}
	ch, err := client.GetConn().Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	err = ch.Qos(1, 0, false)
	if err != nil {
		log.Fatalf("Failed to set QoS: %v", err)
	}

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	log.Infof("Consumer Stock Update started on queue: %s", queueName)

	for {
		select {
		case d, ok := <-msgs:
			if !ok {
				log.Warn("StockUpdateConsumer: channel closed")
				return
			}
			log.Infof("[StockConsumer] Received payload: %s", d.Body)
			err := processStockUpdate(d.Body, repo, searchEngine)
			if err != nil {
				log.Errorf("[StockConsumer] Failed to process: %v", err)
				d.Nack(false, false)
			} else {
				d.Ack(false)
			}
		case <-stop:
			log.Info("StockUpdateConsumer stopping, handing off to search-aware consumer...")
			return
		}
	}
}

func processStockUpdate(body []byte, repo repository.ProductRepositoryInterface, searchEngine search.SearchEngineInterface) error {
	var msg entity.StockUpdateMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}

	ctx := context.Background()

	if err := repo.DecreaseStock(ctx, msg.ProductID, msg.Quantity); err != nil {
		return err
	}

	childProduct, err := repo.GetProductByID(ctx, msg.ProductID)
	if err != nil {
		return fmt.Errorf("failed to fetch child product: %v", err)
	}

	if childProduct.ParentID != nil {
		parentProduct, err := repo.GetProductByID(ctx, *childProduct.ParentID)
		if err != nil {
			return fmt.Errorf("failed to fetch parent product: %v", err)
		}

		log.Infof("Stock updated! Child ID %d, Parent ID %d remaining stock: %d",
			childProduct.ID, parentProduct.ID, parentProduct.Stock)

		if searchEngine != nil {
			if err := searchEngine.IndexProduct(ctx, *parentProduct); err != nil {
				log.Warnf("Failed to sync parent to %s (non-fatal): %v", searchEngine.EngineName(), err)
			}
		} else {
			log.Warn("Search engine not available, skipping search index sync for stock update")
		}
	}

	return nil
}
