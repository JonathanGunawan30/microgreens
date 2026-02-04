package service

import (
	"context"
	"product-service/internal/adapter/message"
	"product-service/internal/adapter/repository"
	"product-service/internal/core/domain/entity"

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

func NewProductService(repo repository.ProductRepositoryInterface, rabbitmq *amqp.Connection, esQueueName string) ProductServiceInterface {
	return &productService{repo: repo, rabbitmq: rabbitmq, esQueueName: esQueueName}
}

type productService struct {
	repo        repository.ProductRepositoryInterface
	rabbitmq    *amqp.Connection
	esQueueName string
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
	return p.repo.SearchProducts(ctx, query)
}

func (p *productService) GetProductByID(ctx context.Context, productID int64) (*entity.ProductEntity, error) {
	return p.repo.GetProductByID(ctx, productID)
}

func (p *productService) CreateProduct(ctx context.Context, product entity.ProductEntity) error {

	createdProduct, err := p.repo.CreateProduct(ctx, product)
	if err != nil {
		return err
	}

	msg := entity.EsSyncMessage{
		Action: entity.ActionInsert,
		Data:   createdProduct,
		ID:     createdProduct.ID,
	}

	go message.PublishProductWithRetry(p.rabbitmq, msg, p.esQueueName)

	return nil
}

func (p *productService) UpdateProduct(ctx context.Context, product entity.ProductEntity) error {
	updatedProduct, err := p.repo.UpdateProduct(ctx, product)
	if err != nil {
		return err
	}

	msg := entity.EsSyncMessage{
		Action: entity.ActionInsert,
		Data:   updatedProduct,
		ID:     updatedProduct.ID,
	}

	go message.PublishProductWithRetry(p.rabbitmq, msg, p.esQueueName)

	return nil
}

func (p *productService) DeleteProductByID(ctx context.Context, id int64) error {
	err := p.repo.DeleteProductByID(ctx, id)
	if err != nil {
		return err
	}

	msg := entity.EsSyncMessage{
		Action: entity.ActionDelete,
		ID:     id,
		Data:   nil,
	}

	go message.PublishProductWithRetry(p.rabbitmq, msg, p.esQueueName)

	return nil
}

func (p *productService) GetHomeProducts(ctx context.Context, limit int) ([]entity.ProductEntity, error) {
	if limit <= 0 {
		limit = 5
	}
	return p.repo.GetHomeProducts(ctx, limit)
}
