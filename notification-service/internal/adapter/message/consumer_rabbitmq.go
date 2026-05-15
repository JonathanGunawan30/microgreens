package message

import (
	"context"
	"encoding/json"
	"fmt"
	"notification-service/config"
	"notification-service/internal/core/domain/entity"
	"notification-service/internal/core/service"
	"notification-service/utils/constant"

	"github.com/labstack/gommon/log"
)

func ConsumeMessage(client *config.RabbitMQClient, queueName string, notifService *service.NotificationService) {
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
				log.Errorf("[%s] Error decoding message: %v", queueName, err)
				d.Nack(false, false)
				continue
			}

			if err := notifService.ProcessNotification(context.Background(), notif); err != nil {
				log.Errorf("[%s] Failed to process notification: %v", queueName, err)
				d.Nack(false, true)
				continue
			}

			log.Info("Successfully send email notification/push notification")
			d.Ack(false)
		}
	}()

	log.Info("Waiting for messages. To exit press CTRL+C")
	<-forever
}

func OrderPushNotificationConsumer(client *config.RabbitMQClient, queueName, exchangeName string, notifService *service.NotificationService) {
	if client == nil {
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

	log.Infof("Consumer Push Notification Order start on queue: %s", queueName)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var order entity.OrderEntity
			if err := json.Unmarshal(d.Body, &order); err != nil {
				d.Nack(false, false)
				continue
			}

			admin := notifService.GetAdmin()
			message := fmt.Sprintf("You have a new order (ID: #%s). Please review it.", order.OrderCode)
			subject := "New Order Alert"

			notif := entity.NotificationEntity{
				Message:          message,
				NotificationType: constant.TypePush,
				ReceiverID:       &admin.ID,
				Subject:          &subject,
			}

			err := notifService.ProcessNotification(context.Background(), notif)
			if err != nil {
				d.Nack(false, true)
				continue
			}

			log.Info("Successfully send push notification")
			d.Ack(false)
		}
	}()

	log.Info("Waiting for messages. To exit press CTRL+C")
	<-forever
}

func OrderEmailNotificationConsumer(client *config.RabbitMQClient, queueName, exchangeName string, notifService *service.NotificationService) {
	if client == nil {
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

	log.Infof("Consumer Email Notification Order start on queue: %s", queueName)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var order entity.OrderEntity
			if err := json.Unmarshal(d.Body, &order); err != nil {
				d.Nack(false, false)
				continue
			}

			admin := notifService.GetAdmin()
			message := fmt.Sprintf("You have a new order (ID: #%s). Please review it.", order.OrderCode)
			subject := "New Order Alert"

			notif := entity.NotificationEntity{
				Message:          message,
				NotificationType: constant.TypeEmail,
				ReceiverEmail:    &admin.Email,
				Subject:          &subject,
			}

			err := notifService.ProcessNotification(context.Background(), notif)
			if err != nil {
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
