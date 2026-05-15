package service

import (
	"context"
	"fmt"
	"order-service/config"
	"order-service/internal/adapter/message"
	"order-service/internal/adapter/repository"
	"order-service/internal/core/domain/entity"
	"order-service/utils/constant"
	"order-service/utils/generator"
	jwt2 "order-service/utils/jwt"
	msg "order-service/utils/message"

	"github.com/labstack/gommon/log"
)

type OrderServiceInterface interface {
	GetAllOrders(ctx context.Context, queryString entity.QueryStringEntity) ([]entity.OrderEntity, int64, int64, error)
	GetOrderByID(ctx context.Context, orderID int64) (*entity.OrderEntity, error)
	GetCustomerOrderByID(ctx context.Context, orderID int64) (*entity.OrderEntity, error)
	GetOrderByOrderCode(ctx context.Context, code string) (*entity.OrderEntity, error)
	CreateOrder(ctx context.Context, req entity.OrderEntity) (int64, error)
	UpdateStatusOrder(ctx context.Context, req entity.OrderEntity) error
	GetAllCustomerOrders(ctx context.Context, queryString entity.QueryStringEntity, accessToken string) ([]entity.OrderEntity, int64, int64, error)
}

type orderService struct {
	repo             repository.OrderRepositoryInterface
	userSnapshotRepo repository.UserSnapshotRepositoryInterface
	prodSnapshotRepo repository.ProductSnapshotRepositoryInterface
	cfg              *config.Config
	rabbitmq         *config.RabbitMQClient
	elasticRepo      repository.ElasticRepositoryInterface
}

func NewOrderService(repo repository.OrderRepositoryInterface, userSnapshotRepo repository.UserSnapshotRepositoryInterface, prodSnapshotRepository repository.ProductSnapshotRepositoryInterface, cfg *config.Config, rabbitmq *config.RabbitMQClient, elasticRepo repository.ElasticRepositoryInterface) OrderServiceInterface {
	return &orderService{
		repo:             repo,
		userSnapshotRepo: userSnapshotRepo,
		prodSnapshotRepo: prodSnapshotRepository,
		cfg:              cfg,
		rabbitmq:         rabbitmq,
		elasticRepo:      elasticRepo,
	}
}

func (o *orderService) GetAllOrders(ctx context.Context, queryString entity.QueryStringEntity) ([]entity.OrderEntity, int64, int64, error) {
	result, count, total, err := o.elasticRepo.SearchOrderElastic(ctx, queryString)
	if err == nil {
		return result, count, total, nil
	}

	orders, count, total, err := o.repo.GetAllOrders(ctx, queryString)
	if err != nil {
		log.Errorf("[OrderService - 1] GetAllOrders: %v", err)
		return nil, 0, 0, err
	}

	return orders, count, total, nil
}

func (o *orderService) GetOrderByID(ctx context.Context, orderID int64) (*entity.OrderEntity, error) {
	order, err := o.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		log.Errorf("[OrderService - 1] GetOrderByID: %v", err)
		return nil, err
	}

	if len(order.OrderItems) > 0 && order.ProductImage == "" {
		order.ProductImage = order.OrderItems[0].ProductImage
	}

	return order, nil
}

func (o *orderService) GetCustomerOrderByID(ctx context.Context, orderID int64) (*entity.OrderEntity, error) {
	order, err := o.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		log.Errorf("[OrderService - 1] GetCustomerOrderByID: %v", err)
		return nil, err
	}

	if len(order.OrderItems) > 0 && order.ProductImage == "" {
		order.ProductImage = order.OrderItems[0].ProductImage
	}

	return order, nil
}

