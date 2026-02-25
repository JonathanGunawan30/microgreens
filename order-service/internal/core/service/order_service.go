package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"order-service/config"
	"order-service/internal/adapter"
	"order-service/internal/adapter/handler/response"
	"order-service/internal/adapter/message"
	"order-service/internal/adapter/repository"
	"order-service/internal/core/domain/entity"
	"order-service/utils/generator"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/gommon/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

type OrderServiceInterface interface {
	GetAllOrders(ctx context.Context, queryString entity.QueryStringEntity, accessToken string) ([]entity.OrderEntity, int64, int64, error)
	GetOrderByID(ctx context.Context, orderID int64, accessToken string) (*entity.OrderEntity, error)
	GetCustomerOrderByID(ctx context.Context, orderID int64, accessToken string) (*entity.OrderEntity, error)
	GetOrderByOrderCode(ctx context.Context, code string, accessToken string) (*entity.OrderEntity, error)
	CreateOrder(ctx context.Context, req entity.OrderEntity, accessToken string) (int64, error)
	UpdateStatusOrder(ctx context.Context, req entity.OrderEntity, accessToken string) error
	GetAllCustomerOrders(ctx context.Context, queryString entity.QueryStringEntity, accessToken string) ([]entity.OrderEntity, int64, int64, error)
}

type orderService struct {
	repo        repository.OrderRepositoryInterface
	cfg         *config.Config
	httpClient  adapter.HttpClient
	rabbitmq    *amqp.Connection
	elasticRepo repository.ElasticRepositoryInterface
}

func NewOrderService(repo repository.OrderRepositoryInterface, cfg *config.Config, client adapter.HttpClient, rabbitmq *amqp.Connection, elasticRepo repository.ElasticRepositoryInterface) OrderServiceInterface {
	return &orderService{
		repo:        repo,
		cfg:         cfg,
		httpClient:  client,
		rabbitmq:    rabbitmq,
		elasticRepo: elasticRepo,
	}
}

func (o *orderService) GetAllOrders(ctx context.Context, queryString entity.QueryStringEntity, accessToken string) ([]entity.OrderEntity, int64, int64, error) {
	result, count, total, err := o.elasticRepo.SearchOrderElastic(ctx, queryString)
	if err == nil {
		return result, count, total, nil
	}

	orders, count, total, err := o.repo.GetAllOrders(ctx, queryString)
	if err != nil {
		log.Errorf("[OrderService - 1] GetAllOrders: %v", err)
		return nil, 0, 0, err
	}

	for i := range orders {
		userResponse, err := o.httpClientUserService(orders[i].BuyerID, accessToken, false)
		if err == nil && userResponse != nil {
			orders[i].BuyerName = userResponse.Data.Name
		}

		for j := range orders[i].OrderItems {
			productID := orders[i].OrderItems[j].ProductID

			productResponse, err := o.httpClientProductService(productID, accessToken, false)
			if err == nil && productResponse != nil {
				orders[i].OrderItems[j].ProductImage = productResponse.Data.Image
			}
		}
	}

	return orders, count, total, nil
}

func (o *orderService) GetOrderByID(ctx context.Context, orderID int64, accessToken string) (*entity.OrderEntity, error) {
	order, err := o.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		log.Errorf("[OrderService - 1] GetOrderByID: %v", err)
		return nil, err
	}

	userRole := GetRoleFromToken(accessToken)
	isCustomer := userRole == "Customer"

	userResponse, err := o.httpClientUserService(order.BuyerID, accessToken, isCustomer)
	if err == nil && userResponse != nil && userResponse.Data.ID != 0 {
		order.BuyerName = userResponse.Data.Name
		order.BuyerEmail = userResponse.Data.Email
		order.BuyerPhone = userResponse.Data.Phone
		order.BuyerAddress = userResponse.Data.Address
	} else {
		log.Warnf("[OrderService - 2] GetOrderByID: Failed to fetch user profile for BuyerID %d", order.BuyerID)
	}

	for i := range order.OrderItems {
		productID := order.OrderItems[i].ProductID

		productResponse, err := o.httpClientProductService(productID, accessToken, isCustomer)
		if err == nil && productResponse != nil {
			order.OrderItems[i].ProductImage = productResponse.Data.Image
			order.OrderItems[i].ProductName = productResponse.Data.Name

			if productResponse.Data.SalePrice > 0 {
				order.OrderItems[i].Price = int64(productResponse.Data.SalePrice)
			}

			if order.ProductImage == "" {
				order.ProductImage = productResponse.Data.Image
			}
		} else {
			log.Warnf("[OrderService - 3] GetOrderByID: Failed to fetch product data for ProductID %d", productID)
		}
	}

	return order, nil
}

