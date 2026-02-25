package message

import (
	"context"
	"encoding/json"
	"fmt"
	"order-service/internal/core/domain/entity"
	"time"

	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishUpdateStock(conn *amqp.Connection, productID, quantity int64, queueName string) {
	ch, err := conn.Channel()
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

func PublishOrderToQueue(conn *amqp.Connection, order entity.OrderEntity, queueName string) error {
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %v", err)
	}

	defer ch.Close()

	_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	body, err := json.Marshal(order)
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

	log.Infof("Successfully publish order: %v", order)
	return nil
}

func PublishEmailUpdateStatus(conn *amqp.Connection, email string, message string, queueName string) error {
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %v", err)
	}

	defer ch.Close()

	_, err = ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	notification := map[string]string{
		"email":   email,
		"message": message,
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

func PublishUpdateStatus(conn *amqp.Connection, orderID int64, status string, queueName string) error {
	ch, err := conn.Channel()
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
