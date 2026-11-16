package search

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"product-service/config"
	"product-service/internal/core/domain/entity"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenSearchEngine(t *testing.T) {
	ctx := context.Background()

	var indexCreated bool
	var lastIndexedID string
	var lastIndexedName string
	var lastDeletedID string
	var lastSearchBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"version":{"distribution":"opensearch"}}`))

		case r.Method == http.MethodGet && r.URL.Path == "/products":
			if !indexCreated {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)

		case r.Method == http.MethodPut && r.URL.Path == "/products":
			bodyBytes, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			assert.Contains(t, string(bodyBytes), "edge_ngram")
			indexCreated = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"acknowledged":true}`))

		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/products/_doc/"):
			lastIndexedID = strings.TrimPrefix(r.URL.Path, "/products/_doc/")
			var doc entity.ProductEntity
			require.NoError(t, json.NewDecoder(r.Body).Decode(&doc))
			lastIndexedName = doc.Name
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"result":"created"}`))

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/products/_doc/"):
			lastDeletedID = strings.TrimPrefix(r.URL.Path, "/products/_doc/")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":"deleted"}`))

		case r.Method == http.MethodPost && r.URL.Path == "/products/_search":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&lastSearchBody))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"hits": {
					"total": {"value": 1},
					"hits": [
						{
							"_source": {
								"id": 42,
								"category_slug": "sayur-daun",
								"category_name": "Sayur Daun",
								"name": "Bayam Merah",
								"image": "https://img.example/bayam.png",
								"description": "Bayam merah segar",
								"reguler_price": 7000,
								"sale_price": 5500,
								"unit": "ikat",
								"weight": 250,
								"stock": 15,
								"variant": 1,
								"status": "active"
							}
						}
					]
				}
			}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		OpenSearch: config.OpenSearch{
			Host: server.URL,
		},
	}

	engine, err := NewOpenSearchEngine(cfg)
	require.NoError(t, err)
	assert.Equal(t, "opensearch", engine.EngineName())

	t.Run("EnsureIndex creates mapping when missing", func(t *testing.T) {
		err := engine.EnsureIndex(ctx)
		require.NoError(t, err)
		assert.True(t, indexCreated)

		err = engine.EnsureIndex(ctx)
		require.NoError(t, err)
	})

	t.Run("IndexProduct stores product document", func(t *testing.T) {
		err := engine.IndexProduct(ctx, entity.ProductEntity{
			ID:           42,
			Name:         "Bayam Merah",
			CategorySlug: "sayur-daun",
			Status:       entity.ProductStatus("active"),
		})
		require.NoError(t, err)
		assert.Equal(t, "42", lastIndexedID)
		assert.Equal(t, "Bayam Merah", lastIndexedName)
	})

	t.Run("SearchProducts returns mapped entities and pagination", func(t *testing.T) {
		products, total, pages, err := engine.SearchProducts(ctx, entity.QueryStringProduct{
			Search:       "bayam",
			CategorySlug: "sayur-daun",
			StartPrice:   1000,
			EndPrice:     10000,
			Page:         1,
			Limit:        10,
			OrderBy:      "price",
			OrderType:    "asc",
		})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, int64(1), pages)
		require.Len(t, products, 1)
		assert.Equal(t, int64(42), products[0].ID)
		assert.Equal(t, "Bayam Merah", products[0].Name)
		assert.Equal(t, "sayur-daun", products[0].CategorySlug)
		assert.Equal(t, "Sayur Daun", products[0].CategoryName)
		assert.Equal(t, float64(10), lastSearchBody["size"])
	})

	t.Run("SuggestProducts sends match_phrase_prefix and fuzzy multi_match", func(t *testing.T) {
		suggestions, err := engine.SuggestProducts(ctx, "bay", 5)
		require.NoError(t, err)
		require.Len(t, suggestions, 1)
		assert.Equal(t, "Bayam Merah", suggestions[0].Name)
		assert.Equal(t, float64(5), lastSearchBody["size"])

		queryJSON, err := json.Marshal(lastSearchBody["query"])
		require.NoError(t, err)
		assert.Contains(t, string(queryJSON), "match_phrase_prefix")
		assert.Contains(t, string(queryJSON), "name.autocomplete^3")
	})

	t.Run("DeleteProduct deletes document by ID", func(t *testing.T) {
		err := engine.DeleteProduct(ctx, 42)
		require.NoError(t, err)
		assert.Equal(t, "42", lastDeletedID)
	})
}
