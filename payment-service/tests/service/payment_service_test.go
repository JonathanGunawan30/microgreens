package service_test

import (
	"context"
	"errors"
	"payment-service/config"
	"payment-service/internal/core/domain/entity"
	"payment-service/internal/core/service"
	"payment-service/tests/mocks"
	msg "payment-service/utils/message"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPaymentService_ProcessPayment(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		ExchangeName: config.ExchangeName{
			PaymentEvent: "payment_exchange",
		},
	}

	t.Run("success COD", func(t *testing.T) {
		repo := new(mocks.PaymentRepositoryInterface)
		orderRepo := new(mocks.OrderSnapshotRepositoryInterface)
		userRepo := new(mocks.UserSnapshotRepositoryInterface)
		midtrans := new(mocks.MidtransClientInterface)

		paymentSvc := service.NewPaymentService(repo, orderRepo, userRepo, midtrans, nil, cfg)

		orderID := int64(1)
		orderSnapshot := entity.OrdersSnapshotEntity{
			OrderID:     orderID,
			OrderCode:   "ORDER-123",
			TotalAmount: 1000,
		}

		payment := entity.PaymentEntity{
			OrderID:       orderID,
			PaymentMethod: "cod",
			UserID:        1,
		}

		orderRepo.On("GetOrderByID", ctx, orderID).Return(&orderSnapshot, nil)
		userRepo.On("GetByUserID", ctx, int64(1)).Return(&entity.UserSnapshotEntity{Name: "User", Email: "user@example.com"}, nil)
		repo.On("CreatePayment", ctx, mock.MatchedBy(func(p entity.PaymentEntity) bool {
			return p.PaymentMethod == "cod" && p.PaymentStatus == "Success"
		})).Return(nil)

		result, err := paymentSvc.ProcessPayment(ctx, payment)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Success", result.PaymentStatus)
		repo.AssertExpectations(t)
		orderRepo.AssertExpectations(t)
		userRepo.AssertExpectations(t)
	})

	t.Run("success Midtrans", func(t *testing.T) {
		repo := new(mocks.PaymentRepositoryInterface)
		orderRepo := new(mocks.OrderSnapshotRepositoryInterface)
		userRepo := new(mocks.UserSnapshotRepositoryInterface)
		midtrans := new(mocks.MidtransClientInterface)

		paymentSvc := service.NewPaymentService(repo, orderRepo, userRepo, midtrans, nil, cfg)

		orderID := int64(1)
		orderSnapshot := entity.OrdersSnapshotEntity{
			OrderID:     orderID,
			OrderCode:   "ORDER-123",
			TotalAmount: 1000,
		}

		payment := entity.PaymentEntity{
			OrderID:       orderID,
			PaymentMethod: "midtrans",
			UserID:        1,
		}

		userSnapshot := entity.UserSnapshotEntity{UserID: 1, Name: "User", Email: "user@example.com"}

		orderRepo.On("GetOrderByID", ctx, orderID).Return(&orderSnapshot, nil)
		userRepo.On("GetByUserID", ctx, int64(1)).Return(&userSnapshot, nil)
		midtrans.On("CreateTransaction", "ORDER-123", int64(1000), "User", "user@example.com").Return("TX-123", nil)
		repo.On("CreatePayment", ctx, mock.MatchedBy(func(p entity.PaymentEntity) bool {
			return p.PaymentMethod == "midtrans" && p.PaymentStatus == "Pending" && *p.PaymentGatewayID == "TX-123"
		})).Return(nil)

		result, err := paymentSvc.ProcessPayment(ctx, payment)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Pending", result.PaymentStatus)
		repo.AssertExpectations(t)
		orderRepo.AssertExpectations(t)
		userRepo.AssertExpectations(t)
		midtrans.AssertExpectations(t)
	})

	t.Run("order not found", func(t *testing.T) {
		orderRepo := new(mocks.OrderSnapshotRepositoryInterface)
		paymentSvc := service.NewPaymentService(nil, orderRepo, nil, nil, nil, cfg)

		orderRepo.On("GetOrderByID", ctx, int64(99)).Return(nil, errors.New("not found"))

		payment := entity.PaymentEntity{OrderID: 99}
		result, err := paymentSvc.ProcessPayment(ctx, payment)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, msg.ErrOrderNotFound, err)
	})

	t.Run("invalid payment method", func(t *testing.T) {
		orderRepo := new(mocks.OrderSnapshotRepositoryInterface)
		paymentSvc := service.NewPaymentService(nil, orderRepo, nil, nil, nil, cfg)

		orderRepo.On("GetOrderByID", ctx, int64(1)).Return(&entity.OrdersSnapshotEntity{}, nil)

		payment := entity.PaymentEntity{OrderID: 1, PaymentMethod: "invalid"}
		result, err := paymentSvc.ProcessPayment(ctx, payment)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, msg.ErrInvalidPaymentMethod, err)
	})
}

