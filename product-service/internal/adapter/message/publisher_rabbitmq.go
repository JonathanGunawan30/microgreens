package message

import (
	"context"
	"encoding/json"
	"fmt"
	"product-service/internal/core/domain/entity"
	"time"

	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishProductToQueue(conn *amqp.Connection, msg entity.EsSyncMessage, queueName string) error {
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %w", err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal msg: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(
		ctx,
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Infof("Successfully published msg Action: %s, ID: %d to queue", msg.Action, msg.ID)
	return nil
}

func PublishProductWithRetry(conn *amqp.Connection, msg entity.EsSyncMessage, queueName string) {
	const (
		maxRetries    = 3
		retryInterval = 2 * time.Second
	)

	for i := 0; i < maxRetries; i++ {
		err := PublishProductToQueue(conn, msg, queueName)
		if err == nil {
			// success
			return
		}

		log.Errorf("[Attempt %d/%d] Failed to publish message (Action: %s, ID: %d): %v",
			i+1, maxRetries, msg.Action, msg.ID, err)

		// cooldown
		time.Sleep(retryInterval)
	}

	log.Errorf("[CRITICAL] GAVE UP publishing message Action: %s, ID: %d. Data might be out of sync.",
		msg.Action, msg.ID)
}
