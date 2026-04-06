package message

import (
	"context"
	"encoding/json"
	"fmt"
	"order-service/internal/adapter/handler/response"
	"order-service/internal/adapter/repository"
	"order-service/internal/core/domain/entity"
	"strconv"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/refresh"
	"github.com/labstack/gommon/log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func StartOrderConsumer(conn *amqp.Connection, queueName, eventName string, es *elasticsearch.TypedClient) {
	if conn == nil {
		log.Errorf("RabbitMQ connection is nil")
		return
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}

	defer ch.Close()

	err = ch.ExchangeDeclare(eventName, "fanout", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare exchange: %v", err)
	}

	q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	err = ch.QueueBind(q.Name, "", eventName, false, nil)
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

	log.Infof("Consumer Order start on queue: %s", queueName)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var order entity.OrderEntity
			if err := json.Unmarshal(d.Body, &order); err != nil {
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

func ConsumeUpdatePaymentMethodES(conn *amqp.Connection, queueName, exchangeName string, es *elasticsearch.TypedClient) {
	if conn == nil {
		log.Errorf("RabbitMQ connection is nil")
		return
	}
	ch, err := conn.Channel()
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

	log.Infof("Consumer Update Payment Method to ES started on queue: %s", queueName)

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
				log.Infof("Successfully updated PaymentMethod to ES for Order %d", payment.OrderID)
			}

		}
	}()

	log.Info("Waiting for messages. To exit press CTRL+C")
	<-forever
}

func ConsumeUpdatePaymentMethodDB(conn *amqp.Connection, queueName, exchangeName string, orderRepo repository.OrderRepositoryInterface) {
	if conn == nil {
		log.Errorf("RabbitMQ connection is nil")
		return
	}
	ch, err := conn.Channel()
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

	log.Infof("Consumer Update Payment Method to DB start on queue: %s", queueName)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var payment response.PaymentMessage
			if err := json.Unmarshal(d.Body, &payment); err != nil {
				log.Errorf("Error decoding message: %v", err)
				d.Nack(false, false)
				continue
			}

			err := orderRepo.UpdatePaymentMethod(context.Background(), payment.OrderID, payment.PaymentMethod)
			if err != nil {
				log.Errorf("Failed to update order payment method DB: %v", err)
				d.Nack(false, true)
				continue
			}

			log.Infof("Successfully updated PaymentMethod to DB for Order %d", payment.OrderID)
			d.Ack(false)
		}
	}()

	log.Info("Waiting for messages. To exit press CTRL+C")
	<-forever
}

func ConsumeUpdateStatus(conn *amqp.Connection, queueName string, es *elasticsearch.TypedClient) {
	if conn == nil {
		log.Errorf("RabbitMQ connection is nil")
		return
	}
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

func ConsumeUserSnapshot(conn *amqp.Connection, queueName, exchangeName string, userSnapshotRepo repository.UserSnapshotRepositoryInterface) {
	if conn == nil {
		log.Errorf("RabbitMQ connection is nil")
		return
	}
	ch, err := conn.Channel()
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

	log.Infof("Consumer User Snapshot start on queue: %s", queueName)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var userSnapshot entity.UserSnapshotEntity
			if err := json.Unmarshal(d.Body, &userSnapshot); err != nil {
				log.Errorf("[ConsumeUserSnapshot] Error decoding message: %v", err)
				d.Nack(false, false)
				continue
			}

			if err := userSnapshotRepo.Upsert(context.Background(), userSnapshot); err != nil {
				log.Errorf("[ConsumeUserSnapshot] Error Upsert to User Snapshot: %v", err)
				d.Nack(false, true)
				continue
			}

			log.Infof("Successfully upsert user snapshot with user ID: %d", userSnapshot.UserID)
			d.Ack(false)
		}
	}()

	log.Info("Waiting for messages. To exit press CTRL+C")
	<-forever
}

func ConsumeProductSnapshot(conn *amqp.Connection, queueName, exchangeName string, repo repository.ProductSnapshotRepositoryInterface) {
	if conn == nil {
		log.Errorf("RabbitMQ connection is nil")
		return
	}
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open channel: %v", err)
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(exchangeName, "fanout", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare exchange: %v", err)
	}

	q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
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

	log.Infof("Consumer Product Snapshot started on queue: %s", queueName)

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var event entity.ProductEvent
			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Errorf("[ConsumeProductSnapshot] Error decoding message: %v", err)
				d.Nack(false, false)
				continue
			}

			ctx := context.Background()
			var processErr error

			switch event.Action {
			case entity.ActionInsert:
				if event.Data == nil {
					log.Errorf("[ConsumeProductSnapshot] Data is nil for action: %s", event.Action)
					d.Nack(false, false)
					continue
				}

				processErr = repo.Upsert(ctx, entity.ProductSnapshotEntity{
					ProductID: event.Data.ID,
					Name:      event.Data.Name,
					Image:     event.Data.Image,
					SalePrice: event.Data.SalePrice,
					Unit:      event.Data.Unit,
					Weight:    event.Data.Weight,
					IsActive:  event.Data.Status == "active",
				})

			case entity.ActionDelete:
				if event.ID == 0 {
					log.Errorf("[ConsumeProductSnapshot] ID is 0 for delete action")
					d.Nack(false, false)
					continue
				}
				processErr = repo.SetInactive(ctx, event.ID)

			default:
				log.Warnf("[ConsumeProductSnapshot] Unknown action: %s", event.Action)
				d.Ack(false)
				continue
			}

			if processErr != nil {
				log.Errorf("[ConsumeProductSnapshot] Failed to process action %s: %v", event.Action, processErr)
				d.Nack(false, true)
				continue
			}

			log.Infof("[ConsumeProductSnapshot] Successfully processed action: %s, product ID: %d", event.Action, event.ID)
			d.Ack(false)
		}
	}()

	log.Info("Waiting for messages. To exit press CTRL+C")
	<-forever
}
