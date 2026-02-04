package repository

import (
	"context"
	"errors"
	"fmt"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/domain/model"
	"product-service/utils/message"
	"strings"

	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
)

type CategoryRepositoryInterface interface {
	GetAllCategories(ctx context.Context, query entity.QueryStringCategory) ([]entity.CategoryEntity, int64, int64, error)
	GetCategoryByID(ctx context.Context, categoryID int64) (*entity.CategoryEntity, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*entity.CategoryEntity, error)
	CreateCategory(ctx context.Context, category entity.CategoryEntity) error
	UpdateCategory(ctx context.Context, category entity.CategoryEntity) error
	DeleteCategoryByID(ctx context.Context, categoryID int64) error
	CheckSlugExists(ctx context.Context, slug string) (bool, error)
	GetAllPublishedCategories(ctx context.Context) ([]entity.CategoryEntity, error)
}

type categoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) CategoryRepositoryInterface {
	return &categoryRepository{
		db: db,
	}
}

func (c *categoryRepository) GetAllCategories(ctx context.Context, query entity.QueryStringCategory) ([]entity.CategoryEntity, int64, int64, error) {
	var modelCategory []model.Category
	var count int64

	allowedSort := map[string]bool{"name": true, "status": true, "slug": true}
	orderBy := "created_at"
	if allowedSort[query.OrderBy] {
		orderBy = query.OrderBy
	}

	allowedType := map[string]bool{"asc": true, "desc": true}
	orderType := "desc"
	if allowedType[strings.ToLower(query.OrderType)] {
		orderType = query.OrderType
	}

	orderClause := fmt.Sprintf("%s %s", orderBy, orderType)

	limit := int(query.Limit)
	if limit <= 0 {
		limit = 10
	}

	page := int(query.Page)
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	q := c.db.WithContext(ctx).Table("categories")

	if query.Search != "" {
		search := "%" + query.Search + "%"
		whereSQL := `name ILIKE ? OR CAST(status AS TEXT) ILIKE ? OR slug ILIKE ?`

		q = q.Where(whereSQL, search, search, search)
	}

	if err := q.Count(&count).Error; err != nil {
		log.Errorf("[CategoryRepository - 1] GetAllCategories: %v", err)
		return nil, 0, 0, err
	}

	total := (count + int64(limit) - 1) / int64(limit)

	if err := q.Order(orderClause).
		Select("categories.*, (SELECT count(*) FROM products WHERE products.category_slug = categories.slug) as product_count").
		Limit(limit).Offset(offset).Find(&modelCategory).Error; err != nil {
		log.Errorf("[CategoryRepository - 2] GetAllCategories: %v", err)
		return nil, 0, 0, err
	}

	result := make([]entity.CategoryEntity, 0, len(modelCategory))

	for _, category := range modelCategory {
		result = append(result, entity.CategoryEntity{
			ID:           category.ID,
			ParentID:     category.ParentID,
			Name:         category.Name,
			Icon:         category.Icon,
			Status:       category.Status,
			Slug:         category.Slug,
			Description:  category.Description,
			ProductCount: category.ProductCount,
		})
	}

	return result, count, total, nil
}

func (c *categoryRepository) GetCategoryByID(ctx context.Context, categroyID int64) (*entity.CategoryEntity, error) {
	var modelCategory model.Category

	if err := c.db.WithContext(ctx).Where("id = ?", categroyID).First(&modelCategory).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("[CategoryRepository - 1] GetByID: %v", err)
			return nil, message.ErrCategoryNotFound
		}
		log.Errorf("[CategoryRepository - 2] GetByID: %v", err)
		return nil, err
	}

	return &entity.CategoryEntity{
		ID:          modelCategory.ID,
		ParentID:    modelCategory.ParentID,
		Name:        modelCategory.Name,
		Icon:        modelCategory.Icon,
		Status:      modelCategory.Status,
		Slug:        modelCategory.Slug,
		Description: modelCategory.Description,
	}, nil

}

