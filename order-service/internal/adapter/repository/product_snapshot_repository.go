package repository

import (
	"context"
	"errors"
	"order-service/internal/core/domain/entity"
	"order-service/internal/core/domain/model"
	"order-service/utils/message"

	"gorm.io/gorm"
)

type ProductSnapshotRepositoryInterface interface {
	Upsert(ctx context.Context, product entity.ProductSnapshotEntity) error
	GetByProductID(ctx context.Context, productID int64) (*entity.ProductSnapshotEntity, error)
	SetInactive(ctx context.Context, productID int64) error
	GetByProductIDs(ctx context.Context, productIDs []int64) ([]entity.ProductSnapshotEntity, error)
}

type productSnapshotRepository struct {
	db *gorm.DB
}

func NewProductSnapshotRepository(db *gorm.DB) ProductSnapshotRepositoryInterface {
	return &productSnapshotRepository{db: db}
}

func (p *productSnapshotRepository) Upsert(ctx context.Context, product entity.ProductSnapshotEntity) error {
	return p.db.WithContext(ctx).
		Where(model.ProductSnapshot{ProductID: product.ProductID}).
		Assign(model.ProductSnapshot{
			Name:      product.Name,
			Image:     product.Image,
			SalePrice: product.SalePrice,
			Unit:      product.Unit,
			Weight:    product.Weight,
			IsActive:  product.IsActive,
		}).
		FirstOrCreate(&model.ProductSnapshot{}).Error
}

func (p *productSnapshotRepository) GetByProductID(ctx context.Context, productID int64) (*entity.ProductSnapshotEntity, error) {
	var m model.ProductSnapshot

	if err := p.db.WithContext(ctx).Where("product_id = ?", productID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, message.ErrProductNotFound
		}
		return nil, err
	}

	return &entity.ProductSnapshotEntity{
		ProductID: m.ProductID,
		Name:      m.Name,
		Image:     m.Image,
		SalePrice: m.SalePrice,
		Unit:      m.Unit,
		Weight:    m.Weight,
		IsActive:  m.IsActive,
	}, nil
}

func (p *productSnapshotRepository) SetInactive(ctx context.Context, productID int64) error {
	result := p.db.WithContext(ctx).Model(&model.ProductSnapshot{}).Where("product_id = ?", productID).Update("is_active", false)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return message.ErrProductNotFound
	}

	return nil
}

func (p *productSnapshotRepository) GetByProductIDs(ctx context.Context, productIDs []int64) ([]entity.ProductSnapshotEntity, error) {
	var models []model.ProductSnapshot

	if err := p.db.WithContext(ctx).
		Where("product_id IN ? AND is_active = ?", productIDs, true).
		Find(&models).Error; err != nil {
		return nil, err
	}

	result := make([]entity.ProductSnapshotEntity, 0, len(models))
	for _, m := range models {
		result = append(result, entity.ProductSnapshotEntity{
			ProductID: m.ProductID,
			Name:      m.Name,
			Image:     m.Image,
			SalePrice: m.SalePrice,
			Unit:      m.Unit,
			Weight:    m.Weight,
			IsActive:  m.IsActive,
		})
	}

	return result, nil
}
