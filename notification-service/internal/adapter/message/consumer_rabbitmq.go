package message

import (
	"context"
	"encoding/json"
	"notification-service/internal/core/domain/entity"
	"notification-service/internal/core/service"

	"github.com/labstack/gommon/log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func ConsumeMessage(conn *amqp.Connection, queueName string, notifService *service.NotificationService) {
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	err = ch.Qos(1, 0, false)
	if err != nil {
		log.Fatalf("Failed to set QoS: %v", err)
	}

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	log.Infof("Consumer Notification started on queue: %s", queueName)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var notif entity.NotificationEntity
			if err := json.Unmarshal(d.Body, &notif); err != nil {
				log.Errorf("Error decoding message: %v", err)
				d.Nack(false, false)
				continue
			}

			if err := notifService.ProcessNotification(context.Background(), notif); err != nil {
				log.Errorf("Failed to send email notification: %v", err)
				d.Nack(false, true)
				continue
			}

			log.Info("Successfully send email notification")
			d.Ack(false)
		}
	}()

	log.Info("Waiting for messages. To exit press CTRL+C")
	<-forever
}
