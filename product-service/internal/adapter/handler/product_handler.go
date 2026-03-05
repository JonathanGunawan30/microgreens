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
			ParentID:     product.ParentID,
			Image:        product.Image,
			CategoryName: product.CategorySlug,
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

	reqEntity := entity.ProductEntity{
		CategorySlug: req.CategorySlug,
		ParentID:     nil,
		Name:         req.Name,
		Image:        req.VariantDetail[0].Image,
		Description:  req.Description,
		RegulerPrice: req.VariantDetail[0].RegulerPrice,
		SalePrice:    req.VariantDetail[0].SalePrice,
		Unit:         req.Unit,
		Weight:       req.VariantDetail[0].Weight,
		Stock:        req.VariantDetail[0].Stock,
		Variant:      req.Variant,
		Status:       entity.ProductStatus(req.Status),
	}

	var productChildren []entity.ProductEntity
	if len(req.VariantDetail) > 1 {
		productChildren = make([]entity.ProductEntity, 0, len(req.VariantDetail))
		for i := 0; i < len(req.VariantDetail); i++ {
			productChildren = append(productChildren, entity.ProductEntity{
				Image:        req.VariantDetail[i].Image,
				RegulerPrice: req.VariantDetail[i].RegulerPrice,
				SalePrice:    req.VariantDetail[i].SalePrice,
				Weight:       req.VariantDetail[i].Weight,
				Stock:        req.VariantDetail[i].Stock,
			})
		}

		reqEntity.Child = productChildren
	}

	if err := p.productService.CreateProduct(ctx, reqEntity); err != nil {
		log.Errorf("[ProductHandler - 3] CreateProduct: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusCreated, resp)
}

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

	reqEntity := entity.ProductEntity{
		ID:           productID,
		CategorySlug: req.CategorySlug,
		ParentID:     nil,
		Name:         req.Name,
		Image:        req.VariantDetail[0].Image,
		Description:  req.Description,
		RegulerPrice: req.VariantDetail[0].RegulerPrice,
		SalePrice:    req.VariantDetail[0].SalePrice,
		Unit:         req.Unit,
		Weight:       req.VariantDetail[0].Weight,
		Stock:        req.VariantDetail[0].Stock,
		Variant:      req.Variant,
		Status:       entity.ProductStatus(req.Status),
	}

	var productChildren []entity.ProductEntity
	if len(req.VariantDetail) > 1 {
		productChildren = make([]entity.ProductEntity, 0, len(req.VariantDetail))
		for i := 0; i < len(req.VariantDetail); i++ {
			productChildren = append(productChildren, entity.ProductEntity{
				Image:        req.VariantDetail[i].Image,
				RegulerPrice: req.VariantDetail[i].RegulerPrice,
				SalePrice:    req.VariantDetail[i].SalePrice,
				Weight:       req.VariantDetail[i].Weight,
				Stock:        req.VariantDetail[i].Stock,
			})
		}

		reqEntity.Child = productChildren
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

func (p *productHandler) GetHomeProducts(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	limit := 5

	products, err := p.productService.GetHomeProducts(ctx, limit)
	if err != nil {
		log.Errorf("[ProductHandler] GetAllHome: %v", err)
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
			CategoryName: product.CategorySlug,
			RegulerPrice: product.RegulerPrice,
			SalePrice:    product.SalePrice,
		})
	}

	resp.Message = "Success"
	resp.Data = respProducts

	return c.JSON(http.StatusOK, resp)
}

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
		orderBy = "sale_price"
		orderType = "asc"

	case "price_desc":
		orderBy = "sale_price"
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
			ParentID:     product.ParentID,
			Image:        product.Image,
			CategoryName: product.CategorySlug,
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
		CategoryName: product.CategorySlug,
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