func (o *orderService) CreateOrder(ctx context.Context, req entity.OrderEntity) (int64, error) {
	req.OrderCode = generator.GenerateOrderCode()
	shippingFee := 0
	if req.ShippingType == "Delivery" {
		shippingFee = 5000
	}
	req.ShippingFee = float64(shippingFee)
	req.Status = "Pending"

	userSnapshot, err := o.userSnapshotRepo.GetByUserID(ctx, req.BuyerID)
	if err != nil {
		log.Errorf("[OrderService] CreateOrder get user snapshot: %v", err)
		return 0, err
	}

	if userSnapshot.Phone == "" {
		return 0, msg.ErrPhoneIsRequired
	}

	if userSnapshot.Address == "" {
		return 0, msg.ErrAddressIsRequired
	}

	req.BuyerName = userSnapshot.Name
	req.BuyerEmail = userSnapshot.Email
	req.BuyerPhone = userSnapshot.Phone
	req.BuyerAddress = userSnapshot.Address

	productIDs := make([]int64, len(req.OrderItems))
	for i, item := range req.OrderItems {
		productIDs[i] = item.ProductID
	}

	snapshots, err := o.prodSnapshotRepo.GetByProductIDs(ctx, productIDs)
	if err != nil {
		log.Errorf("[OrderService] CreateOrder get product snapshots: %v", err)
		return 0, err
	}

	snapshotMap := make(map[int64]*entity.ProductSnapshotEntity)
	for i := range snapshots {
		snapshotMap[snapshots[i].ProductID] = &snapshots[i]
	}

	var subTotal float64
	for i := range req.OrderItems {
		snapshot, ok := snapshotMap[req.OrderItems[i].ProductID]
		if !ok {
			return 0, fmt.Errorf("product %d not found", req.OrderItems[i].ProductID)
		}

		if !snapshot.IsActive {
			return 0, fmt.Errorf("product %d is not available", req.OrderItems[i].ProductID)
		}

		req.OrderItems[i].Price = snapshot.SalePrice
		req.OrderItems[i].ProductName = snapshot.Name
		req.OrderItems[i].ProductImage = snapshot.Image
		req.OrderItems[i].ProductUnit = snapshot.Unit
		req.OrderItems[i].ProductWeight = snapshot.Weight

		subTotal += float64(snapshot.SalePrice) * float64(req.OrderItems[i].Quantity)
	}

	req.TotalAmount = subTotal + float64(shippingFee)

	orderID, err := o.repo.CreateOrder(ctx, req)
	if err != nil {
		log.Errorf("[OrderService - 1] CreateOrder: %v", err)
		return 0, err
	}

	req.ID = orderID

	bgCtx := context.Background()

	go func(bgContext context.Context, orderData entity.OrderEntity) {
		dataOrderByID, err := o.GetOrderByID(bgContext, orderData.ID)
		if err != nil {
			log.Errorf("[BackgroundWorker - GetOrderByID] failed to fill data: %v", err)
			return
		}
		err = message.PublishOrderEvent(o.rabbitmq, *dataOrderByID, o.cfg.ExchangeName.OrderEvent)
		if err != nil {
			log.Errorf("[BackgroundWorker - PublishOrder] failed to send messages: %v", err)
		}
	}(bgCtx, req)

	for _, item := range req.OrderItems {
		go func(prodID int64, qty int64) {
			message.PublishUpdateStock(o.rabbitmq, prodID, qty, o.cfg.PublisherName.ProductUpdateStock)
		}(item.ProductID, int64(item.Quantity))
	}

	return orderID, nil

}

func (o *orderService) UpdateStatusOrder(ctx context.Context, req entity.OrderEntity) error {
	buyerID, statusOrder, orderCode, err := o.repo.UpdateStatusOrder(ctx, req)
	if err != nil {
		log.Errorf("[OrderService - 1] UpdateStatusOrder: %v", err)
		return err
	}

	go func(orderID int64, status string) {
		err := message.PublishUpdateStatus(o.rabbitmq, orderID, status, o.cfg.PublisherName.PublisherUpdateStatus)
		if err != nil {
			log.Errorf("[Background-ES] Failed to publish ES update status: %v", err)
		}
	}(req.ID, statusOrder)

	userSnapshot, err := o.userSnapshotRepo.GetByUserID(ctx, buyerID)
	if err != nil {
		log.Warnf("[OrderService] User snapshot not found for buyerID: %d, skipping notification", buyerID)
		return nil
	}

	if userSnapshot.Email == "" {
		log.Warnf("[OrderService] Email not found for buyerID: %d, skipping notification", buyerID)
		return nil
	}

	emailBody := fmt.Sprintf(
		"Hello,\n\nYour order with ID %s has been updated to status: %s.\n\nThank you for shopping with us!",
		orderCode,
		statusOrder,
	)

	go func() {
		err := message.PublishEmailUpdateStatus(o.rabbitmq, userSnapshot.Email, emailBody, o.cfg.PublisherName.EmailUpdateStatus, buyerID)
		if err != nil {
			log.Errorf("[Background-Email] Failed to publish: %v", err)
		}
	}()

	go func() {
		err := message.PublishSendPushNotifUpdateStatus(o.rabbitmq, emailBody, constant.PUSH_NOTIF, buyerID)
		if err != nil {
			log.Errorf("[Background-PushNotif] Failed to push notif: %v", err)
		}
	}()

	log.Infof("[OrderService] Successfully updated order %s and queued email to %s", orderCode, userSnapshot.Email)
	return nil

}

func (o *orderService) GetAllCustomerOrders(ctx context.Context, queryString entity.QueryStringEntity, accessToken string) ([]entity.OrderEntity, int64, int64, error) {
	userID, err := GetUserIDFromToken(accessToken, o.cfg.App.JwtSecretKey)
	if err != nil {
		log.Errorf("[OrderService - 1] GetAllCustomerOrders: %v", err)
		return nil, 0, 0, err
	}

	queryString.BuyerID = userID

	results, count, total, err := o.repo.GetAllOrders(ctx, queryString)
	if err != nil {
		log.Errorf("[OrderService - 2] GetAllCustomerOrders: %v", err)
		return nil, 0, 0, err
	}

	return results, count, total, nil
}

func (o *orderService) GetOrderByOrderCode(ctx context.Context, code string) (*entity.OrderEntity, error) {
	order, err := o.repo.GetOrderByOrderCode(ctx, code)
	if err != nil {
		log.Errorf("[OrderService - 1] GetOrderByOrderCode: %v", err)
		return nil, err
	}

	if len(order.OrderItems) > 0 && order.ProductImage == "" {
		order.ProductImage = order.OrderItems[0].ProductImage
	}

	return order, nil
}

func GetUserIDFromToken(accessToken, secretKey string) (int64, error) {
	claims, err := jwt2.ValidateToken(accessToken, secretKey)
	if err != nil {
		return 0, err
	}

	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid user_id in token")
	}

	return int64(userIDFloat), nil
}
