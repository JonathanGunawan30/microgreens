package repository

import (
	"context"
	"errors"
	"fmt"
	"order-service/internal/core/domain/entity"
	"order-service/internal/core/domain/model"
	"order-service/utils/constant"
	"order-service/utils/message"
	"time"

	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
)

type OrderRepositoryInterface interface {
	GetAllOrders(ctx context.Context, query entity.QueryStringEntity) ([]entity.OrderEntity, int64, int64, error)
	GetOrderByID(ctx context.Context, orderID int64) (*entity.OrderEntity, error)
	GetOrderByOrderCode(ctx context.Context, code string) (*entity.OrderEntity, error)
	CreateOrder(ctx context.Context, req entity.OrderEntity) (int64, error)
	UpdateStatusOrder(ctx context.Context, req entity.OrderEntity) (int64, string, string, error)
	UpdatePaymentMethod(ctx context.Context, orderID int64, paymentMethod string) error
}

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepositoryInterface {
	return &orderRepository{
		db: db,
	}
}

func (o *orderRepository) GetAllOrders(ctx context.Context, query entity.QueryStringEntity) ([]entity.OrderEntity, int64, int64, error) {
	var modelOrder []model.Order
	var count int64

	q := o.db.WithContext(ctx).Model(&modelOrder)

	if query.Status != "" {
		q.Where("status = ?", query.Status)
	}

	limit := int(query.Limit)
	if limit <= 0 {
		limit = 10
	}

	page := int(query.Page)
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	if query.Search != "" {
		search := "%" + query.Search + "%"
		whereSQL := `(order_code ILIKE ? OR status ILIKE ?)`

		q.Where(whereSQL, search, search)
	}

	if query.BuyerID > 0 {
		q.Where("buyer_id = ?", query.BuyerID)
	}

	if err := q.Count(&count).Error; err != nil {
		log.Errorf("[OrderRepository - 1] GetAllOrders: %v", err)
		return nil, 0, 0, err
	}

	total := (count + int64(limit) - 1) / int64(limit)

	if err := q.Preload("OrderItems").Order("order_date DESC").Limit(limit).Offset(offset).Find(&modelOrder).Error; err != nil {
		log.Errorf("[OrderRepository - 2] GetAllOrders: %v", err)
		return nil, 0, 0, err
	}

	result := make([]entity.OrderEntity, 0, len(modelOrder))

	for _, order := range modelOrder {
		var orderItemEntities []entity.OrderItemEntity
		for _, item := range order.OrderItems {
			orderItemEntities = append(orderItemEntities, entity.OrderItemEntity{
				ID:            item.ID,
				OrderID:       item.OrderID,
				ProductID:     item.ProductID,
				Quantity:      item.Quantity,
				ProductImage:  item.ProductImage,
				ProductUnit:   item.ProductUnit,
				Price:         item.Price,
				ProductName:   item.ProductName,
				ProductWeight: item.ProductWeight,
			})
		}

		result = append(result, entity.OrderEntity{
			ID:            order.ID,
			OrderCode:     order.OrderCode,
			BuyerID:       order.BuyerID,
			OrderDate:     order.OrderDate.Format("2006-01-02"),
			OrderTime:     order.OrderTime,
			Status:        order.Status,
			TotalAmount:   order.TotalAmount,
			PaymentMethod: order.PaymentMethod,
			OrderItems:    orderItemEntities,
			BuyerName:     order.BuyerName,
			BuyerEmail:    order.BuyerEmail,
			BuyerPhone:    order.BuyerPhone,
			BuyerAddress:  order.BuyerAddress,
		})
	}

	return result, count, total, nil
}

func (o *orderRepository) GetOrderByID(ctx context.Context, orderID int64) (*entity.OrderEntity, error) {
	var modelOrder model.Order

	if err := o.db.WithContext(ctx).Preload("OrderItems").Where("id = ?", orderID).First(&modelOrder).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("[OrderRepository - 1] GetOrderByID: %v", err)
			return nil, message.ErrOrderNotFound
		}

		log.Errorf("[OrderRepository - 2] GetOrderByID: %v", err)
		return nil, err
	}

	var orderItemEntities []entity.OrderItemEntity

	for _, item := range modelOrder.OrderItems {
		orderItemEntities = append(orderItemEntities, entity.OrderItemEntity{
			ID:            item.ID,
			OrderID:       modelOrder.ID,
			OrderCode:     modelOrder.OrderCode,
			ProductID:     item.ProductID,
			Quantity:      item.Quantity,
			Price:         item.Price,
			ProductName:   item.ProductName,
			ProductImage:  item.ProductImage,
			ProductUnit:   item.ProductUnit,
			ProductWeight: item.ProductWeight,
		})
	}

	return &entity.OrderEntity{
		ID:            modelOrder.ID,
		OrderCode:     modelOrder.OrderCode,
		BuyerID:       modelOrder.BuyerID,
		OrderDate:     modelOrder.OrderDate.Format("2006-01-02"),
		OrderTime:     modelOrder.OrderTime,
		Status:        modelOrder.Status,
		TotalAmount:   modelOrder.TotalAmount,
		Remarks:       modelOrder.Remarks,
		ShippingFee:   modelOrder.ShippingFee,
		ShippingType:  modelOrder.ShippingType,
		PaymentMethod: modelOrder.PaymentMethod,
		BuyerName:     modelOrder.BuyerName,
		BuyerEmail:    modelOrder.BuyerEmail,
		BuyerPhone:    modelOrder.BuyerPhone,
		BuyerAddress:  modelOrder.BuyerAddress,
		CreatedAt:     modelOrder.CreatedAt,
		OrderItems:    orderItemEntities,
	}, nil
}

