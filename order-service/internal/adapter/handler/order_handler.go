package handler

import (
	"errors"
	"net/http"
	"order-service/config"
	"order-service/internal/adapter"
	"order-service/internal/adapter/handler/request"
	"order-service/internal/adapter/handler/response"
	"order-service/internal/core/domain/entity"
	"order-service/internal/core/service"
	"order-service/utils/conv"
	"order-service/utils/message"
	utils "order-service/utils/security"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
)

type OrderHandlerInterface interface {
	GetAllAdminOrders(c echo.Context) error
	GetOrderByID(c echo.Context) error
	GetOrderByOrderCode(c echo.Context) error
	CreateOrder(c echo.Context) error
	UpdateStatus(c echo.Context) error
	GetAllCustomerOrders(c echo.Context) error
}

type orderHandler struct {
	orderService service.OrderServiceInterface
}

func NewOrderHandler(orderService service.OrderServiceInterface, e *echo.Echo, cfg *config.Config, redisClient *redis.Client) OrderHandlerInterface {
	order := &orderHandler{orderService: orderService}

	e.Use(middleware.Recover())
	mid := adapter.NewMiddlewareAdapter(cfg, redisClient)
	authGroup := e.Group("/auth", mid.CheckToken(cfg.App.JwtSecretKey))
	authGroup.POST("/orders", order.CreateOrder, mid.DistanceCheck())
	authGroup.GET("/orders", order.GetAllCustomerOrders)
	authGroup.GET("/orders/:id", order.GetCustomerOrderByID)
	authGroup.GET("/orders/:code/code", order.GetOrderByOrderCode)

	adminGroup := e.Group("/admin", mid.CheckToken(cfg.App.JwtSecretKey))
	adminGroup.GET("/orders", order.GetAllAdminOrders)
	adminGroup.GET("/orders/:id", order.GetOrderByID)
	adminGroup.PATCH("/orders/:id/status", order.UpdateStatus)

	return order
}

func (o *orderHandler) GetAllAdminOrders(c echo.Context) error {
	var (
		respOrders    []response.OrderAdminList
		ctx           = c.Request().Context()
		categoryQuery entity.QueryStringEntity
	)

	if err := c.Bind(&categoryQuery); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("Invalid Request"))
	}

	if categoryQuery.Page <= 0 {
		categoryQuery.Page = 1
	}

	if categoryQuery.Limit <= 0 {
		categoryQuery.Limit = 10
	}

	accessToken := utils.GetTokenFromHeader(c)

	orders, count, totalPages, err := o.orderService.GetAllOrders(ctx, categoryQuery, accessToken)
	if err != nil {
		log.Errorf("[OrderHandler - 1] GetAllAdminOrders: %v", err)
		return c.JSON(http.StatusInternalServerError, response.Error("internal server error"))
	}

	for _, order := range orders {
		var productImage string
		if len(order.OrderItems) > 0 {
			productImage = order.OrderItems[0].ProductImage
		}
		respOrders = append(respOrders, response.OrderAdminList{
			ID:           order.ID,
			OrderCode:    order.OrderCode,
			ProductImage: productImage,
			CustomerName: order.BuyerName,
			Status:       order.Status,
			TotalAmount:  int64(order.TotalAmount),
		})
	}

	return c.JSON(http.StatusOK, response.SuccessWithPagination("Success", respOrders, categoryQuery.Page, count, categoryQuery.Limit, totalPages))
}

func (o *orderHandler) GetOrderByID(c echo.Context) error {
	var (
		ctx       = c.Request().Context()
		respOrder response.OrderAdminDetail
	)

	orderIDStr := c.Param("id")
	orderID, err := conv.StringToInt64(orderIDStr)
	if err != nil || orderID <= 0 {
		log.Errorf("[OrderHandler - 1] GetOrderByID: %v", err)
		return c.JSON(http.StatusBadRequest, response.Error("invalid order id"))
	}

	accessToken := utils.GetTokenFromHeader(c)

	order, err := o.orderService.GetOrderByID(ctx, orderID, accessToken)
	if err != nil {
		log.Errorf("[OrderHandler - 2] GetOrderByID: %v", err)
		if errors.Is(err, message.ErrOrderNotFound) {
			return c.JSON(http.StatusNotFound, response.Error("order not found"))
		}
		return c.JSON(http.StatusInternalServerError, response.Error("internal server error"))
	}

	respOrder.ID = order.ID
	respOrder.OrderCode = order.OrderCode
	respOrder.ProductImage = order.ProductImage
	respOrder.Status = order.Status
	respOrder.TotalAmount = int64(order.TotalAmount)
	respOrder.OrderDateTime = order.OrderDate
	respOrder.ShippingFee = int64(order.ShippingFee)
	respOrder.ShippingType = order.ShippingType
	respOrder.Remarks = order.Remarks
	respOrder.Customer = response.CustomerOrder{
		CustomerID:      order.BuyerID,
		CustomerName:    order.BuyerName,
		CustomerPhone:   order.BuyerPhone,
		CustomerEmail:   order.BuyerEmail,
		CustomerAddress: order.BuyerAddress,
	}

	for _, item := range order.OrderItems {
		respOrder.OrderDetail = append(respOrder.OrderDetail, response.OrderDetail{
			ProductName:  item.ProductName,
			ProductImage: item.ProductImage,
			ProductPrice: item.Price,
			Quantity:     int64(item.Quantity),
		})
	}

	return c.JSON(http.StatusOK, response.Success("success", respOrder))

}

