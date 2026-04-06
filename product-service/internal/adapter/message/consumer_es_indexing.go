package message

import (
	"context"
	"encoding/json"
	"fmt"
	"product-service/internal/core/domain/entity"

	"github.com/labstack/gommon/log"

	"github.com/elastic/go-elasticsearch/v8"
	amqp "github.com/rabbitmq/amqp091-go"
)

func StartIndexingConsumer(conn *amqp.Connection, esClient *elasticsearch.TypedClient, queueName, exchangeName string) {
	if conn == nil {
		return
	}
	ch, err := conn.Channel()
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

			err := processMessage(d.Body, esClient)

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

func processMessage(body []byte, esClient *elasticsearch.TypedClient) error {
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

		_, err := esClient.Index("products").
			Id(fmt.Sprintf("%d", msg.Data.ID)).
			Request(msg.Data).
			Do(ctx)

		if err != nil {
			return fmt.Errorf("error indexing/updating to elasticsearch: %v", err)
		}
		log.Infof("Successfully indexed/updated product ID: %v", msg.Data.ID)
	case entity.ActionDelete:
		if msg.ID == 0 {
			return fmt.Errorf("action DELETE but ID is missing")
		}

		_, err := esClient.Delete("products", fmt.Sprintf("%d", msg.ID)).Do(ctx)

		if err != nil {
			return fmt.Errorf("error deleting from elasticsearch: %v", err)
		}
		log.Infof("Successfully deleted product ID: %v", msg.ID)
	default:
		log.Warnf("Unknown action received: %s", msg.Action)
	}

	return nil
}
