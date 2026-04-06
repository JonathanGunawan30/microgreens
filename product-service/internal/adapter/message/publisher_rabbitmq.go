package message

import (
	"encoding/json"
	"product-service/internal/core/domain/entity"

	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishProductEvent(conn *amqp.Connection, product entity.ProductEntity, exchangeName string, action entity.ActionType) error {
	if conn == nil {
		return nil
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Errorf("Failed to open a channel: %v", err)
		return err
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(exchangeName, "fanout", true, false, false, false, nil)
	if err != nil {
		log.Errorf("Failed to declare exchange: %v", err)
		return err
	}

	payload := entity.ProductEvent{
		Action: action,
		Data:   &product,
		ID:     product.ID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Errorf("Failed to marshal: %v", err)
		return err
	}

	err = ch.Publish(exchangeName, "", false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		})

	if err != nil {
		log.Errorf("Failed to publish: %v", err)
		return err
	}

	log.Infof("Published product event action: %s, ID: %d", action, product.ID)
	return nil
}