func (o *orderHandler) GetCustomerOrderByID(c echo.Context) error {
	var (
		ctx       = c.Request().Context()
		respOrder response.OrderAdminDetail
	)

	orderIDStr := c.Param("id")
	orderID, err := conv.StringToInt64(orderIDStr)
	if err != nil || orderID <= 0 {
		log.Errorf("[OrderHandler - 1] GetCustomerOrderByID: %v", err)
		return c.JSON(http.StatusBadRequest, response.Error("invalid order id"))
	}

	accessToken := utils.GetTokenFromHeader(c)

	order, err := o.orderService.GetCustomerOrderByID(ctx, orderID, accessToken)
	if err != nil {
		log.Errorf("[OrderHandler - 2] GetCustomerOrderByID: %v", err)
		if errors.Is(err, message.ErrOrderNotFound) {
			return c.JSON(http.StatusNotFound, response.Error("order not found"))
		}
		return c.JSON(http.StatusInternalServerError, response.Error("internal server error"))
	}

	respOrder.ID = order.ID
	respOrder.OrderCode = order.OrderCode
	respOrder.ProductImage = order.ProductImage
	respOrder.Status = order.Status
	respOrder.TotalAmount = int64(order.TotalAmount)
	respOrder.OrderDateTime = order.OrderDate
	respOrder.ShippingFee = int64(order.ShippingFee)
	respOrder.ShippingType = order.ShippingType
	respOrder.PaymentMethod = order.PaymentMethod
	respOrder.Remarks = order.Remarks
	respOrder.Customer = response.CustomerOrder{
		CustomerID:      order.BuyerID,
		CustomerName:    order.BuyerName,
		CustomerPhone:   order.BuyerPhone,
		CustomerEmail:   order.BuyerEmail,
		CustomerAddress: order.BuyerAddress,
	}

	for _, item := range order.OrderItems {
		respOrder.OrderDetail = append(respOrder.OrderDetail, response.OrderDetail{
			ProductName:  item.ProductName,
			ProductImage: item.ProductImage,
			ProductPrice: item.Price,
			Quantity:     int64(item.Quantity),
		})
	}

	return c.JSON(http.StatusOK, response.Success("success", respOrder))

}

func (o *orderHandler) CreateOrder(c echo.Context) error {
	var (
		ctx = c.Request().Context()
		req = request.CreateOrderRequest{}
	)

	if err := c.Bind(&req); err != nil {
		log.Errorf("[OrderHandler - 1] CreateOrder: %v", err)
		return c.JSON(http.StatusBadRequest, response.Error(err.Error()))
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[OrderHandler - 2] CreateOrder: %v", err)
		return c.JSON(http.StatusUnprocessableEntity, response.Error(err.Error()))
	}

	reqEntity := entity.OrderEntity{
		BuyerID:      req.BuyerID,
		OrderDate:    req.OrderDate,
		TotalAmount:  req.TotalAmount,
		ShippingType: req.ShippingType,
		Remarks:      req.Remarks,
		OrderTime:    req.OrderTime,
	}

	var orderDetails []entity.OrderItemEntity
	for _, detail := range req.OrderDetails {
		orderDetails = append(orderDetails, entity.OrderItemEntity{
			ProductID: detail.ProductID,
			Quantity:  detail.Quantity,
		})
	}

	reqEntity.OrderItems = orderDetails

	accessToken := utils.GetTokenFromHeader(c)

	orderID, err := o.orderService.CreateOrder(ctx, reqEntity, accessToken)
	if err != nil {
		log.Errorf("[OrderHandler - 3] CreateOrder: %v", err)
		return c.JSON(http.StatusInternalServerError, response.Error(err.Error()))
	}

	return c.JSON(http.StatusCreated, response.Success("Success", map[string]any{
		"order_id": orderID,
	}))

}