func (o *orderService) GetCustomerOrderByID(ctx context.Context, orderID int64, accessToken string) (*entity.OrderEntity, error) {
	order, err := o.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		log.Errorf("[OrderService - 1] GetCustomerOrderByID: %v", err)
		return nil, err
	}

	userResponse, err := o.httpClientUserService(order.BuyerID, accessToken, true)
	if err == nil && userResponse != nil {
		order.BuyerID = userResponse.Data.ID
		order.BuyerName = userResponse.Data.Name
		order.BuyerEmail = userResponse.Data.Email
		order.BuyerPhone = userResponse.Data.Phone
		order.BuyerAddress = userResponse.Data.Address
	} else {
		log.Warnf("[OrderService - 2] GetCustomerOrderByID: Failed to fetch user profile for BuyerID %d", order.BuyerID)
	}

	for i := range order.OrderItems {
		productID := order.OrderItems[i].ProductID

		productResponse, err := o.httpClientProductService(productID, accessToken, true)
		if err == nil && productResponse != nil {
			order.OrderItems[i].ProductImage = productResponse.Data.Image
			order.OrderItems[i].ProductName = productResponse.Data.Name
			order.OrderItems[i].Price = int64(productResponse.Data.SalePrice)

			if order.ProductImage == "" {
				order.ProductImage = productResponse.Data.Image
			}
		} else {
			log.Warnf("[OrderService - 3] GetOrderByID: Failed to fetch product data for ProductID %d", productID)
		}
	}

	return order, nil
}

func (o *orderService) fetchData(url string, header map[string]string) (map[string]any, error) {
	resp, err := o.httpClient.CallURL("GET", url, header, nil)
	if err != nil {
		log.Errorf("Failed call URL: %v", err)
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (o *orderService) CreateOrder(ctx context.Context, req entity.OrderEntity, accessToken string) (int64, error) {
	req.OrderCode = generator.GenerateOrderCode()
	shippingFee := 0
	if req.ShippingType == "Delivery" {
		shippingFee = 5000
	}
	req.ShippingFee = float64(shippingFee)
	req.Status = "Pending"
	orderID, err := o.repo.CreateOrder(ctx, req)
	if err != nil {
		log.Errorf("[OrderService - 1] CreateOrder: %v", err)
		return 0, err
	}

	req.ID = orderID

	bgCtx := context.Background()

	go func(bgContext context.Context, orderData entity.OrderEntity, token string) {
		dataOrderByID, err := o.GetOrderByID(bgContext, orderData.ID, token)
		if err != nil {
			log.Errorf("[BackgroundWorker - GetOrderByID] failed to fill data: %v", err)
			return
		}
		err = message.PublishOrderToQueue(o.rabbitmq, *dataOrderByID, o.cfg.PublisherName.OrderPublish)
		if err != nil {
			log.Errorf("[BackgroundWorker - PublishOrder] failed to send messages: %v", err)
		}
	}(bgCtx, req, accessToken)

	for _, item := range req.OrderItems {
		go func(prodID int64, qty int64) {
			message.PublishUpdateStock(o.rabbitmq, item.ProductID, int64(item.Quantity), o.cfg.PublisherName.ProductUpdateStock)
		}(item.ProductID, int64(item.Quantity))
	}

	return orderID, nil

}

func (o *orderService) UpdateStatusOrder(ctx context.Context, req entity.OrderEntity, accessToken string) error {
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

	userUrl := fmt.Sprintf("%s/admin/customers/%d", o.cfg.App.UserServiceUrl, buyerID)

	header := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Accept":        "application/json",
	}

	userResponse, err := o.fetchData(userUrl, header)
	if err != nil {
		log.Errorf("[OrderService - 3] UpdateStatusOrder - Failed fetch user: %v", err)
		return err
	}

	var email string
	if userData, ok := userResponse["data"].(map[string]any); ok {
		if e, ok := userData["email"].(string); ok {
			email = e
		}
	}

	if email == "" {
		log.Warnf("[OrderService] Email not found for buyerID: %d, skipping email notification", buyerID)
		return nil
	}

	emailBody := fmt.Sprintf(
		"Hello,\n\nYour order with ID %s has been updated to status: %s.\n\nThank you for shopping with us!",
		orderCode,
		statusOrder,
	)

	bgCtx := context.Background()
	go func(ctx context.Context, targetEmail, body string) {
		err = message.PublishEmailUpdateStatus(o.rabbitmq, targetEmail, body, o.cfg.PublisherName.EmailUpdateStatus)
		if err != nil {
			log.Errorf("[Background-Email] Failed to publish: %v", err)
		}
	}(bgCtx, email, emailBody)

	log.Infof("[OrderService] Successfully updated order %s and queued email to %s", orderCode, email)
	return nil

}

func (o *orderService) GetAllCustomerOrders(ctx context.Context, queryString entity.QueryStringEntity, accessToken string) ([]entity.OrderEntity, int64, int64, error) {
	userProfile, err := o.httpClientUserService(0, accessToken, true)
	if err != nil {
		log.Errorf("[OrderService - 1] GetAllCustomerOrders: %v", err)
		return nil, 0, 0, err
	}

	queryString.BuyerID = userProfile.Data.ID

	results, count, total, err := o.repo.GetAllOrders(ctx, queryString)
	if err != nil {
		log.Errorf("[OrderService - 2] GetAllCustomerOrders: %v", err)
		return nil, 0, 0, err
	}

	for i := range results {
		results[i].BuyerName = userProfile.Data.Name
		results[i].BuyerEmail = userProfile.Data.Email
		results[i].BuyerPhone = userProfile.Data.Phone
		results[i].BuyerAddress = userProfile.Data.Address

		for j := range results[i].OrderItems {
			productResponse, err := o.httpClientProductService(results[i].OrderItems[j].ProductID, accessToken, true)
			if err != nil {
				log.Errorf("[OrderService-3] GetAllCustomer: %v", err)
				continue
			}

			results[i].OrderItems[j].ProductImage = productResponse.Data.Image
			results[i].OrderItems[j].ProductName = productResponse.Data.Name
			results[i].OrderItems[j].Price = int64(productResponse.Data.SalePrice)
			results[i].OrderItems[j].ProductUnit = productResponse.Data.Unit
			results[i].OrderItems[j].ProductWeight = int64(productResponse.Data.Weight)

			if results[i].ProductImage == "" {
				results[i].ProductImage = productResponse.Data.Image
			}
		}
	}

	return results, count, total, nil
}

func (o *orderService) httpClientUserService(userID int64, accessToken string, isCustomer bool) (*response.UserHttpClientResponse, error) {
	baseUrlUser := fmt.Sprintf("%s/%s", o.cfg.App.UserServiceUrl, "admin/customers/"+strconv.FormatInt(userID, 10))

	if isCustomer {
		baseUrlUser = fmt.Sprintf("%s/%s", o.cfg.App.UserServiceUrl, "auth/profile")
	}

	header := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Accept":        "application/json",
	}

	dataUser, err := o.httpClient.CallURL("GET", baseUrlUser, header, nil)
	if err != nil {
		log.Errorf("[OrderService-1] httpClientUserService: %v", err)
		return nil, err
	}
	defer dataUser.Body.Close()

	body, err := io.ReadAll(dataUser.Body)
	if err != nil {
		log.Errorf("[OrderService-2] httpClientUserService: %v", err)
		return nil, err
	}

	if dataUser.StatusCode != 200 {
		log.Errorf("[OrderService-HTTP-ERROR] Status: %d, Body: %s", dataUser.StatusCode, string(body))
	}

	var userResponse response.UserHttpClientResponse
	if err := json.Unmarshal(body, &userResponse); err != nil {
		log.Errorf("[OrderService-3] httpClientUserService: %v", err)
		return nil, err
	}

	return &userResponse, nil
}

