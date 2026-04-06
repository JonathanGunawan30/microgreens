package repository

import (
	"context"
	"errors"
	"math"
	"payment-service/internal/core/domain/entity"
	"payment-service/internal/core/domain/model"
	msg "payment-service/utils/message"

	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
)

type PaymentRepositoryInterface interface {
	CreatePayment(ctx context.Context, payment entity.PaymentEntity) error
	LogPayment(ctx context.Context, paymentID int64, status string) error
	UpdateStatusByID(ctx context.Context, orderID int64, status string) error
	GetAllPayment(ctx context.Context, req entity.PaymentQueryStringRequest) ([]entity.PaymentEntity, int64, int64, error)
	GetPaymentDetail(ctx context.Context, paymentID int64, filterUserID int64) (*entity.PaymentEntity, error)
}

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) PaymentRepositoryInterface {
	return &paymentRepository{
		db: db,
	}
}

func (p *paymentRepository) CreatePayment(ctx context.Context, payment entity.PaymentEntity) error {
	modelPayment := model.Payment{
		OrderID:          payment.OrderID,
		UserID:           payment.UserID,
		PaymentMethod:    payment.PaymentMethod,
		PaymentStatus:    payment.PaymentStatus,
		PaymentGatewayID: payment.PaymentGatewayID,
		GrossAmount:      payment.GrossAmount,
		PaymentURL:       payment.PaymentURL,
		OrderCode:        payment.OrderCode,
		ShippingType:     payment.OrderShippingType,
		OrderDate:        payment.OrderDate,
		OrderTime:        payment.OrderTime,
		OrderRemarks:     payment.OrderRemarks,
		CustomerName:     payment.CustomerName,
		CustomerEmail:    payment.CustomerEmail,
		CustomerAddress:  payment.CustomerAddress,
	}

	err := p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&modelPayment).Error; err != nil {
			log.Errorf("[PaymentRepository] CreatePayment: %v", err)
			return err
		}

		logPayment := model.PaymentLog{
			PaymentID: modelPayment.ID,
			Status:    modelPayment.PaymentStatus,
		}

		if err := tx.Create(&logPayment).Error; err != nil {
			log.Errorf("[PaymentRepository] LogPayment inside tx: %v", err)
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	payment.ID = modelPayment.ID

	return nil
}

func (p *paymentRepository) LogPayment(ctx context.Context, paymentID int64, status string) error {
	logPayment := model.PaymentLog{
		PaymentID: paymentID,
		Status:    status,
	}

	if err := p.db.WithContext(ctx).Create(&logPayment).Error; err != nil {
		log.Errorf("[PaymentRepository] LogPayment: %v", err)
		return err
	}

	return nil
}

func (p *paymentRepository) UpdateStatusByID(ctx context.Context, orderID int64, status string) error {
	tx := p.db.WithContext(ctx).Model(&model.Payment{}).Where("order_id = ?", orderID).Update("payment_status", status)

	if tx.Error != nil {
		log.Errorf("[PaymentRepository] UpdateStatusByID: %v", tx.Error)
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		return msg.ErrPaymentNotFound
	}

	return nil
}

func (p *paymentRepository) GetAllPayment(ctx context.Context, req entity.PaymentQueryStringRequest) ([]entity.PaymentEntity, int64, int64, error) {
	var modelPayments []model.Payment
	var countData int64

	offset := (req.Page - 1) * req.Limit

	query := p.db.WithContext(ctx).Model(&model.Payment{})

	if req.Search != "" {
		query = query.Where("order_code ILIKE ?", "%"+req.Search+"%")
	}
	if req.Status != "" {
		query = query.Where("payment_status = ?", req.Status)
	}
	if req.UserID != 0 {
		query = query.Where("user_id = ?", req.UserID)
	}

	if err := query.Count(&countData).Error; err != nil {
		log.Errorf("[PaymentRepository] GetAll Count Error: %v", err)
		return nil, 0, 0, err
	}

	if countData == 0 {
		return []entity.PaymentEntity{}, 0, 0, nil
	}

	totalPage := int64(math.Ceil(float64(countData) / float64(req.Limit)))

	if err := query.Order("created_at DESC").Limit(int(req.Limit)).Offset(int(offset)).Find(&modelPayments).Error; err != nil {
		log.Errorf("[PaymentRepository] GetAll Find Error: %v", err)
		return nil, 0, 0, err
	}

	entities := make([]entity.PaymentEntity, 0, len(modelPayments))

	for _, val := range modelPayments {
		entities = append(entities, entity.PaymentEntity{
			ID:                val.ID,
			OrderID:           val.OrderID,
			UserID:            val.UserID,
			PaymentMethod:     val.PaymentMethod,
			PaymentStatus:     val.PaymentStatus,
			GrossAmount:       val.GrossAmount,
			PaymentGatewayID:  val.PaymentGatewayID,
			PaymentURL:        val.PaymentURL,
			OrderCode:         val.OrderCode,
			OrderShippingType: val.ShippingType,
		})
	}

	return entities, countData, totalPage, nil
}

func (p *paymentRepository) GetPaymentDetail(ctx context.Context, paymentID int64, filterUserID int64) (*entity.PaymentEntity, error) {
	var modelPayment model.Payment

	query := p.db.WithContext(ctx).Model(&model.Payment{}).Where("id = ?", paymentID)

	if filterUserID != 0 {
		query = query.Where("user_id = ?", filterUserID)
	}

	if err := query.First(&modelPayment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, msg.ErrPaymentNotFound
		}

		log.Errorf("[PaymentRepository] GetPaymentDetail DB Error: %v", err)
		return nil, err
	}

	return &entity.PaymentEntity{
		ID:                modelPayment.ID,
		OrderID:           modelPayment.OrderID,
		UserID:            modelPayment.UserID,
		PaymentMethod:     modelPayment.PaymentMethod,
		PaymentStatus:     modelPayment.PaymentStatus,
		GrossAmount:       modelPayment.GrossAmount,
		PaymentGatewayID:  modelPayment.PaymentGatewayID,
		PaymentURL:        modelPayment.PaymentURL,
		PaymentAt:         modelPayment.CreatedAt.Format("2006-01-02 15:04:05"),
		OrderCode:         modelPayment.OrderCode,
		OrderShippingType: modelPayment.ShippingType,
		OrderDate:         modelPayment.OrderDate,
		OrderTime:         modelPayment.OrderTime,
		OrderRemarks:      modelPayment.OrderRemarks,
		CustomerName:      modelPayment.CustomerName,
		CustomerEmail:     modelPayment.CustomerEmail,
		CustomerAddress:   modelPayment.CustomerAddress,
	}, nil
}
