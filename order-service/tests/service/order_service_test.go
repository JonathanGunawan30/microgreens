package service_test

import (
	"context"
	"errors"
	"order-service/config"
	"order-service/internal/core/domain/entity"
	"order-service/internal/core/service"
	"order-service/mocks"
	"order-service/utils/message"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func generateValidToken(userID int64, secretKey string) string {
	claims := jwt.MapClaims{
		"user_id": float64(userID),
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secretKey))
	return tokenString
}

func TestGetAllCustomerOrders(t *testing.T) {
	mockRepo := new(mocks.OrderRepositoryInterface)
	cfg := &config.Config{}
	cfg.App.JwtSecretKey = "secret"

	svc := service.NewOrderService(mockRepo, nil, nil, cfg, nil, nil)
	ctx := context.Background()
	query := entity.QueryStringEntity{Page: 1, Limit: 10}
	token := generateValidToken(1, "secret")

	t.Run("success", func(t *testing.T) {
		expectedOrders := []entity.OrderEntity{{ID: 1, BuyerID: 1}}
		mockRepo.On("GetAllOrders", ctx, mock.MatchedBy(func(q entity.QueryStringEntity) bool {
			return q.BuyerID == 1
		})).Return(expectedOrders, int64(1), int64(1), nil).Once()

		orders, count, total, err := svc.GetAllCustomerOrders(ctx, query, token)

		assert.NoError(t, err)
		assert.Equal(t, expectedOrders, orders)
		assert.Equal(t, int64(1), count)
		assert.Equal(t, int64(1), total)
	})

	t.Run("invalid token", func(t *testing.T) {
		orders, count, total, err := svc.GetAllCustomerOrders(ctx, query, "invalid-token")

		assert.Error(t, err)
		assert.Nil(t, orders)
		assert.Equal(t, int64(0), count)
		assert.Equal(t, int64(0), total)
	})

	t.Run("repo error", func(t *testing.T) {
		mockRepo.On("GetAllOrders", ctx, mock.Anything).Return(nil, int64(0), int64(0), errors.New("db error")).Once()

		orders, _, _, err := svc.GetAllCustomerOrders(ctx, query, token)

		assert.Error(t, err)
		assert.Nil(t, orders)
	})
}

func TestGetAllOrders(t *testing.T) {
	mockRepo := new(mocks.OrderRepositoryInterface)
	mockUserSnapshotRepo := new(mocks.UserSnapshotRepositoryInterface)
	mockProdSnapshotRepo := new(mocks.ProductSnapshotRepositoryInterface)
	mockElasticRepo := new(mocks.ElasticRepositoryInterface)
	cfg := &config.Config{}

	svc := service.NewOrderService(mockRepo, mockUserSnapshotRepo, mockProdSnapshotRepo, cfg, nil, mockElasticRepo)

	ctx := context.Background()
	query := entity.QueryStringEntity{Page: 1, Limit: 10}

	t.Run("success from elastic", func(t *testing.T) {
		expectedOrders := []entity.OrderEntity{{ID: 1}}
		mockElasticRepo.On("SearchOrderElastic", ctx, query).Return(expectedOrders, int64(1), int64(1), nil).Once()

		orders, count, total, err := svc.GetAllOrders(ctx, query)

		assert.NoError(t, err)
		assert.Equal(t, expectedOrders, orders)
		assert.Equal(t, int64(1), count)
		assert.Equal(t, int64(1), total)
	})

	t.Run("fallback to repo on elastic error", func(t *testing.T) {
		expectedOrders := []entity.OrderEntity{{ID: 1}}
		mockElasticRepo.On("SearchOrderElastic", ctx, query).Return(nil, int64(0), int64(0), errors.New("elastic error")).Once()
		mockRepo.On("GetAllOrders", ctx, query).Return(expectedOrders, int64(1), int64(1), nil).Once()

		orders, count, total, err := svc.GetAllOrders(ctx, query)

		assert.NoError(t, err)
		assert.Equal(t, expectedOrders, orders)
		assert.Equal(t, int64(1), count)
		assert.Equal(t, int64(1), total)
	})

	t.Run("error from repo", func(t *testing.T) {
		mockElasticRepo.On("SearchOrderElastic", ctx, query).Return(nil, int64(0), int64(0), errors.New("elastic error")).Once()
		mockRepo.On("GetAllOrders", ctx, query).Return(nil, int64(0), int64(0), errors.New("db error")).Once()

		orders, count, total, err := svc.GetAllOrders(ctx, query)

		assert.Error(t, err)
		assert.Nil(t, orders)
		assert.Equal(t, int64(0), count)
		assert.Equal(t, int64(0), total)
	})
}

