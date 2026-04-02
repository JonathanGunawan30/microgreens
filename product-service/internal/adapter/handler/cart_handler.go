package handler

import (
	"errors"
	"net/http"
	"product-service/config"
	"product-service/internal/adapter"
	"product-service/internal/adapter/handler/request"
	"product-service/internal/adapter/handler/response"
	"product-service/internal/core/domain/entity"
	"product-service/internal/core/service"
	"product-service/utils/conv"
	"product-service/utils/message"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
)

type CartHandlerInterface interface {
	AddToCart(c echo.Context) error
	GetCart(c echo.Context) error
	DecreaseItem(c echo.Context) error
	RemoveFromCart(c echo.Context) error
	ClearCart(c echo.Context) error
}

type cartHandler struct {
	cartService service.CartServiceInterface
}

func NewCartHandler(e *echo.Echo, cartService service.CartServiceInterface, cfg *config.Config, redisClient *redis.Client) CartHandlerInterface {

	handler := &cartHandler{
		cartService: cartService,
	}

	e.Use(middleware.Recover())
	mid := adapter.NewMiddlewareAdapter(cfg, redisClient)

	cartGroup := e.Group("/auth/cart", mid.CheckToken(cfg.App.JwtSecretKey))

	cartGroup.GET("", handler.GetCart)
	cartGroup.POST("", handler.AddToCart)
	cartGroup.PATCH("/decrease", handler.DecreaseItem)
	cartGroup.DELETE("/:id", handler.RemoveFromCart)
	cartGroup.DELETE("", handler.ClearCart)

	return handler
}

func (h *cartHandler) GetCart(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	user := c.Get("user").(entity.JwtUserData)
	userID := user.ID

	cartItems, err := h.cartService.GetCart(ctx, userID)
	if err != nil {
		log.Errorf("[CartHandler - 1] GetCart: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = cartItems
	return c.JSON(http.StatusOK, resp)
}

func (h *cartHandler) AddToCart(c echo.Context) error {
	var (
		req  request.CartRequest
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	user := c.Get("user").(entity.JwtUserData)
	userID := user.ID

	if err := c.Bind(&req); err != nil {
		log.Errorf("[CartHandler - 1] AddToCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[CartHandler - 2] AddToCart: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	err := h.cartService.AddToCart(ctx, userID, req.ProductID, req.Quantity)
	if err != nil {
		log.Errorf("[CartHandler - 3] AddToCart: %v", err)

		if errors.Is(err, message.ErrProductLTZero) {
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusUnprocessableEntity, resp)
		}

		if errors.Is(err, message.ErrProductNotFound) {
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		resp.Message = "failed add to cart"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusCreated, resp)
}

func (h *cartHandler) DecreaseItem(c echo.Context) error {
	var (
		req  request.CartRequest
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	user := c.Get("user").(entity.JwtUserData)
	userID := user.ID

	if err := c.Bind(&req); err != nil {
		log.Errorf("[CartHandler - 1] DecreaseItem: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	err := h.cartService.DecreaseItem(ctx, userID, req.ProductID, req.Quantity)
	if err != nil {
		log.Errorf("[CartHandler - 2] DecreaseItem: %v", err)

		if errors.Is(err, message.ErrProductLTZero) {
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusUnprocessableEntity, resp)
		}

		resp.Message = "failed decrease item"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}

func (h *cartHandler) RemoveFromCart(c echo.Context) error {

	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	user := c.Get("user").(entity.JwtUserData)
	userID := user.ID

	productIDStr := c.Param("id")
	productID, err := conv.StringToInt64(productIDStr)
	if err != nil || productID <= 0 {
		resp.Message = "invalid product id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	err = h.cartService.RemoveFromCart(ctx, userID, productID)
	if err != nil {
		log.Errorf("[CartHandler - 1] RemoveFromCart: %v", err)
		resp.Message = "failed remove item"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}

func (h *cartHandler) ClearCart(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	user := c.Get("user").(entity.JwtUserData)
	userID := user.ID

	err := h.cartService.ClearCart(ctx, userID)
	if err != nil {
		log.Errorf("[CartHandler - 1] ClearCart: %v", err)
		resp.Message = "failed clear cart"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}
