package service

import (
	"context"
	"product-service/internal/core/domain/entity"
	service2 "product-service/internal/core/service"
	repoMocks "product-service/mocks/repository"
	"product-service/utils/message"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCartService_AddToCart(t *testing.T) {
	cartRepo := new(repoMocks.CartRedisRepositoryInterface)
	prodRepo := new(repoMocks.ProductRepositoryInterface)
	svc := service2.NewCartService(cartRepo, prodRepo)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		prodRepo.On("GetProductByID", ctx, int64(1)).Return(&entity.ProductEntity{ID: 1, Stock: 10}, nil).Once()
		cartRepo.On("GetCart", ctx, int64(100)).Return([]entity.CartItem{}, nil).Once()
		cartRepo.On("AddToCart", ctx, int64(100), mock.Anything).Return(nil).Once()

		err := svc.AddToCart(ctx, 100, 1, 5)

		assert.NoError(t, err)
	})

	t.Run("quantity exceeds", func(t *testing.T) {
		prodRepo.On("GetProductByID", ctx, int64(1)).Return(&entity.ProductEntity{ID: 1, Stock: 10}, nil).Once()
		cartRepo.On("GetCart", ctx, int64(100)).Return([]entity.CartItem{{ProductID: 1, Quantity: 7}}, nil).Once()

		err := svc.AddToCart(ctx, 100, 1, 5) // 7 + 5 = 12 > 10

		assert.Equal(t, message.ErrQuantityExceeds, err)
	})
}

func TestCartService_GetCart(t *testing.T) {
	cartRepo := new(repoMocks.CartRedisRepositoryInterface)
	prodRepo := new(repoMocks.ProductRepositoryInterface)
	svc := service2.NewCartService(cartRepo, prodRepo)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		cartRepo.On("GetCart", ctx, int64(100)).Return([]entity.CartItem{{ProductID: 1, Quantity: 2}}, nil).Once()
		prodRepo.On("GetProductByID", ctx, int64(1)).Return(&entity.ProductEntity{ID: 1, Name: "Prod"}, nil).Once()

		res, err := svc.GetCart(ctx, 100)

		assert.NoError(t, err)
		assert.Len(t, res, 1)
		assert.Equal(t, "Prod", res[0].Name)
	})
}