func TestGetOrderByID(t *testing.T) {
	mockRepo := new(mocks.OrderRepositoryInterface)
	mockElasticRepo := new(mocks.ElasticRepositoryInterface)
	svc := service.NewOrderService(mockRepo, nil, nil, nil, nil, mockElasticRepo)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		expectedOrder := &entity.OrderEntity{ID: 1, OrderItems: []entity.OrderItemEntity{{ProductImage: "img1"}}}
		mockRepo.On("GetOrderByID", ctx, int64(1)).Return(expectedOrder, nil).Once()

		order, err := svc.GetOrderByID(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, expectedOrder, order)
		assert.Equal(t, "img1", order.ProductImage)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.On("GetOrderByID", ctx, int64(1)).Return(nil, errors.New("not found")).Once()

		order, err := svc.GetOrderByID(ctx, 1)

		assert.Error(t, err)
		assert.Nil(t, order)
	})
}

func TestCreateOrder(t *testing.T) {
	mockRepo := new(mocks.OrderRepositoryInterface)
	mockUserSnapshotRepo := new(mocks.UserSnapshotRepositoryInterface)
	mockProdSnapshotRepo := new(mocks.ProductSnapshotRepositoryInterface)
	cfg := &config.Config{}
	cfg.ExchangeName.OrderEvent = "order-event"
	cfg.PublisherName.ProductUpdateStock = "update-stock"

	svc := service.NewOrderService(mockRepo, mockUserSnapshotRepo, mockProdSnapshotRepo, cfg, nil, nil)
	ctx := context.Background()

	req := entity.OrderEntity{
		BuyerID:      1,
		ShippingType: "Delivery",
		OrderItems: []entity.OrderItemEntity{
			{ProductID: 1, Quantity: 2},
		},
	}

	t.Run("success", func(t *testing.T) {
		userSnapshot := &entity.UserSnapshotEntity{
			UserID:  1,
			Name:    "User",
			Email:   "user@mail.com",
			Phone:   "123",
			Address: "Addr",
		}
		mockUserSnapshotRepo.On("GetByUserID", ctx, int64(1)).Return(userSnapshot, nil).Once()

		prodSnapshots := []entity.ProductSnapshotEntity{
			{ProductID: 1, Name: "Prod1", SalePrice: 1000, IsActive: true},
		}
		mockProdSnapshotRepo.On("GetByProductIDs", ctx, []int64{1}).Return(prodSnapshots, nil).Once()

		mockRepo.On("CreateOrder", ctx, mock.AnythingOfType("entity.OrderEntity")).Return(int64(10), nil).Once()
		mockRepo.On("GetOrderByID", mock.Anything, int64(10)).Return(&entity.OrderEntity{ID: 10}, nil).Maybe()

		orderID, err := svc.CreateOrder(ctx, req)

		assert.NoError(t, err)
		assert.Equal(t, int64(10), orderID)
	})

	t.Run("user snapshot error", func(t *testing.T) {
		mockUserSnapshotRepo.On("GetByUserID", ctx, int64(1)).Return((*entity.UserSnapshotEntity)(nil), errors.New("db error")).Once()

		_, err := svc.CreateOrder(ctx, req)

		assert.Error(t, err)
	})

	t.Run("missing phone", func(t *testing.T) {
		userSnapshot := &entity.UserSnapshotEntity{UserID: 1, Name: "User", Phone: ""}
		mockUserSnapshotRepo.On("GetByUserID", ctx, int64(1)).Return(userSnapshot, nil).Once()

		_, err := svc.CreateOrder(ctx, req)

		assert.Error(t, err)
		assert.Equal(t, message.ErrPhoneIsRequired, err)
	})

	t.Run("product snapshot error", func(t *testing.T) {
		userSnapshot := &entity.UserSnapshotEntity{UserID: 1, Name: "User", Phone: "123", Address: "Addr"}
		mockUserSnapshotRepo.On("GetByUserID", ctx, int64(1)).Return(userSnapshot, nil).Once()
		mockProdSnapshotRepo.On("GetByProductIDs", ctx, []int64{1}).Return(nil, errors.New("db error")).Once()

		_, err := svc.CreateOrder(ctx, req)

		assert.Error(t, err)
	})

	t.Run("product not found", func(t *testing.T) {
		userSnapshot := &entity.UserSnapshotEntity{UserID: 1, Name: "User", Phone: "123", Address: "Addr"}
		mockUserSnapshotRepo.On("GetByUserID", ctx, int64(1)).Return(userSnapshot, nil).Once()
		mockProdSnapshotRepo.On("GetByProductIDs", ctx, []int64{1}).Return([]entity.ProductSnapshotEntity{}, nil).Once()

		_, err := svc.CreateOrder(ctx, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "product 1 not found")
	})

	t.Run("product not active", func(t *testing.T) {
		userSnapshot := &entity.UserSnapshotEntity{UserID: 1, Name: "User", Phone: "123", Address: "Addr"}
		mockUserSnapshotRepo.On("GetByUserID", ctx, int64(1)).Return(userSnapshot, nil).Once()
		mockProdSnapshotRepo.On("GetByProductIDs", ctx, []int64{1}).Return([]entity.ProductSnapshotEntity{{ProductID: 1, IsActive: false}}, nil).Once()

		_, err := svc.CreateOrder(ctx, req)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "product 1 is not available")
	})
}

