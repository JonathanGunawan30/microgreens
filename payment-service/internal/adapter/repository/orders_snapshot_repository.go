package repository

import (
	"context"
	"errors"
	"payment-service/internal/core/domain/entity"
	"payment-service/internal/core/domain/model"
	"payment-service/utils/message"

	"gorm.io/gorm"
)

type OrderSnapshotRepositoryInterface interface {
	Upsert(ctx context.Context, order entity.OrdersSnapshotEntity) error
	GetOrderByID(ctx context.Context, orderID int64) (*entity.OrdersSnapshotEntity, error)
	GetOrderByOrderCode(ctx context.Context, orderCode string) (*entity.OrdersSnapshotEntity, error)
}

type ordersSnapshotRepository struct {
	db *gorm.DB
}

func NewOrdersSnapshotRepository(db *gorm.DB) OrderSnapshotRepositoryInterface {
	return &ordersSnapshotRepository{db: db}
}

func (o *ordersSnapshotRepository) Upsert(ctx context.Context, order entity.OrdersSnapshotEntity) error {
	return o.db.WithContext(ctx).
		Where(model.OrdersSnapshot{OrderID: order.OrderID}).
		Assign(model.OrdersSnapshot{
			OrderCode:    order.OrderCode,
			TotalAmount:  order.TotalAmount,
			ShippingType: order.ShippingType,
			Remarks:      order.Remarks,
			OrderDate:    order.OrderDate,
			OrderTime:    order.OrderTime,
		}).FirstOrCreate(&model.OrdersSnapshot{}).Error
}

func (o *ordersSnapshotRepository) GetOrderByID(ctx context.Context, orderID int64) (*entity.OrdersSnapshotEntity, error) {
	var m model.OrdersSnapshot
	if err := o.db.WithContext(ctx).Where("order_id = ?", orderID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, message.ErrOrderNotFound
		}
		return nil, err
	}

	return &entity.OrdersSnapshotEntity{
		OrderID:      m.OrderID,
		OrderCode:    m.OrderCode,
		TotalAmount:  m.TotalAmount,
		ShippingType: m.ShippingType,
		Remarks:      m.Remarks,
		OrderDate:    m.OrderDate,
		OrderTime:    m.OrderTime,
	}, nil
}

func (o *ordersSnapshotRepository) GetOrderByOrderCode(ctx context.Context, orderCode string) (*entity.OrdersSnapshotEntity, error) {
	var m model.OrdersSnapshot

	if err := o.db.WithContext(ctx).Where("order_code = ?", orderCode).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, message.ErrOrderNotFound
		}
		return nil, err
	}

	return &entity.OrdersSnapshotEntity{
		OrderID:      m.OrderID,
		OrderCode:    m.OrderCode,
		TotalAmount:  m.TotalAmount,
		ShippingType: m.ShippingType,
		Remarks:      m.Remarks,
		OrderDate:    m.OrderDate,
		OrderTime:    m.OrderTime,
	}, nil
}