func (o *orderService) httpClientProductService(productID int64, accessToken string, isCustomer bool) (*response.ProductHttpClientResponse, error) {
	baseUrlProduct := fmt.Sprintf("%s/%s", o.cfg.App.ProductServiceUrl, "admin/products/"+strconv.FormatInt(productID, 10))

	if isCustomer {
		baseUrlProduct = fmt.Sprintf("%s/%s", o.cfg.App.ProductServiceUrl, "products/"+strconv.FormatInt(productID, 10))
	}

	header := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Accept":        "application/json",
	}

	resp, err := o.httpClient.CallURL("GET", baseUrlProduct, header, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var productResponse response.ProductHttpClientResponse
	json.Unmarshal(body, &productResponse)

	return &productResponse, nil
}

func (o *orderService) GetOrderByOrderCode(ctx context.Context, code string, accessToken string) (*entity.OrderEntity, error) {
	order, err := o.repo.GetOrderByOrderCode(ctx, code)
	if err != nil {
		log.Errorf("[OrderService - 1] GetOrderByOrderCode: %v", err)
		return nil, err
	}

	userResponse, err := o.httpClientUserService(order.BuyerID, accessToken, false)
	if err == nil && userResponse != nil {
		order.BuyerID = userResponse.Data.ID
		order.BuyerName = userResponse.Data.Name
		order.BuyerEmail = userResponse.Data.Email
		order.BuyerPhone = userResponse.Data.Phone
		order.BuyerAddress = userResponse.Data.Address
	} else {
		log.Warnf("[OrderService - 2] GetOrderByOrderCode: Failed to fetch user profile for BuyerID %d", order.BuyerID)
	}

	for i := range order.OrderItems {
		productID := order.OrderItems[i].ProductID

		productResponse, err := o.httpClientProductService(productID, accessToken, false)
		if err == nil && productResponse != nil {
			order.OrderItems[i].ProductImage = productResponse.Data.Image
			order.OrderItems[i].ProductName = productResponse.Data.Name
			order.OrderItems[i].Price = int64(productResponse.Data.SalePrice)

			if order.ProductImage == "" {
				order.ProductImage = productResponse.Data.Image
			}
		} else {
			log.Warnf("[OrderService - 3] GetOrderByOrderCode: Failed to fetch product data for ProductID %d", productID)
		}
	}

	return order, nil
}

func GetRoleFromToken(tokenString string) string {
	claims := jwt.MapClaims{}
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, claims)
	if err != nil || token == nil {
		return ""
	}

	if role, ok := claims["role"].(string); ok {
		return role
	}
	return ""
}
