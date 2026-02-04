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

type CategoryHandlerInterface interface {
	GetAllAdminCategories(c echo.Context) error
	GetByIDAdminCategory(c echo.Context) error
	GetBySlugAdminCategory(c echo.Context) error
	CreateAdminCategory(c echo.Context) error
	UpdateAdminCategory(c echo.Context) error
	DeleteAdminCategory(c echo.Context) error
	GetAllHomeCategories(c echo.Context) error
	GetAllShopCategories(c echo.Context) error
}

type categoryHandler struct {
	categoryService service.CategoryServiceInterface
}

func NewCategoryHandler(e *echo.Echo, categoryService service.CategoryServiceInterface, cfg *config.Config, redisClient *redis.Client) CategoryHandlerInterface {
	categoryHandler := &categoryHandler{categoryService: categoryService}

	e.Use(middleware.Recover())
	mid := adapter.NewMiddlewareAdapter(cfg, redisClient)

	adminGroup := e.Group("/admin", mid.CheckToken(cfg.App.JwtSecretKey))
	adminGroup.GET("/categories", categoryHandler.GetAllAdminCategories)
	adminGroup.GET("/categories/:id", categoryHandler.GetByIDAdminCategory)
	adminGroup.GET("/categories/slug/:slug", categoryHandler.GetBySlugAdminCategory)
	adminGroup.POST("/categories", categoryHandler.CreateAdminCategory)
	adminGroup.PUT("/categories/:id", categoryHandler.UpdateAdminCategory)
	adminGroup.DELETE("/categories/:id", categoryHandler.DeleteAdminCategory)

	publicGroup := e.Group("/categories")
	publicGroup.GET("", categoryHandler.GetAllShopCategories)
	publicGroup.GET("/featured", categoryHandler.GetAllShopCategories)

	return categoryHandler
}

