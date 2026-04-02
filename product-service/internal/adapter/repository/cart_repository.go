package repository

import (
	"context"
	"fmt"
	"product-service/internal/core/domain/entity"
	"strconv"
	"time"

	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
)

type CartRedisRepositoryInterface interface {
	AddToCart(ctx context.Context, userID int64, item entity.CartItem) error
	GetCart(ctx context.Context, userID int64) ([]entity.CartItem, error)
	RemoveFromCart(ctx context.Context, userID, productID int64) error
	ClearCart(ctx context.Context, userID int64) error
	DecreaseItem(ctx context.Context, userID, productID, quantity int64) error
}

type CartRedisRepository struct {
	Client *redis.Client
}

func NewCartRedisRepository(client *redis.Client) CartRedisRepositoryInterface {
	return &CartRedisRepository{Client: client}
}

func cartKey(userID int64) string {
	return fmt.Sprintf("cart:%d", userID)
}

func (c *CartRedisRepository) AddToCart(ctx context.Context, userID int64, item entity.CartItem) error {
	key := cartKey(userID)

	err := c.Client.HIncrBy(
		ctx,
		key,
		strconv.FormatInt(item.ProductID, 10),
		item.Quantity,
	).Err()

	if err != nil {
		log.Errorf("[CartRepository] AddToCart: %v", err)
		return err
	}

	c.Client.Expire(ctx, key, 24*time.Hour)

	return nil
}

func (c *CartRedisRepository) GetCart(ctx context.Context, userID int64) ([]entity.CartItem, error) {
	key := cartKey(userID)

	result, err := c.Client.HGetAll(ctx, key).Result()
	if err != nil {
		log.Errorf("[CartRepository] GetCart: %v", err)
		return nil, err
	}

	if len(result) == 0 {
		return []entity.CartItem{}, nil
	}

	var items []entity.CartItem

	for productID, qty := range result {
		pid, err := strconv.ParseInt(productID, 10, 64)
		if err != nil {
			continue
		}

		q, err := strconv.ParseInt(qty, 10, 64)
		if err != nil {
			continue
		}

		items = append(items, entity.CartItem{
			ProductID: pid,
			Quantity:  q,
		})
	}

	return items, nil
}

func (c *CartRedisRepository) RemoveFromCart(ctx context.Context, userID, productID int64) error {
	key := cartKey(userID)

	err := c.Client.HDel(
		ctx,
		key,
		strconv.FormatInt(productID, 10),
	).Err()

	if err != nil {
		log.Errorf("[CartRepository] RemoveFromCart: %v", err)
		return err
	}

	return nil
}

func (c *CartRedisRepository) ClearCart(ctx context.Context, userID int64) error {
	key := cartKey(userID)

	err := c.Client.Del(ctx, key).Err()
	if err != nil {
		log.Errorf("[CartRepository] ClearCart: %v", err)
		return err
	}

	return nil
}

func (c *CartRedisRepository) DecreaseItem(ctx context.Context, userID, productID, quantity int64) error {
	key := cartKey(userID)
	field := strconv.FormatInt(productID, 10)

	newQty, err := c.Client.HIncrBy(ctx, key, field, -quantity).Result()
	if err != nil {
		log.Errorf("[CartRepository] RemoveItem : %v", err)
		return err
	}

	if newQty <= 0 {
		return c.Client.HDel(ctx, key, field).Err()
	}

	return nil
}
