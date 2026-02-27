package service

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"payment-service/config"
	"payment-service/internal/adapter"
	"payment-service/internal/adapter/message"
	"payment-service/internal/adapter/repository"
	"payment-service/internal/core/domain/entity"
	msg "payment-service/utils/message"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

type PaymentServiceInterface interface {
	ProcessPayment(ctx context.Context, payment entity.PaymentEntity, accessToken string) (*entity.PaymentEntity, error)
	UpdateStatusByOrderCode(ctx context.Context, orderCode, status, accessToken string) error
	VerifyMidtransSignature(orderID string, statusCode string, grossAmount string, incomingSignature string) bool
	GetAllPayments(ctx context.Context, req entity.PaymentQueryStringRequest, accessToken string) ([]entity.PaymentEntity, int64, int64, error)
	GetPaymentDetail(ctx context.Context, paymentID int64, accessToken string, userID int64) (*entity.PaymentEntity, error)
}

type paymentService struct {
	repo       repository.PaymentRepositoryInterface
	httpClient adapter.HttpClient
	midtrans   adapter.MidtransClientInterface
	rabbitmq   *amqp.Connection
	cfg        *config.Config
}

func NewPaymentService(repo repository.PaymentRepositoryInterface, httpClient adapter.HttpClient, midtrans adapter.MidtransClientInterface, rabbitmq *amqp.Connection, cfg *config.Config) PaymentServiceInterface {
	return &paymentService{
		repo:       repo,
		httpClient: httpClient,
		midtrans:   midtrans,
		rabbitmq:   rabbitmq,
		cfg:        cfg,
	}
}

func (p *paymentService) ProcessPayment(ctx context.Context, payment entity.PaymentEntity, accessToken string) (*entity.PaymentEntity, error) {
	if strings.EqualFold(payment.PaymentMethod, "cod") {
		payment.PaymentStatus = "Success"

		if err := p.repo.CreatePayment(ctx, payment); err != nil {
			log.Errorf("[PaymentService - 1] ProcessPayment: %v", err)
			return nil, err
		}

		go func(paymentEntity entity.PaymentEntity) {
			err := message.PublishPaymentSuccess(p.rabbitmq, paymentEntity, p.cfg.PublisherName.PaymentSuccess)
			if err != nil {
				log.Errorf("[Background-RabbitMQ] Failed to publish payment success: %v", err)
			}
		}(payment)

		return &payment, nil
	}

	if strings.EqualFold(payment.PaymentMethod, "midtrans") {

		userResponse, err := p.httpClientUserService(accessToken, 0, false)
		if err != nil {
			log.Errorf("[PaymentService-3] ProcessPayment: %v", err)
			return nil, err
		}

		orderDetail, err := p.httpClientOrderService(payment.OrderID, accessToken)
		if err != nil {
			log.Errorf("[PaymentService-4] ProcessPayment: %v", err)
			return nil, err
		}

		transactionID, err := p.midtrans.CreateTransaction(orderDetail.OrderCode, int64(payment.GrossAmount), userResponse.Name, userResponse.Email)
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
			err := message.PublishPaymentSuccess(p.rabbitmq, paymentEntity, p.cfg.PublisherName.PaymentSuccess)
			if err != nil {
				log.Errorf("[Background-RabbitMQ] Failed to publish payment success: %v", err)
			}
		}(payment)

		return &payment, nil
	}

	return nil, msg.ErrInvalidPaymentMethod

}