func (h *categoryHandler) GetAllAdminCategories(c echo.Context) error {
	var (
		respCategories []response.CategoryListAdminResponse
		resp           = response.DefaultResponseWithPagination{}
		ctx            = c.Request().Context()
	)

	search := c.QueryParam("search")
	orderBy := "created_at"

	if c.QueryParam("orderBy") != "" {
		orderBy = c.QueryParam("orderBy")
	}

	orderType := "desc"
	if c.QueryParam("orderType") != "" {
		orderType = c.QueryParam("orderType")
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

	categoryQuery := entity.QueryStringCategory{
		Search:    search,
		Page:      page,
		Limit:     limit,
		OrderBy:   orderBy,
		OrderType: orderType,
	}

	categories, count, totalPages, err := h.categoryService.GetAllCategories(ctx, categoryQuery)
	if err != nil {
		log.Errorf("[CategoryHandler - 1] GetAllAdminCategories: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	respCategories = make([]response.CategoryListAdminResponse, 0, len(categories))

	for _, category := range categories {
		respCategories = append(respCategories, response.CategoryListAdminResponse{
			ID:           category.ID,
			Name:         category.Name,
			Icon:         category.Icon,
			Slug:         category.Slug,
			Status:       category.Status,
			TotalProduct: category.ProductCount,
		})
	}

	resp.Message = "Success"
	resp.Data = respCategories
	resp.Pagination = response.PaginationMeta{
		Page:       page,
		TotalCount: count,
		PerPage:    limit,
		TotalPage:  totalPages,
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *categoryHandler) GetByIDAdminCategory(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	categoryIDParam := c.Param("id")
	categoryID, err := conv.StringToInt64(categoryIDParam)
	if err != nil || categoryID <= 0 {
		log.Errorf("[CategoryHandler - 1] GetByIDAdminCategory: invalid category id: '%v'", err)
		resp.Message = "invalid category id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	category, err := h.categoryService.GetCategoryByID(ctx, categoryID)
	if err != nil {
		if errors.Is(err, message.ErrCategoryNotFound) {
			resp.Message = "category not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		log.Errorf("[CategoryHandler - 2] GetByIDAdminCategory: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	respCategory := response.CategoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		Icon:        category.Icon,
		Status:      category.Status,
		Slug:        category.Slug,
		Description: category.Description,
	}

	resp.Message = "Success"
	resp.Data = respCategory
	return c.JSON(http.StatusOK, resp)
}

func (h *categoryHandler) GetBySlugAdminCategory(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	categorySlugParam := c.Param("slug")
	if categorySlugParam == "" {
		log.Errorf("[CategoryHandler - 1] GetBySlugAdminCategory: invalid category slug")
		resp.Message = "invalid category slug"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	category, err := h.categoryService.GetCategoryBySlug(ctx, categorySlugParam)
	if err != nil {
		if errors.Is(err, message.ErrCategoryNotFound) {
			resp.Message = "category not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		log.Errorf("[CategoryHandler - 2] GetBySlugAdminCategory: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	respCategory := response.CategoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		Icon:        category.Icon,
		Status:      category.Status,
		Slug:        category.Slug,
		Description: category.Description,
	}

	resp.Message = "Success"
	resp.Data = respCategory
	return c.JSON(http.StatusOK, resp)
}

func (h *categoryHandler) CreateAdminCategory(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		req  = request.CreateCategoryRequest{}
		ctx  = c.Request().Context()
	)

	if err := c.Bind(&req); err != nil {
		log.Errorf("[CategoryHandler - 1] CreateAdminCategory: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[CategoryHandler - 2] CreateAdminCategory: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	categoryEntity := entity.CategoryEntity{
		ParentID:    req.ParentID,
		Name:        req.Name,
		Icon:        req.Icon,
		Status:      req.Status,
		Description: req.Description,
	}

	if err := h.categoryService.CreateCategory(ctx, categoryEntity); err != nil {
		if errors.Is(err, message.ErrCategoryAlreadyExists) {
			resp.Message = message.ErrCategoryAlreadyExists.Error()
			resp.Data = nil
			return c.JSON(http.StatusConflict, resp)
		}

		log.Errorf("[CategoryHandler - 3] CreateAdminCategory: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusCreated, resp)
}

func (h *categoryHandler) UpdateAdminCategory(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		req  = request.UpdateCategoryRequest{}
		ctx  = c.Request().Context()
	)

	categoryIDParam := c.Param("id")
	categoryID, err := conv.StringToInt64(categoryIDParam)
	if err != nil || categoryID <= 0 {
		log.Errorf("[CategoryHandler - 1] UpdateAdminCategory: invalid category id: '%v'", err)
		resp.Message = "invalid category id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err = c.Bind(&req); err != nil {
		log.Errorf("[UserHandler - 2] UpdateAdminCategory: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err = c.Validate(req); err != nil {
		log.Errorf("[UserHandler - 3] UpdateAdminCategory: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	categoryEntity := entity.CategoryEntity{
		ID:          categoryID,
		ParentID:    req.ParentID,
		Name:        req.Name,
		Icon:        req.Icon,
		Status:      req.Status,
		Description: req.Description,
	}

	err = h.categoryService.UpdateCategory(ctx, categoryEntity)
	if err != nil {
		if errors.Is(err, message.ErrCategoryNotFound) {
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		if errors.Is(err, message.ErrCategoryAlreadyExists) {
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusConflict, resp)
		}

		log.Errorf("[UserHandler - 4] UpdateAdminCategory: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)

}

func (h *categoryHandler) DeleteAdminCategory(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	categoryIDParam := c.Param("id")
	categoryID, err := conv.StringToInt64(categoryIDParam)
	if err != nil || categoryID <= 0 {
		log.Errorf("[CategoryHandler - 1] DeleteAdminCategory: invalid category id: '%v'", err)
		resp.Message = "invalid category id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	err = h.categoryService.DeleteCategoryByID(ctx, categoryID)
	if err != nil {
		if errors.Is(err, message.ErrCategoryNotFound) {
			resp.Message = "category not found"
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		if errors.Is(err, message.ErrCategoryHasProducts) {
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusConflict, resp)
		}

		log.Errorf("[CategoryHandler - 2] DeleteAdminCategory: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}

func (h *categoryHandler) GetAllHomeCategories(c echo.Context) error {
	var (
		respCategories []response.CategoryListHomeResponse
		resp           = response.DefaultResponse{}
		ctx            = c.Request().Context()
	)

	categories, err := h.categoryService.GetAllPublishedCategories(ctx)
	if err != nil {
		log.Errorf("[CategoryHandler - 1] GetAllHomeCategories: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	respCategories = make([]response.CategoryListHomeResponse, 0, len(categories))

	for _, category := range categories {
		if category.ParentID == nil {
			respCategories = append(respCategories, response.CategoryListHomeResponse{
				Name: category.Name,
				Icon: category.Icon,
				Slug: category.Slug,
			})
		}
	}

	resp.Message = "Success"
	resp.Data = respCategories

	return c.JSON(http.StatusOK, resp)
}

func (h *categoryHandler) GetAllShopCategories(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	categories, err := h.categoryService.GetAllPublishedCategories(ctx)
	if err != nil {
		log.Errorf("[CategoryHandler - 1] GetAllShopCategories: %v", err)
		resp.Message = "internal server error"
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	categoryMap := make(map[int64]*response.CategoryListShopResponse)

	for _, cat := range categories {
		categoryMap[cat.ID] = &response.CategoryListShopResponse{
			Name:  cat.Name,
			Slug:  cat.Slug,
			Child: make([]*response.CategoryListShopResponse, 0),
		}
	}

	var rootCategories []*response.CategoryListShopResponse

	for _, cat := range categories {
		currentResp := categoryMap[cat.ID]

		if cat.ParentID == nil {
			rootCategories = append(rootCategories, currentResp)
		} else {
			if parentResp, exists := categoryMap[*cat.ParentID]; exists {
				parentResp.Child = append(parentResp.Child, currentResp)
			} else {
				rootCategories = append(rootCategories, currentResp)
			}
		}
	}

	resp.Message = "Success"
	resp.Data = rootCategories

	return c.JSON(http.StatusOK, resp)
}
