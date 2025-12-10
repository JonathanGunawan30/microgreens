package config

import (
	"fmt"

	"github.com/labstack/gommon/log"
	"github.com/rabbitmq/amqp091-go"
)

func (cfg Config) NewRabbitMQClient() (*amqp091.Connection, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/", cfg.RabbitMQ.Username, cfg.RabbitMQ.Password, cfg.RabbitMQ.Host, cfg.RabbitMQ.Port)
	conn, err := amqp091.Dial(url)
	if err != nil {
		log.Printf("[NewRabbitMQClient] Failed to connect to RabbitMQ: %v", err)
		return nil, err
	}
	return conn, nil
}
