package config

import (
	"fmt"
	"sync"
	"time"

	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQClient struct {
	conn *amqp.Connection
	url  string
	mu   sync.Mutex
}

func (cfg Config) NewRabbitMQClient() (*RabbitMQClient, error) {
	url := fmt.Sprintf("amqp://%s:%s@%s:%s/", cfg.RabbitMQ.Username, cfg.RabbitMQ.Password, cfg.RabbitMQ.Host, cfg.RabbitMQ.Port)
	conn, err := amqp.Dial(url)
	if err != nil {
		log.Errorf("[NewRabbitMQClient] Failed to connect to RabbitMQ: %v", err)
		return nil, err
	}

	client := &RabbitMQClient{conn: conn, url: url}
	go client.watchConnection()
	return client, nil
}

func (r *RabbitMQClient) watchConnection() {
	for {
		reason, ok := <-r.conn.NotifyClose(make(chan *amqp.Error))
		if !ok {
			break
		}
		log.Warnf("[RabbitMQ] Connection lost: %v, reconnecting...", reason)

		for {
			time.Sleep(5 * time.Second)
			conn, err := amqp.Dial(r.url)
			if err != nil {
				log.Warnf("[RabbitMQ] Reconnect failed: %v, retrying...", err)
				continue
			}
			r.mu.Lock()
			r.conn = conn
			r.mu.Unlock()
			log.Info("[RabbitMQ] Reconnected successfully")
			go r.watchConnection()
			return
		}
	}
}

func (r *RabbitMQClient) GetConn() *amqp.Connection {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.conn
}

func (r *RabbitMQClient) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conn.Close()
}