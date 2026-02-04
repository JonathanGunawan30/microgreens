package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/domain/model"
	"product-service/utils/message"
	"strings"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types/enums/sortorder"
	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
)

type ProductRepositoryInterface interface {
	GetAllProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	GetProductByID(ctx context.Context, productID int64) (*entity.ProductEntity, error)
	CreateProduct(ctx context.Context, product entity.ProductEntity) (*entity.ProductEntity, error)
	UpdateProduct(ctx context.Context, product entity.ProductEntity) (*entity.ProductEntity, error)
	DeleteProductByID(ctx context.Context, productID int64) error
	GetHomeProducts(ctx context.Context, limit int) ([]entity.ProductEntity, error)
	DecreaseStock(ctx context.Context, productID int64, quantity int64) error
}

type productRepository struct {
	db *gorm.DB
	es *elasticsearch.TypedClient
}

func NewProductRepository(db *gorm.DB, es *elasticsearch.TypedClient) ProductRepositoryInterface {
	return &productRepository{db: db, es: es}
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

	q := p.db.WithContext(ctx).Table("products")

	if query.Search != "" {
		search := "%" + query.Search + "%"
		whereSQL := `
            (name ILIKE ? OR
			description ILIKE ? OR
            CAST(status AS TEXT) ILIKE ?)`

		q = q.Where(whereSQL, search, search, search)
	}

	if query.StartPrice > 0 {
		q = q.Where("sale_price >= ?", query.StartPrice)
	}

	if query.EndPrice > 0 {
		q = q.Where("sale_price <= ?", query.EndPrice)
	}

	q.Where("status = ?", status)

	if err := q.Count(&count).Error; err != nil {
		log.Errorf("[ProductRepository - 1] GetAllProducts: %v", err)
		return nil, 0, 0, err
	}

	total := (count + int64(limit) - 1) / int64(limit)

	if err := q.Order(orderClause).Limit(limit).Offset(offset).Find(&modelProduct).Error; err != nil {
		log.Errorf("[ProductRepository - 2] GetAllProducts: %v", err)
		return nil, 0, 0, err
	}

	result := make([]entity.ProductEntity, 0, len(modelProduct))

	for _, product := range modelProduct {
		result = append(result, entity.ProductEntity{
			ID:           product.ID,
			CategorySlug: product.CategorySlug,
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
	boolQuery := &types.BoolQuery{}

	if query.Search != "" {
		boolQuery.Must = append(boolQuery.Must, types.Query{
			MultiMatch: &types.MultiMatchQuery{
				Query:     query.Search,
				Fields:    []string{"name^3", "category_slug", "description"},
				Fuzziness: "AUTO",
			},
		})
	}

	if query.CategorySlug != "" {
		boolQuery.Filter = append(boolQuery.Filter, types.Query{
			Term: map[string]types.TermQuery{
				"category_slug.keyword": {
					Value: query.CategorySlug,
				},
			},
		})
	}

	if query.StartPrice > 0 || query.EndPrice > 0 {
		rangeQuery := &types.NumberRangeQuery{}
		if query.StartPrice > 0 {
			val := float64(query.StartPrice)
			rangeQuery.Gte = (*types.Float64)(&val)
		}
		if query.EndPrice > 0 {
			val := float64(query.EndPrice)
			rangeQuery.Lte = (*types.Float64)(&val)
		}

		boolQuery.Filter = append(boolQuery.Filter, types.Query{
			Range: map[string]types.RangeQuery{
				"sale_price": rangeQuery,
			},
		})
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

	sortField := "created_at"
	sortOrder := sortorder.Desc

	if query.OrderBy != "" {
		allowedSort := map[string]string{
			"name":  "name.keyword",
			"price": "sale_price",
			"stock": "stock",
		}

		if val, ok := allowedSort[query.OrderBy]; ok {
			sortField = val
		}
	}

	if query.OrderType == "asc" {
		sortOrder = sortorder.Asc
	}

	res, err := p.es.Search().
		Index("products").
		Query(&types.Query{Bool: boolQuery}).
		From(offset).
		Size(limit).
		Sort(&types.SortOptions{
			SortOptions: map[string]types.FieldSort{
				sortField: {
					Order: &sortOrder,
				},
			},
		}).
		Do(ctx)

	if err != nil {
		log.Errorf("[SearchProudcts - 1] Search Error: %v", err)
		return nil, 0, 0, err
	}

	var products []entity.ProductEntity

	for _, hit := range res.Hits.Hits {
		var m model.Product
		if err := json.Unmarshal(hit.Source_, &m); err == nil {
			products = append(products, entity.ProductEntity{
				ID:           m.ID,
				CategorySlug: m.CategorySlug,
				ParentID:     m.ParentID,
				Name:         m.Name,
				Image:        m.Image,
				Description:  m.Description,
				RegulerPrice: m.RegulerPrice,
				SalePrice:    m.SalePrice,
				Unit:         m.Unit,
				Weight:       m.Weight,
				Stock:        m.Stock,
				Variant:      m.Variant,
				Status:       entity.ProductStatus(m.Status),
				CreatedAt:    m.CreatedAt,
			})
		}

	}
	totalHits := res.Hits.Total.Value
	totalPages := (totalHits + int64(limit) - 1) / int64(limit)

	return products, totalHits, totalPages, nil
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
		Where("status = ?", "active").
		Order("created_at DESC").
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
			RegulerPrice: product.RegulerPrice,
			SalePrice:    product.SalePrice,
			Status:       entity.ProductStatus(product.Status),
		})
	}
	return result, nil
}

func (p *productRepository) DecreaseStock(ctx context.Context, productID int64, quantity int64) error {
	result := p.db.WithContext(ctx).Model(&entity.ProductEntity{}).
		Where("id = ? AND stock >= ?", productID, quantity).
		UpdateColumn("stock", gorm.Expr("stock - ?", quantity))

	if result.Error != nil {
		log.Errorf("[ProductRepository - 1] DecreaseStock productID: %d, quantity: %d, error: %v", productID, quantity, result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("insufficient stock or product not found")
	}

	return nil
}