func TestUpdateStatusOrder(t *testing.T) {
	mockRepo := new(mocks.OrderRepositoryInterface)
	mockUserSnapshotRepo := new(mocks.UserSnapshotRepositoryInterface)
	cfg := &config.Config{}
	cfg.PublisherName.PublisherUpdateStatus = "update-status"
	cfg.PublisherName.EmailUpdateStatus = "email-status"

	svc := service.NewOrderService(mockRepo, mockUserSnapshotRepo, nil, cfg, nil, nil)
	ctx := context.Background()

	req := entity.OrderEntity{ID: 1, Status: "Completed"}

	t.Run("success", func(t *testing.T) {
		mockRepo.On("UpdateStatusOrder", ctx, req).Return(int64(1), "Completed", "ORD-123", nil).Once()
		mockUserSnapshotRepo.On("GetByUserID", ctx, int64(1)).Return(&entity.UserSnapshotEntity{UserID: 1, Email: "user@mail.com"}, nil).Once()

		err := svc.UpdateStatusOrder(ctx, req)

		assert.NoError(t, err)
	})

	t.Run("repo error", func(t *testing.T) {
		mockRepo.On("UpdateStatusOrder", ctx, req).Return(int64(0), "", "", errors.New("db error")).Once()

		err := svc.UpdateStatusOrder(ctx, req)

		assert.Error(t, err)
	})

	t.Run("user snapshot not found", func(t *testing.T) {
		mockRepo.On("UpdateStatusOrder", ctx, req).Return(int64(1), "Completed", "ORD-123", nil).Once()
		mockUserSnapshotRepo.On("GetByUserID", ctx, int64(1)).Return((*entity.UserSnapshotEntity)(nil), errors.New("not found")).Once()

		err := svc.UpdateStatusOrder(ctx, req)

		assert.NoError(t, err) // Service logs warning and returns nil
	})
}

func TestGetOrderByOrderCode(t *testing.T) {
	mockRepo := new(mocks.OrderRepositoryInterface)
	svc := service.NewOrderService(mockRepo, nil, nil, nil, nil, nil)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		expectedOrder := &entity.OrderEntity{ID: 1, OrderCode: "ORD-123"}
		mockRepo.On("GetOrderByOrderCode", ctx, "ORD-123").Return(expectedOrder, nil).Once()

		order, err := svc.GetOrderByOrderCode(ctx, "ORD-123")

		assert.NoError(t, err)
		assert.Equal(t, expectedOrder, order)
	})

	t.Run("error", func(t *testing.T) {
		mockRepo.On("GetOrderByOrderCode", ctx, "ORD-123").Return(nil, errors.New("not found")).Once()

		order, err := svc.GetOrderByOrderCode(ctx, "ORD-123")

		assert.Error(t, err)
		assert.Nil(t, order)
	})
}
