package message

import (
	"context"
	"encoding/json"
	"payment-service/internal/adapter/repository"
	"payment-service/internal/core/domain/entity"

	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

func ConsumeUserSnapshot(conn *amqp.Connection, queueName, exchangeName string, repo repository.UserSnapshotRepositoryInterface) {
	if conn == nil {
		log.Errorf("[ConsumeUserSnapshot] RabbitMQ connection is nil, skipping consumer")
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
		log.Fatalf("Failed to consume message: %v", err)
	}

	log.Info("Consumer user snapshot started on queue: %s", queueName)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var userSnapshot entity.UserSnapshotEntity
			if err := json.Unmarshal(d.Body, &userSnapshot); err != nil {
				log.Errorf("Error decoding message: %v", err)
				d.Nack(false, false)
				continue
			}

			if err := repo.Upsert(context.Background(), userSnapshot); err != nil {
				log.Errorf("Error Upsert to User Snapshot: %v", err)
				d.Nack(false, true)
				continue
			}

			log.Infof("Successfully upsert user snapshot with user ID: %d", userSnapshot.UserID)
			d.Ack(false)
		}
	}()

	log.Info("Waiting for message, to exit press CTRL+C")
	<-forever
}

func ConsumeOrderSnapshot(conn *amqp.Connection, queueName, exchangeName string, repo repository.OrderSnapshotRepositoryInterface) {
	if conn == nil {
		log.Errorf("[ConsumeOrderSnapshot] RabbitMQ connection is nil, skipping consumer")
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
		log.Fatalf("Failed to consume message: %v", err)
	}

	log.Info("Consumer order snapshot started on queue: %s", queueName)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var event entity.OrderEvent
			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Errorf("Error decoding message: %v", err)
				d.Nack(false, false)
				continue
			}

			err := repo.Upsert(context.Background(), entity.OrdersSnapshotEntity{
				OrderID:      event.ID,
				OrderCode:    event.OrderCode,
				TotalAmount:  event.TotalAmount,
				ShippingType: event.ShippingType,
				Remarks:      event.Remarks,
				OrderDate:    event.OrderDate,
				OrderTime:    event.OrderTime,
			})

			if err != nil {
				log.Errorf("Failed to upsert: %v", err)
				d.Nack(false, true)
				continue
			}

			log.Infof("Upserted order ID: %d", event.ID)
			d.Ack(false)
		}
	}()

	log.Info("Waiting for message, to exit press CTRL+C")
	<-forever
}
