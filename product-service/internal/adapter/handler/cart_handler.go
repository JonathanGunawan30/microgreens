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

// GetCart godoc
// @Summary Get cart items
// @Description Get all cart items for authenticated user
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /auth/cart [get]
func (h *cartHandler) GetCart(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	user, ok := c.Get("user").(entity.JwtUserData)
	if !ok {
		resp.Message = "invalid user context"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}
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

// AddToCart godoc
// @Summary Add item to cart
// @Description Add a product to the authenticated user's cart
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.CartRequest true "Cart Request"
// @Success 201 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 404 {object} response.DefaultResponse "Product Not Found"
// @Failure 422 {object} response.DefaultResponse "Validation Error"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /auth/cart [post]
func (h *cartHandler) AddToCart(c echo.Context) error {
	var (
		req  request.CartRequest
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	user, ok := c.Get("user").(entity.JwtUserData)
	if !ok {
		resp.Message = "invalid user context"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}
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

		if errors.Is(err, message.ErrProductLTZero) || errors.Is(err, message.ErrQuantityExceeds) {
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

// DecreaseItem godoc
// @Summary Decrease cart item quantity
// @Description Decrease the quantity of a product in the authenticated user's cart
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.CartRequest true "Cart Request"
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 422 {object} response.DefaultResponse "Validation Error"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /auth/cart/decrease [patch]
func (h *cartHandler) DecreaseItem(c echo.Context) error {
	var (
		req  request.CartRequest
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	user, ok := c.Get("user").(entity.JwtUserData)
	if !ok {
		resp.Message = "invalid user context"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}
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

// RemoveFromCart godoc
// @Summary Remove item from cart
// @Description Remove a specific product from the authenticated user's cart
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /auth/cart/{id} [delete]
func (h *cartHandler) RemoveFromCart(c echo.Context) error {

	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	user, ok := c.Get("user").(entity.JwtUserData)
	if !ok {
		resp.Message = "invalid user context"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}
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

// ClearCart godoc
// @Summary Clear cart
// @Description Remove all items from the authenticated user's cart
// @Tags cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /auth/cart [delete]
func (h *cartHandler) ClearCart(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	user, ok := c.Get("user").(entity.JwtUserData)
	if !ok {
		resp.Message = "invalid user context"
		resp.Data = nil
		return c.JSON(http.StatusUnauthorized, resp)
	}
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
