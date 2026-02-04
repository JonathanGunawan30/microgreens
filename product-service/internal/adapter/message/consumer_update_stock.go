package message

import (
	"context"
	"encoding/json"
	"fmt"
	"product-service/internal/adapter/repository"
	"product-service/internal/core/domain/entity"

	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

func StartStockUpdateConsumer(conn *amqp.Connection, repo repository.ProductRepositoryInterface, queueName string) {
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

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Infof("[StockConsumer] Received payload: %s", d.Body)

			err := processStockUpdate(d.Body, repo)

			if err != nil {
				log.Errorf("[StockConsumer] Failed to process: %v", err)

				d.Nack(false, false)
			} else {
				d.Ack(false)
			}
		}
	}()

	<-forever
}

func processStockUpdate(body []byte, repo repository.ProductRepositoryInterface) error {
	var msg entity.StockUpdateMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return fmt.Errorf("invalid JSON: %v", err)
	}

	ctx := context.Background()

	product, err := repo.GetProductByID(ctx, msg.ProductID)
	if err != nil {
		return fmt.Errorf("product not found ID %d: %v", msg.ProductID, err)
	}

	if product.Stock < msg.Quantity {
		return fmt.Errorf("insufficient stock. ID: %d, Has: %d, Need: %d",
			product.ID, product.Stock, msg.Quantity)
	}

	product.Stock = product.Stock - msg.Quantity

	err = repo.DecreaseStock(ctx, msg.ProductID, msg.Quantity)
	if err != nil {
		return fmt.Errorf("failed to save stock update: %v", err)
	}

	log.Infof("Stock updated! Product ID %d remaining stock: %d", product.ID, product.Stock)
	return nil
}