func (c *categoryRepository) GetCategoryBySlug(ctx context.Context, slug string) (*entity.CategoryEntity, error) {
	var modelCategory model.Category

	if err := c.db.WithContext(ctx).Where("slug = ?", slug).First(&modelCategory).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("[CategoryRepository - 1] GetBySlug: %v", err)
			return nil, message.ErrCategoryNotFound
		}
		log.Errorf("[CategoryRepository -2] GetBySlug: %v", err)
		return nil, err
	}

	return &entity.CategoryEntity{
		ID:           modelCategory.ID,
		ParentID:     modelCategory.ParentID,
		Name:         modelCategory.Name,
		Icon:         modelCategory.Icon,
		Status:       modelCategory.Status,
		Slug:         modelCategory.Slug,
		Description:  modelCategory.Description,
		ProductCount: modelCategory.ProductCount,
	}, nil
}

func (c *categoryRepository) CreateCategory(ctx context.Context, category entity.CategoryEntity) error {
	categoryModel := model.Category{
		ParentID:    category.ParentID,
		Name:        category.Name,
		Icon:        category.Icon,
		Status:      category.Status,
		Slug:        category.Slug,
		Description: category.Description,
	}
	if err := c.db.WithContext(ctx).Create(&categoryModel).Error; err != nil {
		log.Errorf("[CategoryRepository - 1] CrateCategory: %v", err)
		return err
	}

	return nil
}

func (c *categoryRepository) UpdateCategory(ctx context.Context, category entity.CategoryEntity) error {
	categoryModel := model.Category{
		ParentID:    category.ParentID,
		Name:        category.Name,
		Icon:        category.Icon,
		Status:      category.Status,
		Slug:        category.Slug,
		Description: category.Description,
	}

	tx := c.db.WithContext(ctx).Model(&model.Category{}).
		Where("id = ?", category.ID).
		Select("parent_id", "name", "icon", "status", "slug", "description").
		Updates(&categoryModel)

	if tx.Error != nil {
		log.Errorf("[CategoryRepository - 1] UpdateCategory: %v", tx.Error)
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		log.Warnf("[CategoryRepository - 2] UpdateCategory: %v", tx.Error)
		return message.ErrCategoryNotFound
	}
	return nil
}

func (c *categoryRepository) DeleteCategoryByID(ctx context.Context, categoryID int64) error {
	var count int64

	var modelCategory model.Category
	if err := c.db.WithContext(ctx).Where("id = ?", categoryID).First(&modelCategory).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return message.ErrCategoryNotFound
		}
		log.Errorf("[CategoryRepository - 1] DeleteCategoryByID: %v", err)
		return err
	}

	if err := c.db.WithContext(ctx).Model(&model.Product{}).Select("category_slug").Where("category_slug = ?", modelCategory.Slug).Count(&count).Error; err != nil {
		log.Errorf("[CategoryRepository - 1] DeleteCategoryByID: %v", err)
		return err
	}

	if count > 0 {
		log.Warnf("[CategoryRepository - 2] DeleteCategoryByID: Cannot delete category %d, has %d products", categoryID, count)
		return message.ErrCategoryHasProducts
	}

	tx := c.db.WithContext(ctx).Where("id = ?", categoryID).Delete(&model.Category{})

	if tx.Error != nil {
		log.Errorf("[CategoryRepository - 1] DeleteCategoryByID: %v", tx.Error)
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		log.Warnf("[CategoryRepository - 2] DeleteCategoryByID: %v", tx.Error)
		return message.ErrCategoryNotFound
	}
	return nil

}

func (c *categoryRepository) CheckSlugExists(ctx context.Context, slug string) (bool, error) {
	var count int64
	if err := c.db.WithContext(ctx).Model(&model.Category{}).Where("slug = ?", slug).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (c *categoryRepository) GetAllPublishedCategories(ctx context.Context) ([]entity.CategoryEntity, error) {
	var modelCategory []model.Category

	if err := c.db.WithContext(ctx).Table("categories").Select("id", "parent_id", "name", "icon", "slug").Where("status = ?", true).Order("name ASC").Find(&modelCategory).Error; err != nil {
		log.Errorf("[CategoryRepository - 2] GetAllPublishedCategories: %v", err)
		return nil, err
	}

	result := make([]entity.CategoryEntity, 0, len(modelCategory))

	for _, category := range modelCategory {
		result = append(result, entity.CategoryEntity{
			ID:       category.ID,
			ParentID: category.ParentID,
			Name:     category.Name,
			Icon:     category.Icon,
			Slug:     category.Slug,
		})
	}

	return result, nil
}
