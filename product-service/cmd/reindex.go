package cmd

import (
	"context"
	"product-service/config"
	"product-service/internal/adapter/search"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/domain/model"

	"github.com/labstack/gommon/log"
	"github.com/spf13/cobra"
)

var reindexCmd = &cobra.Command{
	Use:   "reindex",
	Short: "Re-index all products from PostgreSQL to the configured search engine",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("Starting product re-index...")
		cfg := config.NewConfig()

		dbConn, err := cfg.ConnectionPostgres()
		if err != nil {
			log.Fatalf("Failed to connect to Postgres: %v", err)
		}

		searchEngine, err := search.NewSearchEngine(cfg)
		if err != nil {
			log.Fatalf("Failed to initialize search engine: %v", err)
		}

		ctx := context.Background()
		if err := searchEngine.EnsureIndex(ctx); err != nil {
			log.Fatalf("Failed to ensure index on %s: %v", searchEngine.EngineName(), err)
		}

		var products []model.Product
		err = dbConn.DB.WithContext(ctx).Table("products").
			Select("products.*, categories.name as category_name").
			Joins("LEFT JOIN categories ON categories.slug = products.category_slug").
			Where("products.deleted_at IS NULL").
			Order("products.id ASC").
			Find(&products).Error
		if err != nil {
			log.Fatalf("Failed to query products from Postgres: %v", err)
		}

		indexed := 0
		failed := 0
		for _, p := range products {
			item := entity.ProductEntity{
				ID:           p.ID,
				CategorySlug: p.CategorySlug,
				CategoryName: p.CategoryName,
				ParentID:     p.ParentID,
				Name:         p.Name,
				Image:        p.Image,
				Description:  p.Description,
				RegulerPrice: p.RegulerPrice,
				SalePrice:    p.SalePrice,
				Unit:         p.Unit,
				Weight:       p.Weight,
				Stock:        p.Stock,
				Variant:      p.Variant,
				Status:       entity.ProductStatus(p.Status),
				CreatedAt:    p.CreatedAt,
			}

			if err := searchEngine.IndexProduct(ctx, item); err != nil {
				log.Errorf("Failed to index product ID %d: %v", p.ID, err)
				failed++
				continue
			}
			indexed++
		}

		log.Infof("Re-index completed on %s: %d indexed, %d failed (total %d)",
			searchEngine.EngineName(), indexed, failed, len(products))
	},
}

func init() {
	rootCmd.AddCommand(reindexCmd)
}
