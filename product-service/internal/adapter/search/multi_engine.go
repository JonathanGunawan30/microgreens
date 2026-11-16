package search

import (
	"context"
	"fmt"
	"product-service/internal/core/domain/entity"

	"github.com/labstack/gommon/log"
)

type MultiSearchEngine struct {
	primary   SearchEngineInterface
	secondary SearchEngineInterface
}

func NewMultiSearchEngine(primary, secondary SearchEngineInterface) SearchEngineInterface {
	return &MultiSearchEngine{
		primary:   primary,
		secondary: secondary,
	}
}

func (m *MultiSearchEngine) EngineName() string {
	return fmt.Sprintf("%s+%s", m.primary.EngineName(), m.secondary.EngineName())
}

func (m *MultiSearchEngine) EnsureIndex(ctx context.Context) error {
	err1 := m.primary.EnsureIndex(ctx)
	if err1 != nil {
		log.Warnf("[MultiSearchEngine] Primary (%s) EnsureIndex failed: %v", m.primary.EngineName(), err1)
	}
	err2 := m.secondary.EnsureIndex(ctx)
	if err2 != nil {
		log.Warnf("[MultiSearchEngine] Secondary (%s) EnsureIndex failed: %v", m.secondary.EngineName(), err2)
	}
	if err1 != nil && err2 != nil {
		return fmt.Errorf("primary error: %v; secondary error: %v", err1, err2)
	}
	return nil
}

func (m *MultiSearchEngine) IndexProduct(ctx context.Context, product entity.ProductEntity) error {
	err1 := m.primary.IndexProduct(ctx, product)
	if err1 != nil {
		log.Warnf("[MultiSearchEngine] Primary (%s) IndexProduct failed: %v", m.primary.EngineName(), err1)
	}
	err2 := m.secondary.IndexProduct(ctx, product)
	if err2 != nil {
		log.Warnf("[MultiSearchEngine] Secondary (%s) IndexProduct failed: %v", m.secondary.EngineName(), err2)
	}
	if err1 != nil && err2 != nil {
		return fmt.Errorf("primary error: %v; secondary error: %v", err1, err2)
	}
	return nil
}

func (m *MultiSearchEngine) DeleteProduct(ctx context.Context, productID int64) error {
	err1 := m.primary.DeleteProduct(ctx, productID)
	if err1 != nil {
		log.Warnf("[MultiSearchEngine] Primary (%s) DeleteProduct failed: %v", m.primary.EngineName(), err1)
	}
	err2 := m.secondary.DeleteProduct(ctx, productID)
	if err2 != nil {
		log.Warnf("[MultiSearchEngine] Secondary (%s) DeleteProduct failed: %v", m.secondary.EngineName(), err2)
	}
	if err1 != nil && err2 != nil {
		return fmt.Errorf("primary error: %v; secondary error: %v", err1, err2)
	}
	return nil
}

func (m *MultiSearchEngine) SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	products, total, pages, err := m.primary.SearchProducts(ctx, query)
	if err == nil {
		return products, total, pages, nil
	}

	log.Warnf("[MultiSearchEngine] Primary (%s) SearchProducts failed (%v), falling back to secondary (%s)", m.primary.EngineName(), err, m.secondary.EngineName())
	return m.secondary.SearchProducts(ctx, query)
}

func (m *MultiSearchEngine) SuggestProducts(ctx context.Context, keyword string, limit int) ([]entity.ProductEntity, error) {
	suggestions, err := m.primary.SuggestProducts(ctx, keyword, limit)
	if err == nil {
		return suggestions, nil
	}

	log.Warnf("[MultiSearchEngine] Primary (%s) SuggestProducts failed (%v), falling back to secondary (%s)", m.primary.EngineName(), err, m.secondary.EngineName())
	return m.secondary.SuggestProducts(ctx, keyword, limit)
}