func (o *orderHandler) UpdateStatus(c echo.Context) error {
	var (
		ctx = c.Request().Context()
		req = request.OrderUpdateStatusRequest{}
	)

	if err := c.Bind(&req); err != nil {
		log.Errorf("[OrderHandler - 1] UpdateStatus: %v", err)
		return c.JSON(http.StatusBadRequest, response.Error(err.Error()))
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[OrderHandler - 2] UpdateStatus: %v", err)
		return c.JSON(http.StatusUnprocessableEntity, response.Error(err.Error()))
	}

	orderIDStr := c.Param("id")
	orderID, err := conv.StringToInt64(orderIDStr)
	if err != nil || orderID <= 0 {
		log.Errorf("[OrderHandler - 3] UpdateStatus: %v", err)
		return c.JSON(http.StatusBadRequest, response.Error("invalid order id"))
	}

	accessToken := utils.GetTokenFromHeader(c)

	reqEntity := entity.OrderEntity{
		Remarks: req.Remarks,
		Status:  req.Status,
		ID:      orderID,
	}

	err = o.orderService.UpdateStatusOrder(ctx, reqEntity, accessToken)
	if err != nil {
		log.Errorf("[OrderHandler - 4] UpdateStatus: %v", err)
		if errors.Is(err, message.ErrOrderNotFound) {
			return c.JSON(http.StatusNotFound, response.Error("order not found"))
		}

		if strings.Contains(err.Error(), "invalid status transition") {
			return c.JSON(http.StatusBadRequest, response.Error(err.Error()))
		}

		return c.JSON(http.StatusInternalServerError, response.Error("internal server error"))
	}

	return c.JSON(http.StatusOK, response.Success("Success", nil))

}

func (o *orderHandler) GetAllCustomerOrders(c echo.Context) error {
	var (
		ctx         = c.Request().Context()
		queryString entity.QueryStringEntity
		respOrders  = make([]response.OrderCustomerList, 0)
	)

	if err := c.Bind(&queryString); err != nil {
		log.Errorf("[OrderHandler] GetAllCustomerOrders Bind Error: %v", err)
		return c.JSON(http.StatusBadRequest, response.Error("invalid request parameters"))
	}

	if queryString.Page <= 0 {
		queryString.Page = 1
	}
	if queryString.Limit <= 0 {
		queryString.Limit = 10
	}

	accessToken := utils.GetTokenFromHeader(c)

	orders, count, totalPages, err := o.orderService.GetAllCustomerOrders(ctx, queryString, accessToken)
	if err != nil {
		log.Errorf("[OrderHandler] GetAllCustomerOrders Service Error: %v", err)
		if errors.Is(err, message.ErrOrderNotFound) {
			return c.JSON(http.StatusNotFound, response.Error("order not found"))
		}
		return c.JSON(http.StatusInternalServerError, response.Error("internal server error"))
	}

	for _, order := range orders {
		var productImage, unit string
		var weight, quantity int64

		if len(order.OrderItems) > 0 {
			firstItem := order.OrderItems[0]
			productImage = firstItem.ProductImage
			unit = firstItem.ProductUnit
			weight = firstItem.ProductWeight
			quantity = int64(firstItem.Quantity)
		}

		respOrders = append(respOrders, response.OrderCustomerList{
			ID:            order.ID,
			OrderCode:     order.OrderCode,
			ProductImage:  productImage,
			Status:        order.Status,
			PaymentMethod: order.PaymentMethod,
			TotalAmount:   int64(order.TotalAmount),
			Weight:        weight,
			Unit:          unit,
			Quantity:      quantity,
			OrderDateTime: order.OrderDate,
		})
	}

	return c.JSON(http.StatusOK, response.SuccessWithPagination(
		"success",
		respOrders,
		queryString.Page,
		count,
		queryString.Limit,
		totalPages,
	))
}

func (o *orderHandler) GetOrderByOrderCode(c echo.Context) error {
	var (
		ctx       = c.Request().Context()
		respOrder response.OrderAdminDetail
	)

	orderCode := c.Param("code")

	accessToken := utils.GetTokenFromHeader(c)

	order, err := o.orderService.GetOrderByOrderCode(ctx, orderCode, accessToken)
	if err != nil {
		log.Errorf("[OrderHandler - 1] GetOrderByOrderCode: %v", err)
		if errors.Is(err, message.ErrOrderNotFound) {
			return c.JSON(http.StatusNotFound, response.Error("order not found"))
		}
		return c.JSON(http.StatusInternalServerError, response.Error("internal server error"))
	}

	respOrder.ID = order.ID
	respOrder.OrderCode = order.OrderCode
	respOrder.ProductImage = order.ProductImage
	respOrder.Status = order.Status
	respOrder.TotalAmount = int64(order.TotalAmount)
	respOrder.OrderDateTime = order.OrderDate
	respOrder.ShippingFee = int64(order.ShippingFee)
	respOrder.ShippingType = order.ShippingType
	respOrder.PaymentMethod = order.PaymentMethod
	respOrder.Remarks = order.Remarks
	respOrder.Customer = response.CustomerOrder{
		CustomerID:      order.BuyerID,
		CustomerName:    order.BuyerName,
		CustomerPhone:   order.BuyerPhone,
		CustomerEmail:   order.BuyerEmail,
		CustomerAddress: order.BuyerAddress,
	}

	for _, item := range order.OrderItems {
		respOrder.OrderDetail = append(respOrder.OrderDetail, response.OrderDetail{
			ProductName:  item.ProductName,
			ProductImage: item.ProductImage,
			ProductPrice: item.Price,
			Quantity:     int64(item.Quantity),
		})
	}

	return c.JSON(http.StatusOK, response.Success("success", respOrder))

}
