package search

import (
	"context"
	"encoding/json"
	"fmt"
	"product-service/config"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/domain/model"
	"strconv"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
	"github.com/labstack/gommon/log"
)

type ElasticsearchEngine struct {
	client *elasticsearch.TypedClient
}

func NewElasticsearchEngine(cfg *config.Config) (*ElasticsearchEngine, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	esClient, err := cfg.NewElasticsearchClient()
	if err != nil {
		return nil, err
	}
	return &ElasticsearchEngine{client: esClient}, nil
}

func (e *ElasticsearchEngine) EngineName() string {
	return "elasticsearch"
}

func (e *ElasticsearchEngine) EnsureIndex(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	exists, err := e.client.Indices.Exists(ProductsIndex).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed checking elasticsearch index: %w", err)
	}
	if exists {
		return nil
	}

	_, err = e.client.Indices.Create(ProductsIndex).
		Raw(strings.NewReader(productsIndexMappingJSON)).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("failed creating elasticsearch index: %w", err)
	}

	return nil
}

func (e *ElasticsearchEngine) IndexProduct(ctx context.Context, product entity.ProductEntity) error {
	if ctx == nil {
		ctx = context.Background()
	}

	_, err := e.client.Index(ProductsIndex).
		Id(strconv.FormatInt(product.ID, 10)).
		Request(product).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("error indexing/updating to elasticsearch: %w", err)
	}
	return nil
}

func (e *ElasticsearchEngine) DeleteProduct(ctx context.Context, productID int64) error {
	if ctx == nil {
		ctx = context.Background()
	}

	_, err := e.client.Delete(ProductsIndex, strconv.FormatInt(productID, 10)).Do(ctx)
	if err != nil {
		return fmt.Errorf("error deleting from elasticsearch: %w", err)
	}
	return nil
}

func (e *ElasticsearchEngine) SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
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

	boolQuery.Filter = append(boolQuery.Filter, types.Query{
		Term: map[string]types.TermQuery{
			"status.keyword": {
				Value: "active",
			},
		},
	})

	boolQuery.MustNot = append(boolQuery.MustNot, types.Query{
		Exists: &types.ExistsQuery{
			Field: "parent_id",
		},
	})

	limit, _, offset, sortField, sortDir := normalizeSearchParams(query)
	esSortOrder := sortorder.Desc
	if sortDir == "asc" {
		esSortOrder = sortorder.Asc
	}

	res, err := e.client.Search().
		Index(ProductsIndex).
		Query(&types.Query{Bool: boolQuery}).
		From(offset).
		Size(limit).
		Sort(&types.SortOptions{
			SortOptions: map[string]types.FieldSort{
				sortField: {
					Order: &esSortOrder,
				},
			},
		}).
		Do(ctx)
	if err != nil {
		log.Errorf("[ElasticsearchEngine] Search Error: %v", err)
		return nil, 0, 0, err
	}

	products := make([]entity.ProductEntity, 0, len(res.Hits.Hits))
	for _, hit := range res.Hits.Hits {
		var m model.Product
		if err := json.Unmarshal(hit.Source_, &m); err == nil {
			products = append(products, modelToProductEntity(m))
		}
	}

	totalHits := res.Hits.Total.Value
	totalPages := (totalHits + int64(limit) - 1) / int64(limit)
	return products, totalHits, totalPages, nil
}

func (e *ElasticsearchEngine) SuggestProducts(ctx context.Context, keyword string, limit int) ([]entity.ProductEntity, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return []entity.ProductEntity{}, nil
	}
	if limit <= 0 {
		limit = 5
	}

	boost := float32(4.0)
	boolQuery := &types.BoolQuery{
		Filter: []types.Query{
			{
				Term: map[string]types.TermQuery{
					"status.keyword": {
						Value: "active",
					},
				},
			},
		},
		MustNot: []types.Query{
			{
				Exists: &types.ExistsQuery{
					Field: "parent_id",
				},
			},
		},
		Should: []types.Query{
			{
				MatchPhrasePrefix: map[string]types.MatchPhrasePrefixQuery{
					"name": {
						Query: keyword,
						Boost: &boost,
					},
				},
			},
			{
				MultiMatch: &types.MultiMatchQuery{
					Query:     keyword,
					Fields:    []string{"name^3", "name.autocomplete^3", "category_name^2", "category_slug"},
					Fuzziness: "AUTO",
				},
			},
		},
		MinimumShouldMatch: 1,
	}

	res, err := e.client.Search().
		Index(ProductsIndex).
		Query(&types.Query{Bool: boolQuery}).
		Size(limit).
		Do(ctx)
	if err != nil {
		log.Errorf("[ElasticsearchEngine] Suggest Error: %v", err)
		return nil, err
	}

	suggestions := make([]entity.ProductEntity, 0, len(res.Hits.Hits))
	for _, hit := range res.Hits.Hits {
		var m model.Product
		if err := json.Unmarshal(hit.Source_, &m); err == nil {
			suggestions = append(suggestions, modelToProductEntity(m))
		}
	}

	return suggestions, nil
}
