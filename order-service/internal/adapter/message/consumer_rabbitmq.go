package message

import (
	"context"
	"encoding/json"
	"fmt"
	"order-service/internal/adapter/handler/response"
	"order-service/internal/core/domain/entity"
	"strconv"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types/enums/refresh"
	"github.com/labstack/gommon/log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func StartOrderConsumer(conn *amqp.Connection, queueName string, es *elasticsearch.TypedClient) {
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

	log.Infof("Consumer Order start on queue: %s", queueName)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var order entity.OrderEntity
			json.Unmarshal(d.Body, &order)
			if err != nil {
				log.Errorf("Error decoding message: %v", err)
				d.Nack(false, false)
				continue
			}

			res, err := es.Index("orders").
				Id(fmt.Sprintf("%d", order.ID)).
				Refresh(refresh.True).
				Request(order).
				Do(context.Background())

			if err != nil {
				log.Errorf("Error indexing to Elasticsearch: %v", err)
				d.Nack(false, true)
				continue
			}

			log.Infof("Successfully indexed order ID: %d, Response: %v", order.ID, res.Result)
			d.Ack(false)
		}
	}()

	log.Info("Waiting for messages. To exit press CTRL+C")
	<-forever
}

func ConsumePaymentSuccess(conn *amqp.Connection, queueName string, es *elasticsearch.TypedClient) {
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

	log.Infof("Consumer Payment Success started on queue: %s", queueName)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var payment response.PaymentMessage
			if err := json.Unmarshal(d.Body, &payment); err != nil {
				log.Errorf("[ConsumePayment] Error decoding message: %v", err)
				d.Nack(false, false)
				continue
			}

			updateDoc := map[string]interface{}{
				"payment_method": payment.PaymentMethod,
			}

			orderIDStr := strconv.FormatInt(payment.OrderID, 10)

			_, err := es.Update("orders", orderIDStr).
				Doc(updateDoc).
				Do(context.Background())

			if err != nil {
				log.Errorf("[ConsumePayment-2] Failed to update ES: %v", err)
				d.Nack(false, true)
				continue
			}

			if err := d.Ack(false); err != nil {
				log.Errorf("[ConsumePayment-3] Failed to Ack message: %v", err)
			} else {
				log.Infof("Successfully updated PaymentMethod for Order %d", payment.OrderID)
			}

		}
	}()

	log.Info("Waiting for messages. To exit press CTRL+C")
	<-forever
}

func ConsumeUpdateStatus(conn *amqp.Connection, queueName string, es *elasticsearch.TypedClient) {
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

	log.Infof("Consumer Update Status started on queue: %s", queueName)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var msg response.UpdateStatusMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Errorf("[ConsumeUpdateStatus] Error decoding message: %v", err)
				d.Nack(false, false)
				continue
			}

			updateDoc := map[string]interface{}{
				"status": msg.Status,
			}

			orderIDStr := strconv.FormatInt(msg.OrderID, 10)

			_, err := es.Update("orders", orderIDStr).
				Doc(updateDoc).
				Do(context.Background())

			if err != nil {
				log.Errorf("[ConsumeUpdateStatus-2] Failed to update ES: %v", err)
				d.Nack(false, true)
				continue
			}

			if err := d.Ack(false); err != nil {
				log.Errorf("[ConsumeUpdateStatus-3] Failed to Ack message: %v", err)
			} else {
				log.Infof("Successfully updated Status to '%s' for Order ID %d in Elasticsearch", msg.Status, msg.OrderID)
			}

		}
	}()

	log.Info("Waiting for messages. To exit press CTRL+C")
	<-forever
}
