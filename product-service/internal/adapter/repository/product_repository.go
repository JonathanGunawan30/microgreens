package repository

import (
	"context"
	"errors"
	"fmt"
	"product-service/internal/adapter/search"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/domain/model"
	"product-service/utils/message"
	"strings"

	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
)

type ProductRepositoryInterface interface {
	GetAllProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	SearchProductsFallback(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	SuggestProducts(ctx context.Context, keyword string, limit int) ([]entity.ProductEntity, error)
	SuggestProductsFallback(ctx context.Context, keyword string, limit int) ([]entity.ProductEntity, error)
	GetProductByID(ctx context.Context, productID int64) (*entity.ProductEntity, error)
	CreateProduct(ctx context.Context, product entity.ProductEntity) (*entity.ProductEntity, error)
	UpdateProduct(ctx context.Context, product entity.ProductEntity) (*entity.ProductEntity, error)
	DeleteProductByID(ctx context.Context, productID int64) error
	GetHomeProducts(ctx context.Context, limit int) ([]entity.ProductEntity, error)
	DecreaseStock(ctx context.Context, productID int64, quantity int64) error
	GetVariantsByParentID(ctx context.Context, parentID int64) ([]entity.ProductEntity, error)
}

type productRepository struct {
	db           *gorm.DB
	searchEngine search.SearchEngineInterface
}

func NewProductRepository(db *gorm.DB, searchEngine search.SearchEngineInterface) ProductRepositoryInterface {
	return &productRepository{db: db, searchEngine: searchEngine}
}

func (p *productRepository) GetAllProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	var modelProduct []model.Product
	var count int64

	allowedSort := map[string]bool{
		"name": true, "reguler_price": true,
		"sale_price": true, "unit": true, "weight": true,
		"stock": true, "variant": true, "status": true,
	}

	orderBy := "created_at"
	if allowedSort[query.OrderBy] {
		orderBy = query.OrderBy
	}

	allowedType := map[string]bool{"asc": true, "desc": true}
	orderType := "desc"
	if allowedType[strings.ToLower(query.OrderType)] {
		orderType = query.OrderType
	}

	allowedStatus := map[string]bool{"draft": true, "active": true, "inactive": true}
	status := "active"
	if allowedStatus[strings.ToLower(query.Status)] {
		status = query.Status
	}

	orderClause := fmt.Sprintf("products.%s %s", orderBy, orderType)

	limit := int(query.Limit)
	if limit <= 0 {
		limit = 10
	}

	page := int(query.Page)
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	q := p.db.WithContext(ctx).Table("products")

	if query.IsParent == "true" {
		q = q.Where("products.parent_id IS NULL")
	} else if query.IsParent == "false" {
		q = q.Where("products.parent_id IS NOT NULL")
	}

	if query.Search != "" {
		search := "%" + query.Search + "%"
		whereSQL := `
            (products.name ILIKE ? OR
			products.description ILIKE ? OR
            CAST(products.status AS TEXT) ILIKE ? OR
			CAST(products.parent_id AS TEXT) ILIKE ?)`

		q = q.Where(whereSQL, search, search, search, search)
	}

	if query.StartPrice > 0 {
		q = q.Where("products.sale_price >= ?", query.StartPrice)
	}

	if query.EndPrice > 0 {
		q = q.Where("products.sale_price <= ?", query.EndPrice)
	}

	q.Where("products.status = ?", status)

	if err := q.Count(&count).Error; err != nil {
		log.Errorf("[ProductRepository - 1] GetAllProducts: %v", err)
		return nil, 0, 0, err
	}

	total := (count + int64(limit) - 1) / int64(limit)

	q = q.Select("products.*, categories.name as category_name").
		Joins("LEFT JOIN categories ON categories.slug = products.category_slug")

	if err := q.Order(orderClause).Limit(limit).Offset(offset).Find(&modelProduct).Error; err != nil {
		log.Errorf("[ProductRepository - 2] GetAllProducts: %v", err)
		return nil, 0, 0, err
	}

	result := make([]entity.ProductEntity, 0, len(modelProduct))

	for _, product := range modelProduct {
		result = append(result, entity.ProductEntity{
			ID:           product.ID,
			CategorySlug: product.CategorySlug,
			CategoryName: product.CategoryName,
			ParentID:     product.ParentID,
			Name:         product.Name,
			Image:        product.Image,
			Description:  product.Description,
			RegulerPrice: product.RegulerPrice,
			SalePrice:    product.SalePrice,
			Unit:         product.Unit,
			Weight:       product.Weight,
			Stock:        product.Stock,
			Variant:      product.Variant,
			Status:       entity.ProductStatus(product.Status),
			CreatedAt:    product.CreatedAt,
		})
	}
	return result, count, total, nil
}

