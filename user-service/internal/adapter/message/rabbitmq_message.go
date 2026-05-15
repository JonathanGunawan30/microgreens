package message

import (
	"encoding/json"
	"user-service/config"
	"user-service/internal/core/domain/entity"
	"user-service/utils"

	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishMessage(client *config.RabbitMQClient, userID int64, email, message, queueName, subject string) error {
	if client == nil {
		log.Errorf("[PublishMessage-0] client is nil")
		return nil
	}
	ch, err := client.GetConn().Channel()
	if err != nil {
		log.Errorf("[PublishMessage-1] Failed to open channel: %v", err)
		return err
	}
	defer ch.Close()

	queue, err := ch.QueueDeclare(queueName, true, false, false, false, nil)

	if err != nil {
		log.Errorf("[PublishMessage-3] Failed to declare a queue: %v", err)
		return err
	}

	notifType := "EMAIL"
	if queueName == utils.PUSH_NOTIF {
		notifType = "PUSH"
	}

	notification := map[string]any{
		"receiver_email":    email,
		"message":           message,
		"receiver_id":       userID,
		"subject":           subject,
		"notification_type": notifType,
	}

	body, err := json.Marshal(notification)
	if err != nil {
		log.Errorf("[PublishMessage-4] Failed to marshal JSON: %v", err)
		return err
	}

	return ch.Publish("", queue.Name, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
}

func PublishUserEvent(client *config.RabbitMQClient, user entity.UserEntity, exchangeName string) error {
	if client == nil {
		log.Errorf("[PublishUserEvent-0] client is nil")
		return nil
	}
	ch, err := client.GetConn().Channel()
	if err != nil {
		log.Errorf("[PublishUserEvent-1] Failed to open channel: %v", err)
		return err
	}

	defer ch.Close()

	err = ch.ExchangeDeclare(exchangeName, "fanout", true, false, false, false, nil)
	if err != nil {
		log.Errorf("[PublishUserEvent-2] Failed to declare exchange: %v", err)
		return err
	}

	payload := entity.UserEvent{
		UserID:  user.ID,
		Name:    user.Name,
		Email:   user.Email,
		Phone:   user.Phone,
		Address: user.Address,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Errorf("[PublishUserEvent-3] Failed to marshal: %v", err)
		return err
	}

	err = ch.Publish(exchangeName, "", false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		})

	if err != nil {
		log.Errorf("[PublishUserEvent-4] Failed to publish: %v", err)
		return err
	}

	log.Infof("[PublishUserEvent] Published user event for user ID %d", user.ID)
	return nil
}
