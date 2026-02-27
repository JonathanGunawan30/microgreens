package handler

import (
	"errors"
	"net/http"
	"payment-service/config"
	"payment-service/internal/adapter"
	"payment-service/internal/adapter/handler/request"
	"payment-service/internal/adapter/handler/response"
	"payment-service/internal/core/domain/entity"
	"payment-service/internal/core/service"
	"payment-service/utils/conv"
	"payment-service/utils/message"
	"payment-service/utils/security"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
)

type PaymentHandlerInterface interface {
	CreatePayment(c echo.Context) error
	MidtransWebHook(c echo.Context) error
	GetAllPayments(c echo.Context) error
	GetPaymentDetail(c echo.Context) error
}

type paymentHandler struct {
	paymentService service.PaymentServiceInterface
}

func NewPaymentHandler(paymentService service.PaymentServiceInterface, e *echo.Echo, cfg *config.Config, redisClient *redis.Client) PaymentHandlerInterface {
	payment := &paymentHandler{paymentService: paymentService}

	e.Use(middleware.Recover())
	mid := adapter.NewMiddlewareAdapter(cfg, redisClient)

	e.POST("/payments/webhook", payment.MidtransWebHook)

	authGroup := e.Group("/auth", mid.CheckToken(cfg.App.JwtSecretKey))
	authGroup.GET("/payments", payment.GetAllPayments)
	authGroup.POST("/payments", payment.CreatePayment)
	authGroup.GET("/payments/:id", payment.GetPaymentDetail)

	adminGroup := e.Group("/admin", mid.CheckToken(cfg.App.JwtSecretKey))
	adminGroup.GET("/payments", payment.GetAllPayments)
	adminGroup.GET("/payments/:id", payment.GetPaymentDetail)

	return payment
}

func (p *paymentHandler) CreatePayment(c echo.Context) error {
	var (
		ctx = c.Request().Context()
		req = request.PaymentRequest{}
	)

	if err := c.Bind(&req); err != nil {
		log.Errorf("[PaymentHandler - 1] CreatePayment: %v", err)
		return c.JSON(http.StatusBadRequest, response.Error("Invalid Request"))
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[PaymentHandler - 2] CreatePayment: %v", err)
		return c.JSON(http.StatusUnprocessableEntity, response.Error(err.Error()))
	}

	accessToken := utils.GetTokenFromHeader(c)

	paymentEntity := entity.PaymentEntity{
		OrderID:       req.OrderID,
		PaymentMethod: req.PaymentMethod,
		GrossAmount:   req.GrossAmount,
		UserID:        req.UserID,
		Remarks:       req.Remarks,
	}

	processPayment, err := p.paymentService.ProcessPayment(ctx, paymentEntity, accessToken)
	if err != nil {
		log.Errorf("[PaymentHandler - 3] CreatePayment: %v", err)

		if errors.Is(err, message.ErrInvalidPaymentMethod) {
			return c.JSON(http.StatusUnprocessableEntity, response.Error("Invalid Payment Method"))
		}

		return c.JSON(http.StatusInternalServerError, response.Error("Internal Server Error"))
	}

	paymentToken := map[string]any{
		"payment_token": processPayment.PaymentGatewayID,
	}

	return c.JSON(http.StatusCreated, response.Success("Success", paymentToken))

}

func (p *paymentHandler) MidtransWebHook(c echo.Context) error {
	var (
		ctx     = c.Request().Context()
		payload request.MidtransWebhookPayload
	)

	if err := c.Bind(&payload); err != nil {
		log.Errorf("[PaymentHandler] MidtransWebhookHandler bind error: %v", err)
		return c.JSON(http.StatusBadRequest, response.Error("Invalid Request"))
	}

	isValid := p.paymentService.VerifyMidtransSignature(
		payload.OrderID,
		payload.StatusCode,
		payload.GrossAmount,
		payload.SignatureKey,
	)

	if !isValid {
		log.Warnf("[SECURITY ALERT] Fake Midtrans webhook detected for Order: %s", payload.OrderID)
		return c.JSON(http.StatusForbidden, response.Error("Invalid signature key"))
	}

	var newStatus string
	switch payload.TransactionStatus {
	case "capture", "settlement":
		newStatus = "success"
	case "deny", "cancel", "expire":
		newStatus = "failed"
	case "pending":
		newStatus = "pending"
	default:
		return c.JSON(http.StatusOK, response.Success("Status ignored", nil))
	}

	accessToken := utils.GetTokenFromHeader(c)
	err := p.paymentService.UpdateStatusByOrderCode(ctx, payload.OrderID, newStatus, accessToken)
	if err != nil {
		log.Errorf("[PaymentHandler] Failed to update DB for Order %s: %v", payload.OrderID, err)
		return c.JSON(http.StatusInternalServerError, response.Error("Internal Server Error"))
	}

	return c.JSON(http.StatusOK, response.Success("Success", nil))
}

