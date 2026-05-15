package message

import (
	"context"
	"encoding/json"
	"fmt"
	"payment-service/config"
	"payment-service/internal/core/domain/entity"
	"time"

	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishUpdatePaymentMethod(client *config.RabbitMQClient, payment entity.PaymentEntity, exchangeName string) error {
	if client == nil {
		log.Errorf("[PublishUpdatePaymentMethod] RabbitMQ client is nil, skipping publish")
		return nil
	}
	ch, err := client.GetConn().Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %v", err)
	}

	defer ch.Close()

	err = ch.ExchangeDeclare(exchangeName, "fanout", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	data := map[string]any{
		"order_id":       payment.OrderID,
		"payment_method": payment.PaymentMethod,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal msg: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(ctx, exchangeName, "", false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	log.Infof("Successfully publish update payment: %v, to exchange: %s", payment, exchangeName)
	return nil
}
