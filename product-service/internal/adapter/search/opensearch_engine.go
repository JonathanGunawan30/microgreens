package search

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"product-service/config"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/domain/model"
	"strconv"
	"strings"
	"time"
)

type OpenSearchEngine struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

func NewOpenSearchEngine(cfg *config.Config) (*OpenSearchEngine, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	host := strings.TrimSpace(cfg.OpenSearch.Host)
	username := cfg.OpenSearch.Username
	password := cfg.OpenSearch.Password

	if host == "" {
		host = strings.TrimSpace(cfg.Elasticsearch.Host)
		username = cfg.Elasticsearch.Username
		password = cfg.Elasticsearch.Password
	}

	if host == "" {
		return nil, fmt.Errorf("opensearch host is empty")
	}

	engine := &OpenSearchEngine{
		baseURL:  strings.TrimRight(host, "/"),
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}

	if err := engine.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("error connecting to opensearch: %w", err)
	}

	return engine, nil
}

func (o *OpenSearchEngine) EngineName() string {
	return "opensearch"
}

func (o *OpenSearchEngine) Ping(ctx context.Context) error {
	status, body, err := o.doRequest(ctx, http.MethodGet, "/", nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("opensearch ping failed (%d): %s", status, string(body))
	}
	return nil
}

func (o *OpenSearchEngine) EnsureIndex(ctx context.Context) error {
	status, body, err := o.doRequest(ctx, http.MethodGet, "/"+ProductsIndex, nil)
	if err != nil {
		return err
	}
	if status == http.StatusOK {
		return nil
	}
	if status != http.StatusNotFound {
		return fmt.Errorf("unexpected status checking opensearch index (%d): %s", status, string(body))
	}

	status, body, err = o.doRequest(ctx, http.MethodPut, "/"+ProductsIndex, []byte(productsIndexMappingJSON))
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("failed to create opensearch index (%d): %s", status, string(body))
	}
	return nil
}

func (o *OpenSearchEngine) IndexProduct(ctx context.Context, product entity.ProductEntity) error {
	payload, err := json.Marshal(product)
	if err != nil {
		return fmt.Errorf("failed to marshal product %d: %w", product.ID, err)
	}

	path := fmt.Sprintf("/%s/_doc/%s", ProductsIndex, strconv.FormatInt(product.ID, 10))
	status, body, err := o.doRequest(ctx, http.MethodPut, path, payload)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("opensearch index error (%d): %s", status, string(body))
	}
	return nil
}

func (o *OpenSearchEngine) DeleteProduct(ctx context.Context, productID int64) error {
	path := fmt.Sprintf("/%s/_doc/%s", ProductsIndex, strconv.FormatInt(productID, 10))
	status, body, err := o.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return nil
	}
	if status >= 400 {
		return fmt.Errorf("opensearch delete error (%d): %s", status, string(body))
	}
	return nil
}

type openSearchResponse struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source json.RawMessage `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func (o *OpenSearchEngine) SearchProducts(ctx context.Context, query entity.QueryStringProduct) ([]entity.ProductEntity, int64, int64, error) {
	filters := []map[string]any{
		{
			"term": map[string]any{
				"status.keyword": "active",
			},
		},
	}

	if query.CategorySlug != "" {
		filters = append(filters, map[string]any{
			"term": map[string]any{
				"category_slug.keyword": query.CategorySlug,
			},
		})
	}

	if query.StartPrice > 0 || query.EndPrice > 0 {
		salePriceRange := map[string]any{}
		if query.StartPrice > 0 {
			salePriceRange["gte"] = query.StartPrice
		}
		if query.EndPrice > 0 {
			salePriceRange["lte"] = query.EndPrice
		}
		filters = append(filters, map[string]any{
			"range": map[string]any{
				"sale_price": salePriceRange,
			},
		})
	}

	boolClause := map[string]any{
		"filter": filters,
		"must_not": []map[string]any{
			{
				"exists": map[string]any{
					"field": "parent_id",
				},
			},
		},
	}

	if query.Search != "" {
		boolClause["must"] = []map[string]any{
			{
				"multi_match": map[string]any{
					"query":     query.Search,
					"fields":    []string{"name^3", "category_slug", "description"},
					"fuzziness": "AUTO",
				},
			},
		}
	}

	limit, _, offset, sortField, sortOrder := normalizeSearchParams(query)

	reqBody := map[string]any{
		"query": map[string]any{
			"bool": boolClause,
		},
		"from": offset,
		"size": limit,
		"sort": []map[string]any{
			{
				sortField: map[string]any{
					"order": sortOrder,
				},
			},
		},
	}

	searchRes, err := o.executeSearch(ctx, reqBody)
	if err != nil {
		return nil, 0, 0, err
	}

	products := make([]entity.ProductEntity, 0, len(searchRes.Hits.Hits))
	for _, hit := range searchRes.Hits.Hits {
		var m model.Product
		if err := json.Unmarshal(hit.Source, &m); err == nil {
			products = append(products, modelToProductEntity(m))
		}
	}

	totalHits := searchRes.Hits.Total.Value
	totalPages := (totalHits + int64(limit) - 1) / int64(limit)
	return products, totalHits, totalPages, nil
}

func (o *OpenSearchEngine) SuggestProducts(ctx context.Context, keyword string, limit int) ([]entity.ProductEntity, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return []entity.ProductEntity{}, nil
	}
	if limit <= 0 {
		limit = 5
	}

	reqBody := map[string]any{
		"size": limit,
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []map[string]any{
					{
						"term": map[string]any{
							"status.keyword": "active",
						},
					},
				},
				"must_not": []map[string]any{
					{
						"exists": map[string]any{
							"field": "parent_id",
						},
					},
				},
				"should": []map[string]any{
					{
						"match_phrase_prefix": map[string]any{
							"name": map[string]any{
								"query": keyword,
								"boost": 4,
							},
						},
					},
					{
						"multi_match": map[string]any{
							"query":     keyword,
							"fields":    []string{"name^3", "name.autocomplete^3", "category_name^2", "category_slug"},
							"fuzziness": "AUTO",
						},
					},
				},
				"minimum_should_match": 1,
			},
		},
	}

	searchRes, err := o.executeSearch(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	suggestions := make([]entity.ProductEntity, 0, len(searchRes.Hits.Hits))
	for _, hit := range searchRes.Hits.Hits {
		var m model.Product
		if err := json.Unmarshal(hit.Source, &m); err == nil {
			suggestions = append(suggestions, modelToProductEntity(m))
		}
	}

	return suggestions, nil
}

func (o *OpenSearchEngine) executeSearch(ctx context.Context, reqBody map[string]any) (*openSearchResponse, error) {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal opensearch query: %w", err)
	}

	status, body, err := o.doRequest(ctx, http.MethodPost, "/"+ProductsIndex+"/_search", payload)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("opensearch search error (%d): %s", status, string(body))
	}

	var searchRes openSearchResponse
	if err := json.Unmarshal(body, &searchRes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal opensearch response: %w", err)
	}
	return &searchRes, nil
}

func (o *OpenSearchEngine) doRequest(ctx context.Context, method, path string, payload []byte) (int, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var bodyReader io.Reader
	if payload != nil {
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, o.baseURL+path, bodyReader)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if o.username != "" {
		req.SetBasicAuth(o.username, o.password)
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}

	return resp.StatusCode, respBody, nil
}