func (o *orderRepository) CreateOrder(ctx context.Context, req entity.OrderEntity) (int64, error) {
	orderDate, err := time.Parse("2006-01-02", req.OrderDate)
	if err != nil {
		log.Errorf("[OrderRepository - 1] CreateOrder: %v", err)
		return 0, err
	}

	if _, err := time.Parse("15:04:05", req.OrderTime); err != nil {
		log.Errorf("[OrderRepository - 2] Invalid Time Format: %v", err)
		return 0, err
	}

	var orderItems []model.OrderItem
	for _, item := range req.OrderItems {
		orderItems = append(orderItems, model.OrderItem{
			ProductID:     item.ProductID,
			Quantity:      item.Quantity,
			Price:         item.Price,
			ProductName:   item.ProductName,
			ProductImage:  item.ProductImage,
			ProductUnit:   item.ProductUnit,
			ProductWeight: item.ProductWeight,
		})
	}

	modelOrder := model.Order{
		OrderCode:    req.OrderCode,
		BuyerID:      req.BuyerID,
		OrderDate:    orderDate,
		Status:       req.Status,
		TotalAmount:  req.TotalAmount,
		ShippingType: req.ShippingType,
		ShippingFee:  req.ShippingFee,
		OrderTime:    req.OrderTime,
		Remarks:      req.Remarks,
		BuyerName:    req.BuyerName,
		BuyerEmail:   req.BuyerEmail,
		BuyerAddress: req.BuyerAddress,
		BuyerPhone:   req.BuyerPhone,
		OrderItems:   orderItems,
	}

	if err = o.db.WithContext(ctx).Create(&modelOrder).Error; err != nil {
		log.Errorf("[OrderRepository - 3] CreateOrder: %v", err)
		return 0, err
	}

	return modelOrder.ID, err
}

func (o *orderRepository) UpdateStatusOrder(ctx context.Context, req entity.OrderEntity) (int64, string, string, error) {
	modelOrder := model.Order{}

	if err := o.db.WithContext(ctx).Select("id", "order_code", "status", "buyer_id", "remarks").Where("id = ?", req.ID).First(&modelOrder).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("[OrderRepository - 1] UpdateStatusOrder: %v", err)
			return 0, "", "", message.ErrOrderNotFound
		}
		log.Errorf("[OrderRepository - 2] UpdateStatusOrder: %v", err)
		return 0, "", "", err
	}

	if !constant.IsValidTransition(modelOrder.Status, req.Status) {
		errMessage := fmt.Sprintf("invalid status transition from %s to %s", modelOrder.Status, req.Status)
		log.Infof("[OrderRepository - 3] UpdateStatusOrder: %s", errMessage)
		return 0, "", "", errors.New(errMessage)
	}

	modelOrder.Status = req.Status
	modelOrder.Remarks = req.Remarks

	updateData := map[string]any{
		"status":  req.Status,
		"remarks": req.Remarks,
	}

	if err := o.db.WithContext(ctx).Model(&modelOrder).Updates(updateData).Error; err != nil {
		log.Errorf("[OrderRepository - 4] UpdateStatus: %v", err)
		return 0, "", "", err
	}

	return modelOrder.BuyerID, modelOrder.Status, modelOrder.OrderCode, nil
}

func (o *orderRepository) GetOrderByOrderCode(ctx context.Context, code string) (*entity.OrderEntity, error) {
	var modelOrder model.Order

	if err := o.db.WithContext(ctx).Preload("OrderItems").Where("order_code = ?", code).First(&modelOrder).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorf("[OderRepository - 1] GetOrderByOrderCode: %v", err)
			return nil, message.ErrOrderNotFound
		}

		log.Errorf("[OrderRepository - 2] GetOrderByOrderCode: %v", err)
		return nil, err
	}

	var orderItemEntities []entity.OrderItemEntity

	for _, item := range modelOrder.OrderItems {
		orderItemEntities = append(orderItemEntities, entity.OrderItemEntity{
			ID:            item.ID,
			ProductID:     item.ProductID,
			Quantity:      item.Quantity,
			Price:         item.Price,
			ProductName:   item.ProductName,
			ProductImage:  item.ProductImage,
			ProductUnit:   item.ProductUnit,
			ProductWeight: item.ProductWeight,
		})
	}

	return &entity.OrderEntity{
		ID:            modelOrder.ID,
		OrderCode:     modelOrder.OrderCode,
		BuyerID:       modelOrder.BuyerID,
		OrderDate:     modelOrder.OrderDate.Format("2006-01-02 15:04:05"),
		Status:        modelOrder.Status,
		TotalAmount:   modelOrder.TotalAmount,
		Remarks:       modelOrder.Remarks,
		ShippingFee:   modelOrder.ShippingFee,
		ShippingType:  modelOrder.ShippingType,
		PaymentMethod: modelOrder.PaymentMethod,
		BuyerName:     modelOrder.BuyerName,
		BuyerEmail:    modelOrder.BuyerEmail,
		BuyerPhone:    modelOrder.BuyerPhone,
		BuyerAddress:  modelOrder.BuyerAddress,
		OrderItems:    orderItemEntities,
	}, nil
}

func (o *orderRepository) UpdatePaymentMethod(ctx context.Context, orderID int64, paymentMethod string) error {
	return o.db.WithContext(ctx).Model(model.Order{}).
		Where("id = ?", orderID).
		Updates(map[string]any{
			"payment_method": paymentMethod,
		}).Error
}