func (p *productRepository) SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	if p.searchEngine == nil {
		return nil, 0, 0, errors.New("search engine not available")
	}
	return p.searchEngine.SearchProducts(ctx, query)
}

func (p *productRepository) SuggestProducts(ctx context.Context, keyword string, limit int) ([]entity.ProductEntity, error) {
	if p.searchEngine == nil {
		return nil, errors.New("search engine not available")
	}
	return p.searchEngine.SuggestProducts(ctx, keyword, limit)
}

func (p *productRepository) SuggestProductsFallback(ctx context.Context, keyword string, limit int) ([]entity.ProductEntity, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return []entity.ProductEntity{}, nil
	}
	if limit <= 0 {
		limit = 5
	}

	var modelProducts []model.Product
	likePattern := "%" + keyword + "%"
	prefixPattern := keyword + "%"

	err := p.db.WithContext(ctx).Table("products").
		Select("products.*, categories.name as category_name").
		Joins("LEFT JOIN categories ON categories.slug = products.category_slug").
		Where("products.status = ? AND products.parent_id IS NULL", "active").
		Where("(products.name ILIKE ? OR products.category_slug ILIKE ? OR categories.name ILIKE ?)", likePattern, likePattern, likePattern).
		Order(gorm.Expr("CASE WHEN products.name ILIKE ? THEN 0 ELSE 1 END, products.name ASC", prefixPattern)).
		Limit(limit).
		Find(&modelProducts).Error
	if err != nil {
		log.Errorf("[ProductRepository] SuggestProductsFallback: %v", err)
		return nil, err
	}

	result := make([]entity.ProductEntity, 0, len(modelProducts))
	for _, product := range modelProducts {
		result = append(result, entity.ProductEntity{
			ID:           product.ID,
			CategorySlug: product.CategorySlug,
			CategoryName: product.CategoryName,
			ParentID:     product.ParentID,
			Name:         product.Name,
			Image:        product.Image,
			RegulerPrice: product.RegulerPrice,
			SalePrice:    product.SalePrice,
			Unit:         product.Unit,
			Weight:       product.Weight,
			Stock:        product.Stock,
			Variant:      product.Variant,
			Status:       entity.ProductStatus(product.Status),
			CreatedAt:    product.CreatedAt,
		})
	}
	return result, nil
}

func (p *productRepository) SearchProductsFallback(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	var modelProduct []model.Product
	var count int64

	q := p.db.WithContext(ctx).Table("products")

	q = q.Where("products.status = ?", "active").
		Where("products.parent_id IS NULL")

	if query.CategorySlug != "" {
		q = q.Where("products.category_slug = ?", query.CategorySlug)
	}

	if query.StartPrice > 0 {
		q = q.Where("products.sale_price >= ?", query.StartPrice)
	}
	if query.EndPrice > 0 {
		q = q.Where("products.sale_price <= ?", query.EndPrice)
	}

	if query.Search != "" {
		search := "%" + query.Search + "%"
		whereSQL := `(products.name ILIKE ? OR products.category_slug ILIKE ? OR products.description ILIKE ?)`
		q = q.Where(whereSQL, search, search, search)
	}

	if err := q.Count(&count).Error; err != nil {
		log.Errorf("[ProductRepository] SearchProductsFallback Count: %v", err)
		return nil, 0, 0, err
	}

	limit := int(query.Limit)
	if limit <= 0 {
		limit = 10
	}
	page := int(query.Page)
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	total := (count + int64(limit) - 1) / int64(limit)

	orderBy := "created_at"
	if query.OrderBy != "" {
		allowedSort := map[string]string{
			"name":  "name",
			"price": "sale_price",
			"stock": "stock",
			"id":    "id",
		}
		if val, ok := allowedSort[query.OrderBy]; ok {
			orderBy = val
		}
	}

	orderType := "DESC"
	if strings.ToLower(query.OrderType) == "asc" {
		orderType = "ASC"
	}
	orderClause := fmt.Sprintf("products.%s %s", orderBy, orderType)

	q = q.Select("products.*, categories.name as category_name").
		Joins("LEFT JOIN categories ON categories.slug = products.category_slug")

	if err := q.Order(orderClause).Limit(limit).Offset(offset).Find(&modelProduct).Error; err != nil {
		log.Errorf("[ProductRepository] SearchProductsFallback Find: %v", err)
		return nil, 0, 0, err
	}

	result := make([]entity.ProductEntity, 0, len(modelProduct))
	for _, product := range modelProduct {
		result = append(result, entity.ProductEntity{
			ID:           product.ID,
			CategorySlug: product.CategorySlug,
			CategoryName: product.CategoryName,
			ParentID:     product.ParentID,
			Name:         product.Name,
			Image:        product.Image,
			Description:  product.Description,
			RegulerPrice: product.RegulerPrice,
			SalePrice:    product.SalePrice,
			Unit:         product.Unit,
			Weight:       product.Weight,
			Stock:        product.Stock,
			Variant:      product.Variant,
			Status:       entity.ProductStatus(product.Status),
			CreatedAt:    product.CreatedAt,
		})
	}

	return result, count, total, nil
}

