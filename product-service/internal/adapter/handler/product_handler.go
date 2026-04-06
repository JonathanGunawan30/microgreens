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
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
)

type ProductHandlerInterface interface {
	GetAllAdminProducts(c echo.Context) error
	GetProductByID(c echo.Context) error
	CreateProduct(c echo.Context) error
	UpdateProduct(c echo.Context) error
	DeleteProductByID(c echo.Context) error
	GetHomeProducts(c echo.Context) error
	GetShopProducts(c echo.Context) error
	GetHomeProductDetail(c echo.Context) error
}

type productHandler struct {
	productService service.ProductServiceInterface
}

func NewProductHandler(e *echo.Echo, productService service.ProductServiceInterface, cfg *config.Config, redisClient *redis.Client) ProductHandlerInterface {
	productHandler := &productHandler{productService: productService}

	e.Use(middleware.Recover())
	mid := adapter.NewMiddlewareAdapter(cfg, redisClient)

	adminGroup := e.Group("/admin", mid.CheckToken(cfg.App.JwtSecretKey))
	adminGroup.GET("/products", productHandler.GetAllAdminProducts)
	adminGroup.GET("/products/:id", productHandler.GetProductByID)
	adminGroup.POST("/products", productHandler.CreateProduct)
	adminGroup.PUT("/products/:id", productHandler.UpdateProduct)
	adminGroup.DELETE("/products/:id", productHandler.DeleteProductByID)

	homeGroup := e.Group("/products")
	homeGroup.GET("", productHandler.GetShopProducts)
	homeGroup.GET("/featured", productHandler.GetHomeProducts)
	homeGroup.GET("/:id", productHandler.GetHomeProductDetail)

	return productHandler
}

