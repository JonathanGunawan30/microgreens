package service

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"payment-service/config"
	"payment-service/internal/adapter"
	"payment-service/internal/adapter/message"
	"payment-service/internal/adapter/repository"
	"payment-service/internal/core/domain/entity"
	msg "payment-service/utils/message"
	"strings"

	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

type PaymentServiceInterface interface {
	ProcessPayment(ctx context.Context, payment entity.PaymentEntity) (*entity.PaymentEntity, error)
	UpdateStatusByOrderCode(ctx context.Context, orderCode, status string) error
	VerifyMidtransSignature(orderID string, statusCode string, grossAmount string, incomingSignature string) bool
	GetAllPayments(ctx context.Context, req entity.PaymentQueryStringRequest) ([]entity.PaymentEntity, int64, int64, error)
	GetPaymentDetail(ctx context.Context, paymentID int64, userID int64) (*entity.PaymentEntity, error)
}

type paymentService struct {
	repo               repository.PaymentRepositoryInterface
	ordersSnapshotRepo repository.OrderSnapshotRepositoryInterface
	usersSnapshotRepo  repository.UserSnapshotRepositoryInterface
	midtrans           adapter.MidtransClientInterface
	rabbitmq           *amqp.Connection
	cfg                *config.Config
}

func NewPaymentService(repo repository.PaymentRepositoryInterface, ordersSnapshotRepo repository.OrderSnapshotRepositoryInterface, usersSnapshotRepo repository.UserSnapshotRepositoryInterface, midtrans adapter.MidtransClientInterface, rabbitmq *amqp.Connection, cfg *config.Config) PaymentServiceInterface {
	return &paymentService{
		repo:               repo,
		ordersSnapshotRepo: ordersSnapshotRepo,
		usersSnapshotRepo:  usersSnapshotRepo,
		midtrans:           midtrans,
		rabbitmq:           rabbitmq,
		cfg:                cfg,
	}
}

func (p *paymentService) ProcessPayment(ctx context.Context, payment entity.PaymentEntity) (*entity.PaymentEntity, error) {
	orderSnapshot, err := p.ordersSnapshotRepo.GetOrderByID(ctx, payment.OrderID)
	if err != nil {
		return nil, msg.ErrOrderNotFound
	}

	payment.GrossAmount = orderSnapshot.TotalAmount
	payment.OrderCode = orderSnapshot.OrderCode
	payment.OrderShippingType = orderSnapshot.ShippingType
	payment.OrderDate = orderSnapshot.OrderDate
	payment.OrderTime = orderSnapshot.OrderTime
	payment.OrderRemarks = orderSnapshot.Remarks

	if strings.EqualFold(payment.PaymentMethod, "cod") {
		payment.PaymentStatus = "Success"

		userSnapshot, err := p.usersSnapshotRepo.GetByUserID(ctx, payment.UserID)
		if err == nil {
			payment.CustomerName = userSnapshot.Name
			payment.CustomerEmail = userSnapshot.Email
			payment.CustomerAddress = userSnapshot.Address
		}

		if err := p.repo.CreatePayment(ctx, payment); err != nil {
			log.Errorf("[PaymentService - 1] ProcessPayment: %v", err)
			return nil, err
		}

		go func(paymentEntity entity.PaymentEntity) {
			err := message.PublishUpdatePaymentMethod(p.rabbitmq, paymentEntity, p.cfg.ExchangeName.PaymentEvent)
			if err != nil {
				log.Errorf("[Background-RabbitMQ] Failed to publish update payment method: %v", err)
			}
		}(payment)

		return &payment, nil
	}

	if strings.EqualFold(payment.PaymentMethod, "midtrans") {

		userSnapshot, err := p.usersSnapshotRepo.GetByUserID(ctx, payment.UserID)
		if err != nil {
			log.Errorf("[PaymentService-3] ProcessPayment: %v", err)
			return nil, err
		}

		payment.CustomerName = userSnapshot.Name
		payment.CustomerEmail = userSnapshot.Email
		payment.CustomerAddress = userSnapshot.Address

		transactionID, err := p.midtrans.CreateTransaction(orderSnapshot.OrderCode, int64(payment.GrossAmount), userSnapshot.Name, userSnapshot.Email)
		if err != nil {
			log.Errorf("[PaymentService-5] ProcessPayment: %v", err)
			return nil, err
		}

		payment.PaymentStatus = "Pending"
		payment.PaymentGatewayID = &transactionID

		if err := p.repo.CreatePayment(ctx, payment); err != nil {
			log.Errorf("[PaymentService-6] ProcessPayment: %v", err)
			return nil, err
		}

		go func(paymentEntity entity.PaymentEntity) {
			err := message.PublishUpdatePaymentMethod(p.rabbitmq, paymentEntity, p.cfg.ExchangeName.PaymentEvent)
			if err != nil {
				log.Errorf("[Background-RabbitMQ] Failed to publish update payment method: %v", err)
			}
		}(payment)

		return &payment, nil
	}

	return nil, msg.ErrInvalidPaymentMethod

}

func (p *paymentService) UpdateStatusByOrderCode(ctx context.Context, orderCode, status string) error {
	orderSnapshot, err := p.ordersSnapshotRepo.GetOrderByOrderCode(ctx, orderCode)
	if err != nil {
		log.Errorf("[PaymentService - 1] UpdateStatusByOrderCode: %v", err)
		return err
	}

	if err := p.repo.UpdateStatusByID(ctx, orderSnapshot.OrderID, status); err != nil {
		log.Errorf("[PaymentService - 2] UpdateStatusByOrderCode: %v", err)
		return err
	}

	return nil

}

func (p *paymentService) VerifyMidtransSignature(orderID string, statusCode string, grossAmount string, incomingSignature string) bool {
	serverKey := p.cfg.Midtrans.ServerKey

	payload := orderID + statusCode + grossAmount + serverKey

	hasher := sha512.New()
	hasher.Write([]byte(payload))
	expectedSignature := hex.EncodeToString(hasher.Sum(nil))

	return expectedSignature == incomingSignature
}

func (p *paymentService) GetAllPayments(ctx context.Context, req entity.PaymentQueryStringRequest) ([]entity.PaymentEntity, int64, int64, error) {
	payments, count, total, err := p.repo.GetAllPayment(ctx, req)
	if err != nil {
		log.Errorf("[PaymentService] GetAllPayments: %v", err)
		return nil, 0, 0, err
	}

	return payments, count, total, nil
}

func (p *paymentService) GetPaymentDetail(ctx context.Context, paymentID int64, filterUserID int64) (*entity.PaymentEntity, error) {
	detail, err := p.repo.GetPaymentDetail(ctx, paymentID, filterUserID)
	if err != nil {
		log.Errorf("[PaymentService] GetPaymentDetail: %v", err)
		return nil, err
	}

	return detail, nil
}
