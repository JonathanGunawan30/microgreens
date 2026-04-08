package message

import (
	"context"
	"encoding/json"
	"fmt"
	"product-service/internal/adapter/repository"
	"product-service/internal/core/domain/entity"
	"strconv"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

func StartStockUpdateConsumer(conn *amqp.Connection, repo repository.ProductRepositoryInterface, queueName string, es *elasticsearch.TypedClient, stop <-chan struct{}) {
	if conn == nil {
		return
	}
	ch, err := conn.Channel()
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
			err := processStockUpdate(d.Body, repo, es)
			if err != nil {
				log.Errorf("[StockConsumer] Failed to process: %v", err)
				d.Nack(false, false)
			} else {
				d.Ack(false)
			}
		case <-stop:
			log.Info("StockUpdateConsumer stopping, handing off to ES-aware consumer...")
			return
		}
	}
}

func processStockUpdate(body []byte, repo repository.ProductRepositoryInterface, es *elasticsearch.TypedClient) error {
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

		if es != nil {
			if err := syncToES(ctx, es, parentProduct); err != nil {
				log.Warnf("Failed to sync parent to ES (non-fatal): %v", err)
			}
		} else {
			log.Warn("Elasticsearch not available, skipping ES sync for stock update")
		}
	}

	return nil
}

func syncToES(ctx context.Context, es *elasticsearch.TypedClient, product *entity.ProductEntity) error {
	_, err := es.Index("products").
		Id(strconv.FormatInt(product.ID, 10)).
		Request(product).
		Do(ctx)
	return err
}