func (p *productRepository) GetProductByID(ctx context.Context, productID int64) (*entity.ProductEntity, error) {
	var modelProduct model.Product

	if err := p.db.WithContext(ctx).Where("id = ?", productID).First(&modelProduct).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("[ProductRepository - 1] GetProductByID: %v", err)
			return nil, message.ErrProductNotFound
		}
		log.Errorf("[ProductRepository - 2] GetProductByID: %v", err)
		return nil, err
	}

	var modelProductChild []model.Product
	if modelProduct.ParentID == nil {
		if err := p.db.WithContext(ctx).Where("parent_id = ?", productID).Find(&modelProductChild).Error; err != nil {
			log.Errorf("[ProductRepository - 3] GetProductByID: %v", err)
			return nil, err
		}
	}

	childEntity := make([]entity.ProductEntity, 0, len(modelProductChild))
	if len(modelProductChild) > 0 {
		for _, child := range modelProductChild {
			childEntity = append(childEntity, entity.ProductEntity{
				ID:           child.ID,
				CategorySlug: child.CategorySlug,
				ParentID:     child.ParentID,
				Name:         child.Name,
				Image:        child.Image,
				Description:  child.Description,
				RegulerPrice: child.RegulerPrice,
				SalePrice:    child.SalePrice,
				Unit:         child.Unit,
				Weight:       child.Weight,
				Stock:        child.Stock,
				Variant:      child.Variant,
				Status:       entity.ProductStatus(child.Status),
			})
		}
	}

	return &entity.ProductEntity{
		ID:           modelProduct.ID,
		CategorySlug: modelProduct.CategorySlug,
		ParentID:     modelProduct.ParentID,
		Name:         modelProduct.Name,
		Image:        modelProduct.Image,
		Description:  modelProduct.Description,
		RegulerPrice: modelProduct.RegulerPrice,
		SalePrice:    modelProduct.SalePrice,
		Unit:         modelProduct.Unit,
		Weight:       modelProduct.Weight,
		Stock:        modelProduct.Stock,
		Variant:      modelProduct.Variant,
		Status:       entity.ProductStatus(modelProduct.Status),
		Child:        childEntity,
	}, nil
}

