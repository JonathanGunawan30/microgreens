package service

import (
	"context"
	"product-service/internal/adapter/message"
	"product-service/internal/adapter/repository"
	"product-service/internal/core/domain/entity"
	msgs "product-service/utils/message"

	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ProductServiceInterface interface {
	GetAllProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	GetProductByID(ctx context.Context, productID int64) (*entity.ProductEntity, error)
	CreateProduct(ctx context.Context, product entity.ProductEntity) error
	UpdateProduct(ctx context.Context, product entity.ProductEntity) error
	DeleteProductByID(ctx context.Context, productID int64) error
	GetHomeProducts(ctx context.Context, limit int) ([]entity.ProductEntity, error)
}

type productService struct {
	repo         repository.ProductRepositoryInterface
	repoCategory repository.CategoryRepositoryInterface
	rabbitmq     *amqp.Connection
	exchangeName string
}

func NewProductService(repo repository.ProductRepositoryInterface, rabbitmq *amqp.Connection, exchangeName string, repoCategory repository.CategoryRepositoryInterface) ProductServiceInterface {
	return &productService{
		repo:         repo,
		rabbitmq:     rabbitmq,
		repoCategory: repoCategory,
		exchangeName: exchangeName,
	}
}

func (p *productService) GetAllProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	if query.EndPrice > 0 && query.StartPrice > query.EndPrice {
		query.StartPrice, query.EndPrice = query.EndPrice, query.StartPrice
	}
	return p.repo.GetAllProducts(ctx, query)
}

func (p *productService) SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	if query.EndPrice > 0 && query.StartPrice > query.EndPrice {
		query.StartPrice, query.EndPrice = query.EndPrice, query.StartPrice
	}
	products, count, totalPages, err := p.repo.SearchProducts(ctx, query)

	if err != nil {
		log.Warnf("[ProductService] Failed to search products with Elasticsearch (Fallback to relational database)")
		return p.repo.SearchProductsFallback(ctx, query)
	}

	return products, count, totalPages, nil
}

func (p *productService) GetProductByID(ctx context.Context, productID int64) (*entity.ProductEntity, error) {
	result, err := p.repo.GetProductByID(ctx, productID)
	if err != nil {
		return nil, err
	}

	resultCategory, err := p.repoCategory.GetCategoryBySlug(ctx, result.CategorySlug)
	if err != nil {
		return nil, err
	}

	if resultCategory == nil {
		return nil, msgs.ErrCategoryNotFound
	}

	result.CategoryName = resultCategory.Name
	return result, nil
}

func (p *productService) CreateProduct(ctx context.Context, product entity.ProductEntity) error {

	categoryBySlug, err := p.repoCategory.GetCategoryBySlug(ctx, product.CategorySlug)
	if err != nil {
		return err
	}

	product.CategoryName = categoryBySlug.Name

	createdProduct, err := p.repo.CreateProduct(ctx, product)
	if err != nil {
		return err
	}

	go p.publishProductVariants(*createdProduct, entity.ActionInsert)

	return nil
}

func (p *productService) UpdateProduct(ctx context.Context, product entity.ProductEntity) error {
	updatedProduct, err := p.repo.UpdateProduct(ctx, product)
	if err != nil {
		return err
	}

	go p.publishProductVariants(*updatedProduct, entity.ActionInsert)

	return nil
}

func (p *productService) DeleteProductByID(ctx context.Context, id int64) error {
	productToDelete, err := p.GetProductByID(ctx, id)
	if err != nil {
		return err
	}

	err = p.repo.DeleteProductByID(ctx, id)
	if err != nil {
		return err
	}

	go func(prod entity.ProductEntity) {
		parentToPublish := prod
		parentToPublish.Child = nil

		if err := message.PublishProductEvent(p.rabbitmq, parentToPublish, p.exchangeName, entity.ActionDelete); err != nil {
			log.Errorf("[ProductService] Failed to publish delete event for parent %d: %v", parentToPublish.ID, err)
		}

		if len(prod.Child) > 0 {
			for _, child := range prod.Child {
				if err := message.PublishProductEvent(p.rabbitmq, child, p.exchangeName, entity.ActionDelete); err != nil {
					log.Errorf("[ProductService] Failed to publish delete event for child %d: %v", child.ID, err)
				}
			}
		}
	}(*productToDelete)

	return nil
}

func (p *productService) GetHomeProducts(ctx context.Context, limit int) ([]entity.ProductEntity, error) {
	if limit <= 0 {
		limit = 10
	}
	return p.repo.GetHomeProducts(ctx, limit)
}

func (p *productService) publishProductVariants(product entity.ProductEntity, action entity.ActionType) {
	var variantsToPublish []entity.ProductEntity

	productToPublish := product
	productToPublish.Child = nil

	if len(product.Child) > 0 {
		variantsToPublish = append(variantsToPublish, productToPublish)

		variantsToPublish = append(variantsToPublish, product.Child...)

	} else if product.ParentID == nil {

		variantsToPublish = append(variantsToPublish, productToPublish)

		children, err := p.repo.GetVariantsByParentID(context.Background(), product.ID)
		if err != nil {
			log.Errorf("[ProductService] Failed to fetch children for product %d: %v", product.ID, err)
		} else if len(children) > 0 {
			variantsToPublish = append(variantsToPublish, children...)
		}

	} else {
		variantsToPublish = append(variantsToPublish, productToPublish)
	}

	for _, variant := range variantsToPublish {
		if err := message.PublishProductEvent(p.rabbitmq, variant, p.exchangeName, action); err != nil {
			log.Errorf("[ProductService] Failed to publish %s event for variant %d: %v", action, variant.ID, err)
		} else {
			log.Infof("[ProductService] Successfully queued %s event for product/variant ID: %d", action, variant.ID)
		}
	}
}
