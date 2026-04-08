package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"order-service/internal/core/domain/entity"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
	"github.com/labstack/gommon/log"
)

type ElasticRepositoryInterface interface {
	SearchOrderElastic(ctx context.Context, q entity.QueryStringEntity) ([]entity.OrderEntity, int64, int64, error)
}

type elasticRepository struct {
	esClient *elasticsearch.TypedClient
}

func NewElasticRepository(es *elasticsearch.TypedClient) ElasticRepositoryInterface {
	return &elasticRepository{
		esClient: es,
	}
}

func (e *elasticRepository) SearchOrderElastic(ctx context.Context, q entity.QueryStringEntity) ([]entity.OrderEntity, int64, int64, error) {
	if e.esClient == nil {
		return nil, 0, 0, errors.New("elasticsearch not available")
	}

	fromInt := int((q.Page - 1) * q.Limit)
	limitInt := int(q.Limit)

	var mustClauses []types.Query

	if q.Status != "" {
		mustClauses = append(mustClauses, types.Query{
			Match: map[string]types.MatchQuery{
				"status": {Query: q.Status},
			},
		})
	}

	if q.Search != "" {
		mustClauses = append(mustClauses, types.Query{
			MultiMatch: &types.MultiMatchQuery{
				Query:  q.Search,
				Fields: []string{"order_code", "status", "buyer_name"},
			},
		})
	} else {
		mustClauses = append(mustClauses, types.Query{
			MatchAll: types.NewMatchAllQuery(),
		})
	}

	req := &search.Request{
		From: &fromInt,
		Size: &limitInt,
		Query: &types.Query{
			Bool: &types.BoolQuery{
				Must: mustClauses,
			},
		},
		Sort: []types.SortCombinations{
			types.SortOptions{
				SortOptions: map[string]types.FieldSort{
					"id": {Order: &sortorder.Asc},
				},
			},
		},
	}

	res, err := e.esClient.Search().
		Index("orders").
		Request(req).
		Do(ctx)

	if err != nil {
		return nil, 0, 0, fmt.Errorf("[SearchOrderElastic] failed to execute search: %w", err)
	}

	var orders []entity.OrderEntity

	totalData := 0
	if res.Hits.Total != nil {
		totalData = int(res.Hits.Total.Value)
	}

	totalPage := 0
	if q.Limit > 0 {
		totalPage = int(math.Ceil(float64(totalData) / float64(q.Limit)))
	}

	for _, hit := range res.Hits.Hits {
		var order entity.OrderEntity

		if err := json.Unmarshal(hit.Source_, &order); err != nil {
			log.Printf("[SearchOrderElastic] Warning: failed to unmarshal hit data (ID: %s): %v", *hit.Id_, err)
			continue
		}

		orders = append(orders, order)
	}

	return orders, int64(totalData), int64(totalPage), nil
}