func (p *productRepository) CreateProduct(ctx context.Context, product entity.ProductEntity) (*entity.ProductEntity, error) {
	tx := p.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		log.Errorf("[ProductRepository - 1] CreateProduct: %v", tx.Error)
		return nil, tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	modelProduct := model.Product{
		CategorySlug: product.CategorySlug,
		Name:         product.Name,
		Image:        product.Image,
		Description:  product.Description,
		RegulerPrice: product.RegulerPrice,
		SalePrice:    product.SalePrice,
		Unit:         product.Unit,
		Weight:       product.Weight,
		Stock:        product.Stock,
		Variant:      product.Variant,
		Status:       model.ProductStatus(product.Status),
		ParentID:     nil,
	}

	if err := tx.Create(&modelProduct).Error; err != nil {
		tx.Rollback()
		log.Errorf("[ProductRepository -2] CreateProduct: %v", err)
		return nil, err
	}

	var modelChildren []model.Product
	if len(product.Child) > 0 {
		for _, child := range product.Child {
			modelChild := model.Product{
				CategorySlug: product.CategorySlug,
				ParentID:     &modelProduct.ID,
				Name:         product.Name,
				Image:        child.Image,
				Description:  product.Description,
				RegulerPrice: child.RegulerPrice,
				SalePrice:    child.SalePrice,
				Unit:         product.Unit,
				Weight:       child.Weight,
				Stock:        child.Stock,
				Variant:      product.Variant,
				Status:       model.ProductStatus(product.Status),
			}
			modelChildren = append(modelChildren, modelChild)
		}

		if err := tx.Create(&modelChildren).Error; err != nil {
			tx.Rollback()
			log.Errorf("[ProductRepository - 3] CreateProduct (Children): %v", err)
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		log.Errorf("[ProductRepository - 4] CreateProduct (Commit): %v", err)
		return nil, err
	}

	product.ID = modelProduct.ID
	product.CreatedAt = modelProduct.CreatedAt

	if len(product.Child) > 0 && len(modelChildren) > 0 {
		for i := range product.Child {
			product.Child[i].ID = modelChildren[i].ID
			product.Child[i].ParentID = &modelProduct.ID
			product.Child[i].CategorySlug = modelProduct.CategorySlug
			product.Child[i].CategoryName = product.CategoryName
			product.Child[i].CreatedAt = modelProduct.CreatedAt
			product.Child[i].Name = modelProduct.Name
			product.Child[i].Description = modelProduct.Description
			product.Child[i].Unit = modelProduct.Unit
			product.Child[i].Status = entity.ProductStatus(modelProduct.Status)
			product.Child[i].Variant = modelProduct.Variant
		}
	}
	return &product, nil

}

func (p *productRepository) UpdateProduct(ctx context.Context, product entity.ProductEntity) (*entity.ProductEntity, error) {
	tx := p.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		log.Errorf("[ProductRepository - 1] UpdateProduct: %v", tx.Error)
		return nil, tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	modelProduct := model.Product{
		ID:           product.ID,
		CategorySlug: product.CategorySlug,
		Name:         product.Name,
		Image:        product.Image,
		Description:  product.Description,
		RegulerPrice: product.RegulerPrice,
		SalePrice:    product.SalePrice,
		Unit:         product.Unit,
		Weight:       product.Weight,
		Stock:        product.Stock,
		Variant:      product.Variant,
		Status:       model.ProductStatus(product.Status),
	}

	result := tx.Model(&model.Product{}).Where("id = ?", product.ID).Updates(&modelProduct)
	if result.Error != nil {
		tx.Rollback()
		log.Errorf("[ProductRepository - 2] UpdateProduct (Parent): %v", result.Error)
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		log.Warnf("[ProductRepository - 3] UpdateProduct: %v", tx.Error)
		return nil, message.ErrProductNotFound
	}

	if err := tx.Where("parent_id = ?", product.ID).Delete(&model.Product{}).Error; err != nil {
		tx.Rollback()
		log.Errorf("[ProductRepository - 4] UpdateProduct (Delete Children): %v", err)
		return nil, err
	}

	var modelChildren []model.Product
	if len(product.Child) > 0 {
		for _, child := range product.Child {
			modelChild := model.Product{
				CategorySlug: product.CategorySlug,
				ParentID:     &modelProduct.ID,
				Name:         product.Name,
				Image:        child.Image,
				Description:  product.Description,
				RegulerPrice: child.RegulerPrice,
				SalePrice:    child.SalePrice,
				Unit:         product.Unit,
				Weight:       child.Weight,
				Stock:        child.Stock,
				Variant:      product.Variant,
				Status:       model.ProductStatus(product.Status),
			}
			modelChildren = append(modelChildren, modelChild)
		}

		if err := tx.Create(&modelChildren).Error; err != nil {
			tx.Rollback()
			log.Errorf("[ProductRepository - 5] UpdateProduct (Insert New Children): %v", err)
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		log.Errorf("[ProductRepository - 6] UpdateProduct (Commit): %v", err)
		return nil, err
	}

	product.ID = modelProduct.ID

	if len(product.Child) > 0 && len(modelChildren) > 0 {
		for i := range product.Child {
			product.Child[i].ID = modelChildren[i].ID
			product.Child[i].ParentID = &modelProduct.ID
			product.Child[i].CategorySlug = modelProduct.CategorySlug
			product.Child[i].Name = modelProduct.Name
			product.Child[i].Description = modelProduct.Description
			product.Child[i].Unit = modelProduct.Unit
			product.Child[i].Status = entity.ProductStatus(modelProduct.Status)
			product.Child[i].Variant = modelProduct.Variant
		}
	}
	return &product, nil
}

func (p *productRepository) DeleteProductByID(ctx context.Context, productID int64) error {
	tx := p.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		log.Errorf("[ProductRepository - 1] DeleteProductByID: %v", tx.Error)
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Where("parent_id = ?", productID).Delete(&model.Product{}).Error; err != nil {
		tx.Rollback()
		log.Errorf("[ProductRepository - 2] DeleteProductByID (Children): %v", err)
		return err
	}

	result := tx.Where("id = ?", productID).Delete(&model.Product{})
	if result.Error != nil {
		tx.Rollback()
		log.Errorf("[ProductRepository - 3] DeleteProductByID (Parent): %v", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		log.Warnf("[ProductRepository - 4] DeleteProductByID: %v", tx.Error)
		return message.ErrProductNotFound
	}

	if err := tx.Commit().Error; err != nil {
		log.Errorf("[ProductRepository - 5] DeleteProductByID (Commit): %v", err)
		return err
	}

	return nil

}

func (p *productRepository) GetHomeProducts(ctx context.Context, limit int) ([]entity.ProductEntity, error) {
	var modelProduct []model.Product

	err := p.db.WithContext(ctx).Table("products").
		Select("products.*, categories.name as category_name").
		Joins("LEFT JOIN categories ON categories.slug = products.category_slug").
		Where("products.status = ?", "active").
		Where("products.parent_id IS NULL").
		Order("products.created_at DESC").
		Limit(limit).
		Find(&modelProduct).Error

	if err != nil {
		log.Errorf("[ProductRepository] GetHomeProducts: %v", err)
		return nil, err
	}

	result := make([]entity.ProductEntity, 0, len(modelProduct))

	for _, product := range modelProduct {
		result = append(result, entity.ProductEntity{
			ID:           product.ID,
			Name:         product.Name,
			Image:        product.Image,
			CategorySlug: product.CategorySlug,
			CategoryName: product.CategoryName,
			RegulerPrice: product.RegulerPrice,
			SalePrice:    product.SalePrice,
			Status:       entity.ProductStatus(product.Status),
		})
	}
	return result, nil
}

func (p *productRepository) DecreaseStock(ctx context.Context, productID int64, quantity int64) error {
	tx := p.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	var product model.Product
	if err := tx.Select("id, parent_id").Where("id = ?", productID).First(&product).Error; err != nil {
		tx.Rollback()
		return err
	}

	log.Printf("product.ID=%d, product.ParentID=%v", product.ID, product.ParentID)

	result := tx.Model(&model.Product{}).
		Where("id = ? AND stock >= ?", productID, quantity).
		UpdateColumn("stock", gorm.Expr("stock - ?", quantity))

	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		return fmt.Errorf("insufficient stock")
	}

	if product.ParentID != nil {
		log.Printf("updating parent stock, parentID=%d", *product.ParentID)

		parentResult := tx.Model(&model.Product{}).
			Where("id = ?", *product.ParentID).
			UpdateColumn("stock", gorm.Expr("stock - ?", quantity))

		if parentResult.Error != nil {
			tx.Rollback()
			return parentResult.Error
		}

		if parentResult.RowsAffected == 0 {
			tx.Rollback()
			return fmt.Errorf("failed to update parent stock: parent not found or no rows affected")
		}

		log.Printf("parent stock updated, rows affected=%d", parentResult.RowsAffected)
	} else {
		log.Printf("skipping parent update: ParentID is nil")
	}

	return tx.Commit().Error
}

func (p *productRepository) GetVariantsByParentID(ctx context.Context, parentID int64) ([]entity.ProductEntity, error) {
	var modelVariants []model.Product

	if err := p.db.WithContext(ctx).Where("parent_id = ?", parentID).Find(&modelVariants).Error; err != nil {
		log.Errorf("[ProductRepository] GetVariantsByParentID: %v", err)
		return nil, err
	}

	variantEntities := make([]entity.ProductEntity, 0, len(modelVariants))

	for _, variant := range modelVariants {
		variantEntities = append(variantEntities, entity.ProductEntity{
			ID:           variant.ID,
			CategorySlug: variant.CategorySlug,
			ParentID:     variant.ParentID,
			Name:         variant.Name,
			Image:        variant.Image,
			Description:  variant.Description,
			RegulerPrice: variant.RegulerPrice,
			SalePrice:    variant.SalePrice,
			Unit:         variant.Unit,
			Weight:       variant.Weight,
			Stock:        variant.Stock,
			Variant:      variant.Variant,
			Status:       entity.ProductStatus(variant.Status),
		})
	}

	return variantEntities, nil
}
