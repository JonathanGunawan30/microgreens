package search

import (
	"context"
	"fmt"
	"product-service/config"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/domain/model"
	"strings"

	"github.com/labstack/gommon/log"
)

const ProductsIndex = "products"

type SearchEngineInterface interface {
	EngineName() string
	EnsureIndex(ctx context.Context) error
	IndexProduct(ctx context.Context, product entity.ProductEntity) error
	DeleteProduct(ctx context.Context, productID int64) error
	SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error)
	SuggestProducts(ctx context.Context, keyword string, limit int) ([]entity.ProductEntity, error)
}

func NewSearchEngine(cfg *config.Config) (SearchEngineInterface, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	mode := strings.ToLower(strings.TrimSpace(cfg.SearchEngine))
	if mode == "" {
		if cfg.Elasticsearch.Host == "" && cfg.OpenSearch.Host != "" {
			mode = "opensearch"
		} else {
			mode = "elasticsearch"
		}
	}

	ctx := context.Background()

	switch mode {
	case "none", "off", "postgres":
		return nil, fmt.Errorf("search engine disabled via SEARCH_ENGINE=%s", mode)

	case "opensearch":
		osEngine, err := NewOpenSearchEngine(cfg)
		if err != nil {
			return nil, err
		}
		if err := osEngine.EnsureIndex(ctx); err != nil {
			log.Warnf("[SearchEngine] Failed to ensure OpenSearch index: %v", err)
		}
		return osEngine, nil

	case "elasticsearch":
		esEngine, err := NewElasticsearchEngine(cfg)
		if err == nil {
			if ensureErr := esEngine.EnsureIndex(ctx); ensureErr != nil {
				log.Warnf("[SearchEngine] Failed to ensure Elasticsearch index: %v", ensureErr)
			}
			return esEngine, nil
		}

		if cfg.OpenSearch.Host != "" {
			log.Warnf("[SearchEngine] Elasticsearch unavailable (%v), falling back to OpenSearch", err)
			osEngine, osErr := NewOpenSearchEngine(cfg)
			if osErr == nil {
				if ensureErr := osEngine.EnsureIndex(ctx); ensureErr != nil {
					log.Warnf("[SearchEngine] Failed to ensure OpenSearch index: %v", ensureErr)
				}
				return osEngine, nil
			}
			return nil, fmt.Errorf("elasticsearch error (%v) and opensearch fallback error (%v)", err, osErr)
		}

		return nil, err

	case "auto", "both":
		var esEngine SearchEngineInterface
		var osEngine SearchEngineInterface
		var esErr, osErr error

		if cfg.Elasticsearch.Host != "" {
			esEngine, esErr = NewElasticsearchEngine(cfg)
			if esErr != nil {
				log.Warnf("[SearchEngine] Auto mode: Elasticsearch unavailable: %v", esErr)
				esEngine = nil
			} else if ensureErr := esEngine.EnsureIndex(ctx); ensureErr != nil {
				log.Warnf("[SearchEngine] Auto mode: Failed to ensure Elasticsearch index: %v", ensureErr)
			}
		}

		if cfg.OpenSearch.Host != "" {
			osEngine, osErr = NewOpenSearchEngine(cfg)
			if osErr != nil {
				log.Warnf("[SearchEngine] Auto mode: OpenSearch unavailable: %v", osErr)
				osEngine = nil
			} else if ensureErr := osEngine.EnsureIndex(ctx); ensureErr != nil {
				log.Warnf("[SearchEngine] Auto mode: Failed to ensure OpenSearch index: %v", ensureErr)
			}
		}

		if esEngine != nil && osEngine != nil {
			return NewMultiSearchEngine(esEngine, osEngine), nil
		}
		if esEngine != nil {
			return esEngine, nil
		}
		if osEngine != nil {
			return osEngine, nil
		}

		return nil, fmt.Errorf("no search engine available in auto mode (es=%v, os=%v)", esErr, osErr)

	default:
		return nil, fmt.Errorf("unsupported SEARCH_ENGINE value: %s", cfg.SearchEngine)
	}
}

func normalizeSearchParams(query entity.QueryStringProduct) (limit, page, offset int, sortField, sortOrder string) {
	limit = int(query.Limit)
	if limit <= 0 {
		limit = 10
	}

	page = int(query.Page)
	if page <= 0 {
		page = 1
	}

	offset = (page - 1) * limit

	sortField = "created_at"
	if query.OrderBy != "" {
		allowedSort := map[string]string{
			"name":  "name.keyword",
			"price": "sale_price",
			"stock": "stock",
			"id":    "id",
		}
		if val, ok := allowedSort[query.OrderBy]; ok {
			sortField = val
		}
	}

	sortOrder = "desc"
	if strings.EqualFold(query.OrderType, "asc") {
		sortOrder = "asc"
	}

	return limit, page, offset, sortField, sortOrder
}

func modelToProductEntity(m model.Product) entity.ProductEntity {
	return entity.ProductEntity{
		ID:           m.ID,
		CategorySlug: m.CategorySlug,
		CategoryName: m.CategoryName,
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
	}
}

const productsIndexMappingJSON = `{
	"settings": {
		"analysis": {
			"tokenizer": {
				"autocomplete_tokenizer": {
					"type": "edge_ngram",
					"min_gram": 2,
					"max_gram": 20,
					"token_chars": ["letter", "digit"]
				}
			},
			"analyzer": {
				"autocomplete_analyzer": {
					"type": "custom",
					"tokenizer": "autocomplete_tokenizer",
					"filter": ["lowercase"]
				}
			}
		}
	},
	"mappings": {
		"properties": {
			"id": { "type": "long" },
			"category_slug": {
				"type": "text",
				"fields": { "keyword": { "type": "keyword" } }
			},
			"category_name": {
				"type": "text",
				"fields": { "keyword": { "type": "keyword" } }
			},
			"parent_id": { "type": "long" },
			"name": {
				"type": "text",
				"fields": {
					"keyword": { "type": "keyword" },
					"autocomplete": {
						"type": "text",
						"analyzer": "autocomplete_analyzer",
						"search_analyzer": "standard"
					}
				}
			},
			"image": { "type": "keyword" },
			"description": { "type": "text" },
			"reguler_price": { "type": "long" },
			"sale_price": { "type": "long" },
			"unit": { "type": "keyword" },
			"weight": { "type": "long" },
			"stock": { "type": "long" },
			"variant": { "type": "integer" },
			"status": {
				"type": "text",
				"fields": { "keyword": { "type": "keyword" } }
			},
			"created_at": { "type": "date" }
		}
	}
}`
