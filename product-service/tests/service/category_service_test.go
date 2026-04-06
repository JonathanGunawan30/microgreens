package service

import (
	"context"
	"errors"
	"product-service/internal/core/domain/entity"
	service2 "product-service/internal/core/service"
	repoMocks "product-service/mocks/repository"
	"product-service/utils/message"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCategoryService_GetAllCategories(t *testing.T) {
	repo := new(repoMocks.CategoryRepositoryInterface)
	svc := service2.NewCategoryService(repo)
	ctx := context.Background()
	query := entity.QueryStringCategory{}

	expectedCategories := []entity.CategoryEntity{{ID: 1, Name: "Test"}}
	repo.On("GetAllCategories", ctx, query).Return(expectedCategories, int64(1), int64(1), nil)

	res, count, total, err := svc.GetAllCategories(ctx, query)

	assert.NoError(t, err)
	assert.Equal(t, expectedCategories, res)
	assert.Equal(t, int64(1), count)
	assert.Equal(t, int64(1), total)
}

func TestCategoryService_GetCategoryByID(t *testing.T) {
	repo := new(repoMocks.CategoryRepositoryInterface)
	svc := service2.NewCategoryService(repo)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		expected := &entity.CategoryEntity{ID: 1, Name: "Test"}
		repo.On("GetCategoryByID", ctx, int64(1)).Return(expected, nil).Once()

		res, err := svc.GetCategoryByID(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, expected, res)
	})

	t.Run("error", func(t *testing.T) {
		repo.On("GetCategoryByID", ctx, int64(2)).Return(nil, errors.New("error")).Once()

		res, err := svc.GetCategoryByID(ctx, 2)

		assert.Error(t, err)
		assert.Nil(t, res)
	})
}

func TestCategoryService_GetCategoryBySlug(t *testing.T) {
	repo := new(repoMocks.CategoryRepositoryInterface)
	svc := service2.NewCategoryService(repo)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		expected := &entity.CategoryEntity{ID: 1, Name: "Test", Slug: "test"}
		repo.On("GetCategoryBySlug", ctx, "test").Return(expected, nil).Once()

		res, err := svc.GetCategoryBySlug(ctx, "test")

		assert.NoError(t, err)
		assert.Equal(t, expected, res)
	})
}

func TestCategoryService_CreateCategory(t *testing.T) {
	repo := new(repoMocks.CategoryRepositoryInterface)
	svc := service2.NewCategoryService(repo)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		cat := entity.CategoryEntity{Name: "Test Category"}
		repo.On("CheckSlugExists", ctx, "test-category").Return(false, nil).Once()
		repo.On("CreateCategory", ctx, mock.MatchedBy(func(c entity.CategoryEntity) bool {
			return c.Name == "Test Category" && c.Slug == "test-category"
		})).Return(nil).Once()

		err := svc.CreateCategory(ctx, cat)

		assert.NoError(t, err)
	})

	t.Run("already exists", func(t *testing.T) {
		cat := entity.CategoryEntity{Name: "Test Category"}
		repo.On("CheckSlugExists", ctx, "test-category").Return(true, nil).Once()

		err := svc.CreateCategory(ctx, cat)

		assert.Equal(t, message.ErrCategoryAlreadyExists, err)
	})

	t.Run("check slug error", func(t *testing.T) {
		cat := entity.CategoryEntity{Name: "Test Category"}
		repo.On("CheckSlugExists", ctx, "test-category").Return(false, errors.New("error")).Once()

		err := svc.CreateCategory(ctx, cat)

		assert.Error(t, err)
	})
}

func TestCategoryService_UpdateCategory(t *testing.T) {
	repo := new(repoMocks.CategoryRepositoryInterface)
	svc := service2.NewCategoryService(repo)
	ctx := context.Background()

	t.Run("success same name", func(t *testing.T) {
		cat := entity.CategoryEntity{ID: 1, Name: "Test"}
		repo.On("GetCategoryByID", ctx, int64(1)).Return(&entity.CategoryEntity{ID: 1, Name: "Test", Slug: "test"}, nil).Once()
		repo.On("UpdateCategory", ctx, mock.MatchedBy(func(c entity.CategoryEntity) bool {
			return c.ID == 1 && c.Name == "Test" && c.Slug == "test"
		})).Return(nil).Once()

		err := svc.UpdateCategory(ctx, cat)

		assert.NoError(t, err)
	})

	t.Run("success different name", func(t *testing.T) {
		cat := entity.CategoryEntity{ID: 1, Name: "New Test"}
		repo.On("GetCategoryByID", ctx, int64(1)).Return(&entity.CategoryEntity{ID: 1, Name: "Old Test", Slug: "old-test"}, nil).Once()
		repo.On("CheckSlugExists", ctx, "new-test").Return(false, nil).Once()
		repo.On("UpdateCategory", ctx, mock.MatchedBy(func(c entity.CategoryEntity) bool {
			return c.ID == 1 && c.Name == "New Test" && c.Slug == "new-test"
		})).Return(nil).Once()

		err := svc.UpdateCategory(ctx, cat)

		assert.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		cat := entity.CategoryEntity{ID: 1, Name: "Test"}
		repo.On("GetCategoryByID", ctx, int64(1)).Return(nil, nil).Once()

		err := svc.UpdateCategory(ctx, cat)

		assert.Equal(t, message.ErrCategoryNotFound, err)
	})

	t.Run("already exists", func(t *testing.T) {
		cat := entity.CategoryEntity{ID: 1, Name: "New Test"}
		repo.On("GetCategoryByID", ctx, int64(1)).Return(&entity.CategoryEntity{ID: 1, Name: "Old Test", Slug: "old-test"}, nil).Once()
		repo.On("CheckSlugExists", ctx, "new-test").Return(true, nil).Once()

		err := svc.UpdateCategory(ctx, cat)

		assert.Equal(t, message.ErrCategoryAlreadyExists, err)
	})
}

func TestCategoryService_DeleteCategoryByID(t *testing.T) {
	repo := new(repoMocks.CategoryRepositoryInterface)
	svc := service2.NewCategoryService(repo)
	ctx := context.Background()

	repo.On("DeleteCategoryByID", ctx, int64(1)).Return(nil)

	err := svc.DeleteCategoryByID(ctx, 1)

	assert.NoError(t, err)
}

func TestCategoryService_GetAllPublishedCategories(t *testing.T) {
	repo := new(repoMocks.CategoryRepositoryInterface)
	svc := service2.NewCategoryService(repo)
	ctx := context.Background()

	expected := []entity.CategoryEntity{{ID: 1, Name: "Test"}}
	repo.On("GetAllPublishedCategories", ctx).Return(expected, nil)

	res, err := svc.GetAllPublishedCategories(ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, res)
}
