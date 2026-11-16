package message

import (
	"context"
	"encoding/json"
	"fmt"
	"product-service/config"
	"product-service/internal/adapter/search"
	"product-service/internal/core/domain/entity"

	"github.com/labstack/gommon/log"
)

func StartIndexingConsumer(client *config.RabbitMQClient, searchEngine search.SearchEngineInterface, queueName, exchangeName string) {
	if client == nil || searchEngine == nil {
		return
	}
	ch, err := client.GetConn().Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(exchangeName, "fanout", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare exchange: %v", err)
	}

	q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)

	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	err = ch.QueueBind(q.Name, "", exchangeName, false, nil)
	if err != nil {
		log.Fatalf("Failed to bind queue: %v", err)
	}

	err = ch.Qos(1, 0, false)
	if err != nil {
		log.Fatalf("Failed to set QoS: %v", err)
	}

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	log.Info("RabbitMQ Consumer started. Waiting for messages...")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Infof("Received a message: %s", d.Body)

			err := processMessage(d.Body, searchEngine)

			if err != nil {
				log.Errorf("Error processing message: %v", err)
				d.Nack(false, true)
				continue
			}

			d.Ack(false)
		}
	}()

	<-forever
}

func processMessage(body []byte, searchEngine search.SearchEngineInterface) error {
	var msg entity.ProductEvent
	if err := json.Unmarshal(body, &msg); err != nil {
		return fmt.Errorf("error decoding JSON wrapper: %v", err)
	}

	ctx := context.Background()

	switch msg.Action {
	case entity.ActionInsert:
		if msg.Data == nil {
			return fmt.Errorf("action INSERT but data is nil")
		}

		if err := searchEngine.IndexProduct(ctx, *msg.Data); err != nil {
			return fmt.Errorf("error indexing/updating to %s: %v", searchEngine.EngineName(), err)
		}
		log.Infof("Successfully indexed/updated product ID %v on %s", msg.Data.ID, searchEngine.EngineName())
	case entity.ActionDelete:
		if msg.ID == 0 {
			return fmt.Errorf("action DELETE but ID is missing")
		}

		if err := searchEngine.DeleteProduct(ctx, msg.ID); err != nil {
			return fmt.Errorf("error deleting from %s: %v", searchEngine.EngineName(), err)
		}
		log.Infof("Successfully deleted product ID %v on %s", msg.ID, searchEngine.EngineName())
	default:
		log.Warnf("Unknown action received: %s", msg.Action)
	}

	return nil
}