func (p *paymentHandler) GetAllPayments(c echo.Context) error {
	var (
		ctx          = c.Request().Context()
		resp         []response.PaymentListResponse
		paymentQuery entity.PaymentQueryStringRequest
	)

	if err := c.Bind(&paymentQuery); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("Invalid Request"))
	}

	if paymentQuery.Page <= 0 {
		paymentQuery.Page = 1
	}
	if paymentQuery.Limit <= 0 {
		paymentQuery.Limit = 10
	}

	userData, ok := c.Get("user").(entity.JwtUserData)
	if !ok {
		return c.JSON(http.StatusUnauthorized, response.Error("Unauthorized"))
	}

	if strings.ToLower(userData.RoleName) == "customer" {
		paymentQuery.UserID = userData.ID
	}

	accessToken := utils.GetTokenFromHeader(c)

	entities, totalData, totalPage, err := p.paymentService.GetAllPayments(ctx, paymentQuery, accessToken)
	if err != nil {
		log.Errorf("[PaymentHandler] GetAllAdmin: %v", err)
		return c.JSON(http.StatusInternalServerError, response.Error("Internal Server Error"))
	}

	for _, paymentEntity := range entities {
		resp = append(resp, response.PaymentListResponse{
			ID:            paymentEntity.ID,
			OrderCode:     paymentEntity.OrderCode,
			PaymentStatus: paymentEntity.PaymentStatus,
			PaymentMethod: paymentEntity.PaymentMethod,
			GrossAmount:   paymentEntity.GrossAmount,
			ShippingType:  paymentEntity.OrderShippingType,
		})
	}

	return c.JSON(http.StatusOK, response.SuccessWithPagination("Success", resp, paymentQuery.Page, paymentQuery.Limit, totalData, totalPage))
}

func (p *paymentHandler) GetPaymentDetail(c echo.Context) error {
	var (
		ctx  = c.Request().Context()
		resp = response.PaymentDetailResponse{}
	)

	paymentIDStr := c.Param("id")
	paymentID, err := conv.StringToInt64(paymentIDStr)
	if err != nil || paymentID <= 0 {
		log.Errorf("[PaymentHandler] GetPaymentDetail: %v", err)
		return c.JSON(http.StatusBadRequest, response.Error("invalid payment id"))
	}

	accessToken := utils.GetTokenFromHeader(c)

	userData, ok := c.Get("user").(entity.JwtUserData)
	if !ok {
		return c.JSON(http.StatusUnauthorized, response.Error("Unauthorized"))
	}

	var filterUserID int64 = 0

	if strings.ToLower(userData.RoleName) == "customer" {
		filterUserID = userData.ID
	}

	paymentDetail, err := p.paymentService.GetPaymentDetail(ctx, paymentID, accessToken, filterUserID)
	if err != nil {
		log.Errorf("[PaymentHandler] GetPaymentDetail: %v", err)

		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "not found") {
			return c.JSON(http.StatusNotFound, response.Error("data not found"))
		}

		if strings.Contains(errMsg, "invalid") || strings.Contains(errMsg, "bad request") {
			return c.JSON(http.StatusBadRequest, response.Error(err.Error()))
		}

		return c.JSON(http.StatusInternalServerError, response.Error("internal server error"))
	}

	resp.ID = paymentDetail.ID
	resp.OrderCode = paymentDetail.OrderCode
	resp.PaymentMethod = paymentDetail.PaymentMethod
	resp.PaymentStatus = paymentDetail.PaymentStatus
	resp.GrossAmount = paymentDetail.GrossAmount
	resp.ShippingType = paymentDetail.OrderShippingType
	resp.PaymentAt = paymentDetail.PaymentAt
	resp.OrderAt = paymentDetail.OrderAt
	resp.OrderRemarks = paymentDetail.OrderRemarks
	resp.CustomerName = paymentDetail.CustomerName
	resp.CustomerAddress = paymentDetail.CustomerAddress

	return c.JSON(http.StatusOK, response.Success("Success", resp))

}