func TestPaymentService_UpdateStatusByOrderCode(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{}

	t.Run("success", func(t *testing.T) {
		repo := new(mocks.PaymentRepositoryInterface)
		orderRepo := new(mocks.OrderSnapshotRepositoryInterface)
		paymentSvc := service.NewPaymentService(repo, orderRepo, nil, nil, nil, cfg)

		orderRepo.On("GetOrderByOrderCode", ctx, "ORDER-123").Return(&entity.OrdersSnapshotEntity{OrderID: 1}, nil)
		repo.On("UpdateStatusByID", ctx, int64(1), "Success").Return(nil)

		err := paymentSvc.UpdateStatusByOrderCode(ctx, "ORDER-123", "Success")

		assert.NoError(t, err)
	})

	t.Run("order not found", func(t *testing.T) {
		orderRepo := new(mocks.OrderSnapshotRepositoryInterface)
		paymentSvc := service.NewPaymentService(nil, orderRepo, nil, nil, nil, cfg)

		orderRepo.On("GetOrderByOrderCode", ctx, "ORDER-X").Return(nil, errors.New("not found"))

		err := paymentSvc.UpdateStatusByOrderCode(ctx, "ORDER-X", "Success")

		assert.Error(t, err)
	})
}

func TestPaymentService_VerifyMidtransSignature(t *testing.T) {
	cfg := &config.Config{
		Midtrans: config.Midtrans{
			ServerKey: "secret",
		},
	}
	paymentSvc := service.NewPaymentService(nil, nil, nil, nil, nil, cfg)

	// Sample data
	orderID := "ORDER-123"
	statusCode := "200"
	grossAmount := "1000.00"
	// payload: ORDER-1232001000.00secret
	// SHA512 of payload
	// echo -n "ORDER-1232001000.00secret" | sha512sum
	
	isValid := paymentSvc.VerifyMidtransSignature(orderID, statusCode, grossAmount, "wrong")
	assert.False(t, isValid)
}

func TestPaymentService_GetAllPayments(t *testing.T) {
	ctx := context.Background()
	repo := new(mocks.PaymentRepositoryInterface)
	paymentSvc := service.NewPaymentService(repo, nil, nil, nil, nil, nil)

	req := entity.PaymentQueryStringRequest{}
	repo.On("GetAllPayment", ctx, req).Return([]entity.PaymentEntity{{ID: 1}}, int64(1), int64(1), nil)

	res, count, total, err := paymentSvc.GetAllPayments(ctx, req)

	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, int64(1), count)
	assert.Equal(t, int64(1), total)
}

func TestPaymentService_GetPaymentDetail(t *testing.T) {
	ctx := context.Background()
	repo := new(mocks.PaymentRepositoryInterface)
	paymentSvc := service.NewPaymentService(repo, nil, nil, nil, nil, nil)

	repo.On("GetPaymentDetail", ctx, int64(1), int64(1)).Return(&entity.PaymentEntity{ID: 1}, nil)

	res, err := paymentSvc.GetPaymentDetail(ctx, 1, 1)

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, int64(1), res.ID)
}
