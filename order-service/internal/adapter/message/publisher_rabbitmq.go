package message

import (
	"context"
	"encoding/json"
	"fmt"
	"order-service/config"
	"order-service/internal/core/domain/entity"
	"order-service/utils/constant"
	"time"

	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishUpdateStock(client *config.RabbitMQClient, productID, quantity int64, queueName string) {
	if client == nil {
		log.Errorf("RabbitMQ client is nil")
		return
	}
	ch, err := client.GetConn().Channel()
	if err != nil {
		log.Errorf("failed to open a channel: %v", err)
		return
	}

	defer ch.Close()

	_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)

	if err != nil {
		log.Errorf("failed to declare a queue: %v", err)
		return
	}

	order := entity.PublishOrderItemEntity{
		ProductID: productID,
		Quantity:  quantity,
	}

	body, err := json.Marshal(order)
	if err != nil {
		log.Errorf("failed to marshal msg: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(ctx, "", queueName, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})

	if err != nil {
		log.Errorf("failed to publish message: %v", err)
		return
	}

	log.Infof("Successfully publish order: %d", order)
}

func PublishOrderEvent(client *config.RabbitMQClient, order entity.OrderEntity, eventName string) error {
	if client == nil {
		return fmt.Errorf("rabbitmq client is nil")
	}
	ch, err := client.GetConn().Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %v", err)
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(eventName, "fanout", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	body, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal msg: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(ctx, eventName, "", false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Infof("Published order event: %v", order.ID)
	return nil
}

func PublishEmailUpdateStatus(client *config.RabbitMQClient, email, message, queueName string, userID int64) error {
	if client == nil {
		return fmt.Errorf("rabbitmq client is nil")
	}
	ch, err := client.GetConn().Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %v", err)
	}

	defer ch.Close()

	_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	notifType := "EMAIL"
	if queueName == constant.PUSH_NOTIF {
		notifType = constant.PUSH_NOTIF
	}

	notification := map[string]any{
		"receiver_email":    email,
		"message":           message,
		"receiver_id":       userID,
		"subject":           "Status Order Updated",
		"notification_type": notifType,
	}

	body, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal msg: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(ctx, "", queueName, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Infof("Successfully publish email update status: %v", notification)
	return nil
}

func PublishUpdateStatus(client *config.RabbitMQClient, orderID int64, status string, queueName string) error {
	if client == nil {
		return fmt.Errorf("rabbitmq client is nil")
	}
	ch, err := client.GetConn().Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %v", err)
	}

	defer ch.Close()

	_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	data := map[string]any{
		"status":   status,
		"order_id": orderID,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal msg: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(ctx, "", queueName, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Infof("Successfully update status: %v", data)
	return nil
}

func PublishSendPushNotifUpdateStatus(client *config.RabbitMQClient, message, queueName string, userID int64) error {
	if client == nil {
		return fmt.Errorf("rabbitmq client is nil")
	}
	ch, err := client.GetConn().Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %v", err)
	}

	defer ch.Close()

	_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	notifType := constant.PUSH_NOTIF

	notification := map[string]any{
		"receiver_email":    "",
		"message":           message,
		"receiver_id":       userID,
		"subject":           "Status Order Updated",
		"notification_type": notifType,
	}

	body, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal msg: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(ctx, "", queueName, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Infof("Successfully push notif update status: %v", notification)
	return nil
}