// GetAllAdminProducts godoc
// @Summary Get all products (admin)
// @Description Get paginated list of products with filters for admin
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param search query string false "Search by name"
// @Param orderBy query string false "Order by (price_asc, price_desc, newest)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param startPrice query int false "Filter by minimum price"
// @Param endPrice query int false "Filter by maximum price"
// @Param status query string false "Filter by status"
// @Param category query string false "Filter by category slug"
// @Param is_parent query string false "Filter by parent product"
// @Success 200 {object} response.DefaultResponseWithPagination{data=[]response.ProductListResponse} "Success"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /admin/products [get]
func (p *productHandler) GetAllAdminProducts(c echo.Context) error {
	var (
		respProducts []response.ProductListResponse
		resp         = response.DefaultResponseWithPagination{}
		ctx          = c.Request().Context()
	)

	search := c.QueryParam("search")

	orderBy := "created_at"
	orderType := "desc"

	orderParam := c.QueryParam("orderBy")

	switch orderParam {
	case "price_asc":
		orderBy = "sale_price"
		orderType = "asc"

	case "price_desc":
		orderBy = "sale_price"
		orderType = "desc"

	case "newest":
		orderBy = "id"
		orderType = "desc"
	}

	var page int64 = 1
	if pageStr := c.QueryParam("page"); pageStr != "" {
		page, _ = conv.StringToInt64(pageStr)
		if page <= 0 {
			page = 1
		}
	}

	limitStr := c.QueryParam("limit")
	var limit int64 = 10
	if limitStr != "" {
		limit, _ = conv.StringToInt64(limitStr)
		if limit <= 0 {
			limit = 10
		}
	}

	startPriceStr := c.QueryParam("startPrice")
	var startPrice int64 = 0
	if startPriceStr != "" {
		startPrice, _ = conv.StringToInt64(startPriceStr)
		if startPrice < 0 {
			startPrice = 0
		}
	}

	endPriceStr := c.QueryParam("endPrice")
	var endPrice int64 = 0
	if endPriceStr != "" {
		endPrice, _ = conv.StringToInt64(endPriceStr)
		if endPrice < 0 {
			endPrice = 0
		}
	}

	status := "active"
	if c.QueryParam("status") != "" {
		status = c.QueryParam("status")
	}

	isParent := c.QueryParam("is_parent")

	productQuery := entity.QueryStringProduct{
		Search:       search,
		Page:         page,
		Limit:        limit,
		OrderBy:      orderBy,
		OrderType:    orderType,
		CategorySlug: c.QueryParam("category"),
		StartPrice:   startPrice,
		EndPrice:     endPrice,
		Status:       status,
		IsParent:     isParent,
	}

	products, count, totalPages, err := p.productService.GetAllProducts(ctx, productQuery)
	if err != nil {
		log.Errorf("[ProductHandler - 1] GetAllAdminProducts: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	respProducts = make([]response.ProductListResponse, 0, len(products))
	for _, product := range products {
		respProducts = append(respProducts, response.ProductListResponse{
			ID:           product.ID,
			Name:         product.Name,
			Image:        product.Image,
			CategoryName: product.CategoryName,
			Status:       response.ProductStatus(product.Status),
			SalePrice:    product.SalePrice,
			CreatedAt:    product.CreatedAt,
		})
	}

	resp.Message = "Success"
	resp.Data = respProducts
	resp.Pagination = response.PaginationMeta{
		Page:       page,
		TotalCount: count,
		PerPage:    limit,
		TotalPage:  totalPages,
	}

	return c.JSON(http.StatusOK, resp)
}

// GetProductByID godoc
// @Summary Get product by ID (admin)
// @Description Get product detail by ID for admin
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Success 200 {object} response.DefaultResponse{data=response.ProductDetailResponse} "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /admin/products/{id} [get]
func (p *productHandler) GetProductByID(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	productIDStr := c.Param("id")
	productID, err := conv.StringToInt64(productIDStr)
	if err != nil || productID <= 0 {
		log.Errorf("[ProductHandler - 1] GetProductByID: %v", err)
		resp.Message = "invalid product id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	product, err := p.productService.GetProductByID(ctx, productID)
	if err != nil {
		if errors.Is(err, message.ErrProductNotFound) {
			resp.Message = "product not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		log.Errorf("[ProductHandler - 2] GetProductByID: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	var childResponses []response.ProductChildResponse
	if len(product.Child) > 0 {
		childResponses = make([]response.ProductChildResponse, 0, len(product.Child))
		for _, child := range product.Child {
			childResponses = append(childResponses, response.ProductChildResponse{
				ID:           child.ID,
				Name:         child.Name,
				SalePrice:    child.SalePrice,
				RegulerPrice: child.RegulerPrice,
				Weight:       child.Weight,
				Stock:        child.Stock,
				Image:        child.Image,
				Unit:         child.Unit,
			})
		}
	}

	productResponse := response.ProductDetailResponse{
		ID:           product.ID,
		Name:         product.Name,
		ParentID:     product.ParentID,
		Image:        product.Image,
		CategoryName: product.CategoryName,
		CategorySlug: product.CategorySlug,
		Status:       response.ProductStatus(product.Status),
		Description:  product.Description,
		SalePrice:    product.SalePrice,
		RegulerPrice: product.RegulerPrice,
		CreatedAt:    product.CreatedAt,
		Unit:         product.Unit,
		Weight:       product.Weight,
		Stock:        product.Stock,
		Child:        childResponses,
	}

	resp.Message = "Success"
	resp.Data = productResponse
	return c.JSON(http.StatusOK, resp)
}

// CreateProduct godoc
// @Summary Create product (admin)
// @Description Create a new product with its variants
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.ProductRequest true "Create Product Request"
// @Success 201 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 404 {object} response.DefaultResponse "Category Not Found"
// @Failure 422 {object} response.DefaultResponse "Validation Error"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /admin/products [post]
func (p *productHandler) CreateProduct(c echo.Context) error {
	var (
		req  = request.ProductRequest{}
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	if err := c.Bind(&req); err != nil {
		log.Errorf("[ProductHandler - 1] CreateProduct: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[ProductHandler - 2] CreateProduct: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	totalStock := int64(0)

	minSalePrice := req.VariantDetail[0].SalePrice
	minRegPrice := req.VariantDetail[0].RegulerPrice

	var productChildren = make([]entity.ProductEntity, 0, len(req.VariantDetail))
	for _, v := range req.VariantDetail {
		totalStock += v.Stock

		if v.SalePrice < minSalePrice {
			minSalePrice = v.SalePrice
		}

		if v.RegulerPrice < minRegPrice {
			minRegPrice = v.RegulerPrice
		}

		productChildren = append(productChildren, entity.ProductEntity{
			Image:        v.Image,
			RegulerPrice: v.RegulerPrice,
			SalePrice:    v.SalePrice,
			Weight:       v.Weight,
			Stock:        v.Stock,
		})
	}

	reqEntity := entity.ProductEntity{
		CategorySlug: req.CategorySlug,
		ParentID:     nil,
		Name:         req.Name,
		Description:  req.Description,
		Unit:         req.Unit,
		Variant:      req.Variant,
		Status:       entity.ProductStatus(req.Status),
		Image:        req.VariantDetail[0].Image,
		RegulerPrice: minRegPrice,
		SalePrice:    minSalePrice,
		Weight:       0,
		Stock:        totalStock,
		Child:        productChildren,
	}

	if err := p.productService.CreateProduct(ctx, reqEntity); err != nil {
		log.Errorf("[ProductHandler - 3] CreateProduct: %v", err)

		if errors.Is(err, message.ErrCategoryNotFound) {
			resp.Message = "category not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusCreated, resp)
}

// UpdateProduct godoc
// @Summary Update product (admin)
// @Description Update an existing product by ID
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Param request body request.ProductRequest true "Update Product Request"
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 422 {object} response.DefaultResponse "Validation Error"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /admin/products/{id} [put]
func (p *productHandler) UpdateProduct(c echo.Context) error {
	var (
		req  = request.ProductRequest{}
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	productIDParam := c.Param("id")
	productID, err := conv.StringToInt64(productIDParam)
	if err != nil || productID <= 0 {
		log.Errorf("[ProductHandler -1] UpdateProduct: %v", err)
		resp.Message = "invalid product id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Bind(&req); err != nil {
		log.Errorf("[ProductHandler - 2] UpdateProduct: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[ProductHandler - 3] UpdateProduct: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	totalStock := int64(0)
	minSalePrice := req.VariantDetail[0].SalePrice
	minRegPrice := req.VariantDetail[0].RegulerPrice

	var productChildren = make([]entity.ProductEntity, 0, len(req.VariantDetail))

	for _, v := range req.VariantDetail {
		totalStock += v.Stock

		if v.SalePrice < minSalePrice {
			minSalePrice = v.SalePrice
		}
		if v.RegulerPrice < minRegPrice {
			minRegPrice = v.RegulerPrice
		}

		productChildren = append(productChildren, entity.ProductEntity{
			Image:        v.Image,
			RegulerPrice: v.RegulerPrice,
			SalePrice:    v.SalePrice,
			Weight:       v.Weight,
			Stock:        v.Stock,
		})
	}

	reqEntity := entity.ProductEntity{
		ID:           productID,
		CategorySlug: req.CategorySlug,
		ParentID:     nil,
		Name:         req.Name,
		Description:  req.Description,
		Unit:         req.Unit,
		Variant:      req.Variant,
		Status:       entity.ProductStatus(req.Status),
		Image:        req.VariantDetail[0].Image,
		RegulerPrice: minRegPrice,
		SalePrice:    minSalePrice,
		Stock:        totalStock,
		Weight:       0,
		Child:        productChildren,
	}

	err = p.productService.UpdateProduct(ctx, reqEntity)
	if err != nil {
		if errors.Is(err, message.ErrProductNotFound) {
			resp.Message = "product not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		log.Errorf("[ProductHandler - 4] UpdateProduct: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}

// DeleteProductByID godoc
// @Summary Delete product (admin)
// @Description Delete a product by ID
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /admin/products/{id} [delete]
func (p *productHandler) DeleteProductByID(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	productIDStr := c.Param("id")
	productID, err := conv.StringToInt64(productIDStr)
	if err != nil || productID <= 0 {
		log.Errorf("[ProductHandler - 1] DeleteProductByID: %v", err)
		resp.Message = "invalid product id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	err = p.productService.DeleteProductByID(ctx, productID)
	if err != nil {
		if errors.Is(err, message.ErrProductNotFound) {
			resp.Message = "product not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		log.Errorf("[ProductHandler - 2] DeleteProductByID: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}

// GetHomeProducts godoc
// @Summary Get featured products (public)
// @Description Get a limited list of featured products for the home page
// @Tags products
// @Accept json
// @Produce json
// @Param limit query int false "Items limit (default: 10)"
// @Success 200 {object} response.DefaultResponse{data=[]response.ProductHomeListResponse} "Success"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /products/featured [get]
func (p *productHandler) GetHomeProducts(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	limit := 10
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	products, err := p.productService.GetHomeProducts(ctx, limit)
	if err != nil {
		log.Errorf("[ProductHandler] GetHomeProducts: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	respProducts := make([]response.ProductHomeListResponse, 0, len(products))
	for _, product := range products {
		respProducts = append(respProducts, response.ProductHomeListResponse{
			ID:           product.ID,
			Name:         product.Name,
			Image:        product.Image,
			CategoryName: product.CategoryName,
			RegulerPrice: product.RegulerPrice,
			SalePrice:    product.SalePrice,
		})
	}

	resp.Message = "Success"
	resp.Data = respProducts

	return c.JSON(http.StatusOK, resp)
}

// GetShopProducts godoc
// @Summary Get all shop products (public)
// @Description Get paginated list of active products with filters for the shop page
// @Tags products
// @Accept json
// @Produce json
// @Param search query string false "Search by name"
// @Param category query string false "Filter by category slug"
// @Param orderBy query string false "Order by (price_asc, price_desc, newest)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param startPrice query int false "Filter by minimum price"
// @Param endPrice query int false "Filter by maximum price"
// @Success 200 {object} response.DefaultResponseWithPagination{data=[]response.ProductListResponse} "Success"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /products [get]
func (p *productHandler) GetShopProducts(c echo.Context) error {
	var (
		respProducts []response.ProductListResponse
		resp         = response.DefaultResponseWithPagination{}
		ctx          = c.Request().Context()
	)

	var page int64 = 1
	if pageStr := c.QueryParam("page"); pageStr != "" {
		page, _ = conv.StringToInt64(pageStr)
		if page <= 0 {
			page = 1
		}
	}

	limitStr := c.QueryParam("limit")
	var limit int64 = 10
	if limitStr != "" {
		limit, _ = conv.StringToInt64(limitStr)
		if limit <= 0 {
			limit = 10
		}
	}

	startPriceStr := c.QueryParam("startPrice")
	var startPrice int64 = 0
	if startPriceStr != "" {
		startPrice, _ = conv.StringToInt64(startPriceStr)
		if startPrice < 0 {
			startPrice = 0
		}
	}

	endPriceStr := c.QueryParam("endPrice")
	var endPrice int64 = 0
	if endPriceStr != "" {
		endPrice, _ = conv.StringToInt64(endPriceStr)
		if endPrice < 0 {
			endPrice = 0
		}
	}

	orderBy := "created_at"
	orderType := "desc"

	orderParam := c.QueryParam("orderBy")

	switch orderParam {
	case "price_asc":
		orderBy = "price"
		orderType = "asc"

	case "price_desc":
		orderBy = "price"
		orderType = "desc"

	case "newest":
		orderBy = "id"
		orderType = "desc"
	}

	productQuery := entity.QueryStringProduct{
		Search:       c.QueryParam("search"),
		Page:         page,
		Limit:        limit,
		OrderBy:      orderBy,
		OrderType:    orderType,
		CategorySlug: c.QueryParam("category"),
		StartPrice:   startPrice,
		EndPrice:     endPrice,
		Status:       "active",
	}

	products, count, totalPages, err := p.productService.SearchProducts(ctx, productQuery)
	if err != nil {
		log.Errorf("[ProductHandler - 1] GetAllAdminProducts: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	respProducts = make([]response.ProductListResponse, 0, len(products))
	for _, product := range products {
		respProducts = append(respProducts, response.ProductListResponse{
			ID:           product.ID,
			Name:         product.Name,
			Image:        product.Image,
			CategoryName: product.CategoryName,
			Status:       response.ProductStatus(product.Status),
			SalePrice:    product.SalePrice,
			RegulerPrice: product.RegulerPrice,
		})
	}

	resp.Message = "Success"
	resp.Data = respProducts
	resp.Pagination = response.PaginationMeta{
		Page:       page,
		TotalCount: count,
		PerPage:    limit,
		TotalPage:  totalPages,
	}

	return c.JSON(http.StatusOK, resp)
}

// GetHomeProductDetail godoc
// @Summary Get product detail (public)
// @Description Get detailed information of a specific product by ID for the home/shop page
// @Tags products
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Success 200 {object} response.DefaultResponse{data=response.ProductHomeDetailResponse} "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /products/{id} [get]
func (p *productHandler) GetHomeProductDetail(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	productIDStr := c.Param("id")
	productID, err := conv.StringToInt64(productIDStr)
	if err != nil || productID <= 0 {
		log.Errorf("[ProductHandler - 1] GetHomeProductDetail: %v", err)
		resp.Message = "invalid product id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	product, err := p.productService.GetProductByID(ctx, productID)
	if err != nil {
		if errors.Is(err, message.ErrProductNotFound) {
			resp.Message = "product not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		log.Errorf("[ProductHandler - 2] GetHomeProductDetail: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	var childResponses []response.ProductChildHomeResponse
	if len(product.Child) > 0 {
		childResponses = make([]response.ProductChildHomeResponse, 0, len(product.Child))
		for _, child := range product.Child {
			childResponses = append(childResponses, response.ProductChildHomeResponse{
				ID:           child.ID,
				Name:         child.Name,
				SalePrice:    child.SalePrice,
				RegulerPrice: child.RegulerPrice,
				Weight:       child.Weight,
				Stock:        child.Stock,
				Image:        child.Image,
			})
		}
	}

	productResponse := response.ProductHomeDetailResponse{
		ID:           product.ID,
		Name:         product.Name,
		CategoryName: product.CategoryName,
		CategorySlug: product.CategorySlug,
		Description:  product.Description,
		Unit:         product.Unit,
		Image:        product.Image,
		SalePrice:    product.SalePrice,
		RegulerPrice: product.RegulerPrice,
		Weight:       product.Weight,
		Child:        childResponses,
		Stock:        product.Stock,
	}

	resp.Message = "Success"
	resp.Data = productResponse
	return c.JSON(http.StatusOK, resp)
}