func (p *paymentService) UpdateStatusByOrderCode(ctx context.Context, orderCode, status, accessToken string) error {
	orderDetail, err := p.httpClientOrderByCodeService(orderCode, accessToken)
	if err != nil {
		log.Errorf("[PaymentService - 1] UpdateStatusByOrderCode: %v", err)
		return err
	}

	if err := p.repo.UpdateStatusByID(ctx, orderDetail.ID, status); err != nil {
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

func (p *paymentService) GetAllPayments(ctx context.Context, req entity.PaymentQueryStringRequest, accessToken string) ([]entity.PaymentEntity, int64, int64, error) {
	payments, count, total, err := p.repo.GetAllPayment(ctx, req)
	if err != nil {
		log.Errorf("[PaymentService] GetAllPayments: %v", err)
		return nil, 0, 0, err
	}

	for key, val := range payments {
		orderDetail, err := p.httpClientOrderService(val.OrderID, accessToken)
		if err != nil {
			log.Errorf("[PaymentService] GetAllPayments: %v", err)
			return nil, 0, 0, err
		}

		payments[key].OrderCode = orderDetail.OrderCode
		payments[key].OrderShippingType = orderDetail.ShippingType

	}
	return payments, count, total, nil
}

func (p *paymentService) GetPaymentDetail(ctx context.Context, paymentID int64, accessToken string, filterUserID int64) (*entity.PaymentEntity, error) {
	detail, err := p.repo.GetPaymentDetail(ctx, paymentID, filterUserID)
	if err != nil {
		log.Errorf("[PaymentService] GetPaymentDetail: %v", err)
		return nil, err
	}

	orderDetail, err := p.httpClientOrderService(detail.OrderID, accessToken)
	if err != nil {
		log.Errorf("[PaymentService] GetPaymentDetail: %v", err)
		return nil, err
	}

	isAdmin := filterUserID == 0
	userDetail, err := p.httpClientUserService(accessToken, detail.UserID, isAdmin)
	if err != nil {
		log.Errorf("[PaymentService] GetPaymentDetail: %v", err)
		return nil, err
	}

	detail.CustomerName = userDetail.Name
	detail.CustomerEmail = userDetail.Email
	detail.CustomerAddress = userDetail.Address

	detail.OrderCode = orderDetail.OrderCode
	detail.OrderShippingType = orderDetail.ShippingType
	detail.OrderAt = orderDetail.OrderDateTime
	detail.OrderRemarks = orderDetail.Remarks

	return detail, nil
}

func (p *paymentService) httpClientOrderService(orderID int64, accessToken string) (*entity.OrderDetailHttpResponse, error) {
	baseOrderUrl := fmt.Sprintf("%s/%s", p.cfg.App.OrderServiceUrl, "auth/orders/"+strconv.FormatInt(orderID, 10))
	header := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Accept":        "application/json",
	}

	orderData, err := p.httpClient.CallURL(echo.GET, baseOrderUrl, header, nil)
	if err != nil {
		log.Errorf("[PaymentService-1] httpClientOrderService: %v", err)
		return nil, err
	}

	defer orderData.Body.Close()

	body, err := io.ReadAll(orderData.Body)
	if err != nil {
		log.Errorf("[PaymentService-2] httpClientOrderService: %v", err)
		return nil, err
	}

	var orderDetail entity.OrderHttpClientResponse
	err = json.Unmarshal(body, &orderDetail)
	if err != nil {
		log.Errorf("[PaymentService-3] httpClientOrderService: %v", err)
		return nil, err
	}

	return &orderDetail.Data, err
}

func (p *paymentService) httpClientUserService(accessToken string, targetUserID int64, isAdmin bool) (*entity.UserHttpResponse, error) {
	var baseUserUrl string

	if isAdmin {
		baseUserUrl = fmt.Sprintf("%s/%s", p.cfg.App.UserServiceUrl, "admin/customers/"+strconv.FormatInt(targetUserID, 10))
	} else {
		baseUserUrl = fmt.Sprintf("%s/%s", p.cfg.App.UserServiceUrl, "auth/profile")
	}
	header := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Accept":        "application/json",
	}

	userData, err := p.httpClient.CallURL(echo.GET, baseUserUrl, header, nil)
	if err != nil {
		log.Errorf("[PaymentService-1] httpClientUserService: %v", err)
		return nil, err
	}

	defer userData.Body.Close()

	body, err := io.ReadAll(userData.Body)
	if err != nil {
		log.Errorf("[PaymentService-2] httpClientUserService: %v", err)
		return nil, err
	}

	var userResponse entity.UserHttpClientResponse
	err = json.Unmarshal(body, &userResponse)
	if err != nil {
		log.Errorf("[PaymentService-3] httpClientUserService: %v", err)
		return nil, err
	}

	return &userResponse.Data, err
}

func (p *paymentService) httpClientOrderByCodeService(orderCode, accessToken string) (*entity.OrderDetailHttpResponse, error) {
	baseUrl := fmt.Sprintf("%s/%s", p.cfg.App.OrderServiceUrl, "auth/orders/"+orderCode+"/code")
	header := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Accept":        "application/json",
	}

	data, err := p.httpClient.CallURL(echo.GET, baseUrl, header, nil)
	if err != nil {
		log.Errorf("[PaymentService-1] httpClientOrderByCodeService: %v", err)
		return nil, err
	}

	defer data.Body.Close()

	body, err := io.ReadAll(data.Body)
	if err != nil {
		log.Errorf("[PaymentService-2] httpClientOrderByCodeService: %v", err)
		return nil, err
	}

	var orderDetail entity.OrderHttpClientResponse
	err = json.Unmarshal(body, &orderDetail)
	if err != nil {
		log.Errorf("[PaymentService-3] httpClientOrderByCodeService: %v", err)
		return nil, err
	}

	return &orderDetail.Data, err
}
