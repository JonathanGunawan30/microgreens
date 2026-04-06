package service

import (
	"context"
	"product-service/internal/adapter/handler/response"
	"product-service/internal/adapter/repository"
	"product-service/internal/core/domain/entity"
	msg "product-service/utils/message"
)

type CartServiceInterface interface {
	AddToCart(ctx context.Context, userID, productID, quantity int64) error
	GetCart(ctx context.Context, userID int64) ([]response.CartResponse, error)
	RemoveFromCart(ctx context.Context, userID, productID int64) error
	ClearCart(ctx context.Context, userID int64) error
	DecreaseItem(ctx context.Context, userID, productID, quantity int64) error
}

type cartService struct {
	cartRepository    repository.CartRedisRepositoryInterface
	productRepository repository.ProductRepositoryInterface
}

func NewCartService(cartRepo repository.CartRedisRepositoryInterface, productRepo repository.ProductRepositoryInterface) CartServiceInterface {
	return &cartService{
		cartRepository:    cartRepo,
		productRepository: productRepo,
	}
}

func (c *cartService) AddToCart(ctx context.Context, userID, productID, quantity int64) error {
	if quantity <= 0 {
		return msg.ErrProductLTZero
	}

	getProductByID, err := c.productRepository.GetProductByID(ctx, productID)
	if err != nil {
		return err
	}

	if getProductByID == nil {
		return msg.ErrProductNotFound
	}

	cartItems, err := c.cartRepository.GetCart(ctx, userID)
	if err != nil {
		return err
	}

	var currentQtyInCart int64 = 0
	for _, item := range cartItems {
		if item.ProductID == productID {
			currentQtyInCart = item.Quantity
			break
		}
	}

	totalRequestedQty := currentQtyInCart + quantity
	if totalRequestedQty > getProductByID.Stock {
		return msg.ErrQuantityExceeds
	}

	item := entity.CartItem{
		ProductID: productID,
		Quantity:  quantity,
	}

	return c.cartRepository.AddToCart(ctx, userID, item)

}

func (c *cartService) GetCart(ctx context.Context, userID int64) ([]response.CartResponse, error) {
	cartItems, err := c.cartRepository.GetCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	var cartResponses []response.CartResponse
	for _, item := range cartItems {
		product, err := c.productRepository.GetProductByID(ctx, item.ProductID)
		if err != nil {
			continue
		}

		cartResponses = append(cartResponses, response.CartResponse{
			ID:            product.ID,
			Name:          product.Name,
			Image:         product.Image,
			ProductStatus: string(product.Status),
			SalePrice:     product.SalePrice,
			Quantity:      item.Quantity,
			Unit:          product.Unit,
			Weight:        product.Weight,
		})
	}

	return cartResponses, nil
}

func (c *cartService) RemoveFromCart(ctx context.Context, userID, productID int64) error {
	return c.cartRepository.RemoveFromCart(ctx, userID, productID)
}

func (c *cartService) ClearCart(ctx context.Context, userID int64) error {
	return c.cartRepository.ClearCart(ctx, userID)
}

func (c *cartService) DecreaseItem(ctx context.Context, userID, productID, quantity int64) error {
	if quantity <= 0 {
		return msg.ErrProductLTZero
	}

	return c.cartRepository.DecreaseItem(ctx, userID, productID, quantity)
}
