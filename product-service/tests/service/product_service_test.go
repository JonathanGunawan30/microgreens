package service

import (
	"context"
	"errors"
	"product-service/internal/core/domain/entity"
	service2 "product-service/internal/core/service"
	"product-service/tests/mocks/repository"
	"product-service/utils/message"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestProductService_GetAllProducts(t *testing.T) {
	repo := new(mocks.ProductRepositoryInterface)
	repoCat := new(mocks.CategoryRepositoryInterface)
	svc := service2.NewProductService(repo, nil, "test-exchange", repoCat)
	ctx := context.Background()
	query := entity.QueryStringProduct{}

	repo.On("GetAllProducts", ctx, query).Return([]entity.ProductEntity{{ID: 1}}, int64(1), int64(1), nil)

	res, count, total, err := svc.GetAllProducts(ctx, query)

	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, int64(1), count)
	assert.Equal(t, int64(1), total)
}

func TestProductService_GetProductByID(t *testing.T) {
	repo := new(mocks.ProductRepositoryInterface)
	repoCat := new(mocks.CategoryRepositoryInterface)
	svc := service2.NewProductService(repo, nil, "test-exchange", repoCat)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo.On("GetProductByID", ctx, int64(1)).Return(&entity.ProductEntity{ID: 1, CategorySlug: "test"}, nil).Once()
		repoCat.On("GetCategoryBySlug", ctx, "test").Return(&entity.CategoryEntity{Name: "Test Cat"}, nil).Once()

		res, err := svc.GetProductByID(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, "Test Cat", res.CategoryName)
	})

	t.Run("not found", func(t *testing.T) {
		repo.On("GetProductByID", ctx, int64(2)).Return(nil, message.ErrProductNotFound).Once()

		res, err := svc.GetProductByID(ctx, 2)

		assert.Error(t, err)
		assert.Nil(t, res)
	})
}

func TestProductService_CreateProduct(t *testing.T) {
	repo := new(mocks.ProductRepositoryInterface)
	repoCat := new(mocks.CategoryRepositoryInterface)
	svc := service2.NewProductService(repo, nil, "test-exchange", repoCat)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		product := entity.ProductEntity{Name: "Prod", CategorySlug: "cat"}
		repoCat.On("GetCategoryBySlug", ctx, "cat").Return(&entity.CategoryEntity{Name: "Cat"}, nil).Once()
		repo.On("CreateProduct", ctx, mock.Anything).Return(&entity.ProductEntity{ID: 1, Name: "Prod"}, nil).Once()
		repo.On("GetVariantsByParentID", mock.Anything, int64(1)).Return([]entity.ProductEntity{}, nil).Maybe()

		err := svc.CreateProduct(ctx, product)

		assert.NoError(t, err)
	})
}

func TestProductService_UpdateProduct(t *testing.T) {
	repo := new(mocks.ProductRepositoryInterface)
	repoCat := new(mocks.CategoryRepositoryInterface)
	svc := service2.NewProductService(repo, nil, "test-exchange", repoCat)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		product := entity.ProductEntity{ID: 1, Name: "Updated"}
		repo.On("UpdateProduct", ctx, product).Return(&entity.ProductEntity{ID: 1, Name: "Updated"}, nil).Once()
		repo.On("GetVariantsByParentID", mock.Anything, int64(1)).Return([]entity.ProductEntity{}, nil).Maybe()

		err := svc.UpdateProduct(ctx, product)

		assert.NoError(t, err)
	})
}

func TestProductService_DeleteProductByID(t *testing.T) {
	repo := new(mocks.ProductRepositoryInterface)
	repoCat := new(mocks.CategoryRepositoryInterface)
	svc := service2.NewProductService(repo, nil, "test-exchange", repoCat)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo.On("GetProductByID", ctx, int64(1)).Return(&entity.ProductEntity{ID: 1, CategorySlug: "test"}, nil).Once()
		repoCat.On("GetCategoryBySlug", ctx, "test").Return(&entity.CategoryEntity{Name: "Test Cat"}, nil).Once()
		repo.On("DeleteProductByID", ctx, int64(1)).Return(nil).Once()

		err := svc.DeleteProductByID(ctx, 1)

		assert.NoError(t, err)
	})

	t.Run("error delete", func(t *testing.T) {
		repo.On("GetProductByID", ctx, int64(1)).Return(&entity.ProductEntity{ID: 1, CategorySlug: "test"}, nil).Once()
		repoCat.On("GetCategoryBySlug", ctx, "test").Return(&entity.CategoryEntity{Name: "Test Cat"}, nil).Once()
		repo.On("DeleteProductByID", ctx, int64(1)).Return(errors.New("error")).Once()

		err := svc.DeleteProductByID(ctx, 1)

		assert.Error(t, err)
	})
}
