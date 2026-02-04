package service

import (
	"context"
	"product-service/internal/adapter/repository"
	"product-service/internal/core/domain/entity"
	"product-service/utils/conv"
	"product-service/utils/message"

	"github.com/labstack/gommon/log"
)

type CategoryServiceInterface interface {
	GetAllCategories(ctx context.Context, query entity.QueryStringCategory) ([]entity.CategoryEntity, int64, int64, error)
	GetCategoryByID(ctx context.Context, categoryID int64) (*entity.CategoryEntity, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*entity.CategoryEntity, error)
	CreateCategory(ctx context.Context, category entity.CategoryEntity) error
	UpdateCategory(ctx context.Context, category entity.CategoryEntity) error
	DeleteCategoryByID(ctx context.Context, categoryID int64) error
	GetAllPublishedCategories(ctx context.Context) ([]entity.CategoryEntity, error)
}
type categoryService struct {
	repo repository.CategoryRepositoryInterface
}

func NewCategoryService(repo repository.CategoryRepositoryInterface) CategoryServiceInterface {
	return &categoryService{repo: repo}
}

func (c *categoryService) GetAllCategories(ctx context.Context, query entity.QueryStringCategory) ([]entity.CategoryEntity, int64, int64, error) {
	return c.repo.GetAllCategories(ctx, query)
}

func (c *categoryService) GetCategoryByID(ctx context.Context, categoryID int64) (*entity.CategoryEntity, error) {
	return c.repo.GetCategoryByID(ctx, categoryID)
}

func (c *categoryService) GetCategoryBySlug(ctx context.Context, slug string) (*entity.CategoryEntity, error) {
	return c.repo.GetCategoryBySlug(ctx, slug)
}

func (c *categoryService) CreateCategory(ctx context.Context, category entity.CategoryEntity) error {
	slug := conv.GenerateSlug(category.Name)
	exists, err := c.repo.CheckSlugExists(ctx, slug)
	if err != nil {
		log.Errorf("[CategoryService - 1] CreateCategory: %v", err)
		return err
	}

	if exists {
		log.Warn("[CategoryService - 2] CreateCategory: Category Already Exists")
		return message.ErrCategoryAlreadyExists
	}

	category.Slug = slug
	return c.repo.CreateCategory(ctx, category)
}

func (c *categoryService) UpdateCategory(ctx context.Context, category entity.CategoryEntity) error {
	existingCategory, err := c.repo.GetCategoryByID(ctx, category.ID)
	if err != nil {
		log.Errorf("[CategoryService - 1] UpdateCategory: %v", err)
		return err
	}

	if existingCategory == nil {
		return message.ErrCategoryNotFound
	}

	category.Slug = existingCategory.Slug

	if category.Name != existingCategory.Name {
		newSlug := conv.GenerateSlug(category.Name)

		exists, err := c.repo.CheckSlugExists(ctx, newSlug)
		if err != nil {
			log.Errorf("[CategoryService - 2] UpdateCategory: %v", err)
			return err
		}

		if exists {
			log.Warn("[CategoryService - 3] UpdateCategory: Category Already Exists")
			return message.ErrCategoryAlreadyExists
		}
		category.Slug = newSlug
	}

	return c.repo.UpdateCategory(ctx, category)
}

func (c *categoryService) DeleteCategoryByID(ctx context.Context, categoryID int64) error {
	return c.repo.DeleteCategoryByID(ctx, categoryID)
}

func (c *categoryService) GetAllPublishedCategories(ctx context.Context) ([]entity.CategoryEntity, error) {
	return c.repo.GetAllPublishedCategories(ctx)
}
