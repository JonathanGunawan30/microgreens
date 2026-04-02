package handler

import (
	"errors"
	"net/http"
	"strconv"
	"user-service/config"
	"user-service/internal/adapter"
	"user-service/internal/adapter/handler/request"
	"user-service/internal/adapter/handler/response"
	"user-service/internal/core/domain/entity"
	"user-service/internal/core/service"
	"user-service/utils/message"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
)

type RoleHandlerInterface interface {
	GetAllRole(c echo.Context) error
	GetRoleByID(c echo.Context) error
	CreateRole(c echo.Context) error
	UpdateRole(c echo.Context) error
	DeleteRoleByID(c echo.Context) error
}

type roleHandler struct {
	roleService service.RoleServiceInterface
}

func NewRoleHandler(e *echo.Echo, roleService service.RoleServiceInterface, cfg *config.Config, redisClient *redis.Client) RoleHandlerInterface {
	roleHandler := &roleHandler{roleService: roleService}

	e.Use(middleware.Recover())
	mid := adapter.NewMiddlewareAdapter(cfg, redisClient)

	adminGroup := e.Group("/admin", mid.CheckToken(cfg.App.JwtSecretKey))
	adminGroup.GET("/roles", roleHandler.GetAllRole)
	adminGroup.GET("/roles/:id", roleHandler.GetRoleByID)
	adminGroup.POST("/roles", roleHandler.CreateRole)
	adminGroup.PUT("/roles/:id", roleHandler.UpdateRole)
	adminGroup.DELETE("/roles/:id", roleHandler.DeleteRoleByID)

	return roleHandler
}

// GetAllRole godoc
// @Summary Get all roles
// @Description Get list of all roles
// @Tags roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param search query string false "Search roles by name"
// @Success 200 {object} response.DefaultResponse{data=[]response.RoleResponse} "Success"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /admin/roles [get]
func (r *roleHandler) GetAllRole(c echo.Context) error {
	var (
		respRole []response.RoleResponse
		resp     = response.DefaultResponse{}
		ctx      = c.Request().Context()
	)

	search := c.QueryParam("search")

	roles, err := r.roleService.GetAllRole(ctx, search)
	if err != nil {
		log.Errorf("[RoleHandler - 1] GetAllRole: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	for _, role := range roles {
		respRole = append(respRole, response.RoleResponse{
			ID:   role.ID,
			Name: role.Name,
		})
	}

	resp.Message = "Success"
	resp.Data = respRole
	return c.JSON(http.StatusOK, resp)
}

// GetRoleByID godoc
// @Summary Get role by ID
// @Description Get role details by its ID
// @Tags roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Role ID"
// @Success 200 {object} response.DefaultResponse{data=response.RoleResponse} "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /admin/roles/{id} [get]
func (r *roleHandler) GetRoleByID(c echo.Context) error {
	var (
		resp     = response.DefaultResponse{}
		ctx      = c.Request().Context()
		respRole = response.RoleResponse{}
	)

	strID := c.Param("id")
	if strID == "" {
		log.Errorf("[RoleHandler - 1] GetRoleByID: missing role id")
		resp.Message = "missing role id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	ID, err := strconv.Atoi(strID)
	if err != nil {
		log.Errorf("[RoleHandler - 2] GetRoleByID: invalid role id")
		resp.Message = "invalid role id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	role, err := r.roleService.GetRoleByID(ctx, int64(ID))
	if err != nil {
		if errors.Is(err, message.ErrRoleNotFound) {
			log.Errorf("[RoleHandler - 3] GetRoleByID: %v", err)
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}

		log.Errorf("[RoleHandler - 4] GetRoleByID: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	respRole.ID = role.ID
	respRole.Name = role.Name

	resp.Message = "Success"
	resp.Data = respRole
	return c.JSON(http.StatusOK, resp)
}

// CreateRole godoc
// @Summary Create new role
// @Description Create a new role with name
// @Tags roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body request.RoleRequest true "Role Name"
// @Success 201 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 422 {object} response.DefaultResponse "Validation Error"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /admin/roles [post]
func (r *roleHandler) CreateRole(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
		req  = request.RoleRequest{}
	)

	if err := c.Bind(&req); err != nil {
		log.Errorf("[RoleHandler - 4] CreateRole: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[RoleHandler -5] CreateRole: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	roleEntity := entity.RoleEntity{
		Name: req.Name,
	}

	err := r.roleService.CreateRole(ctx, roleEntity)
	if err != nil {
		log.Errorf("[RoleHandler - 6] CreateRole: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusCreated, resp)
}

// UpdateRole godoc
// @Summary Update role
// @Description Update role name by its ID
// @Tags roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Role ID"
// @Param request body request.RoleRequest true "Updated Role Name"
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 422 {object} response.DefaultResponse "Validation Error"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /admin/roles/{id} [put]
func (r *roleHandler) UpdateRole(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
		req  = request.RoleRequest{}
	)

	if err := c.Bind(&req); err != nil {
		log.Errorf("[RoleHandler - 1] UpdateRole: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	if err := c.Validate(req); err != nil {
		log.Errorf("[RoleHandler - 2] UpdateRole: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusUnprocessableEntity, resp)
	}

	roleIDString := c.Param("id")
	if roleIDString == "" {
		log.Errorf("[RoleHandler - 3] UpdateRole: missing role id")
		resp.Message = "missing role id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	roleID, err := strconv.Atoi(roleIDString)
	if err != nil {
		log.Errorf("[RoleHandler - 4] UpdateRole: invalid role id")
		resp.Message = "invalid role id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	roleEntity := entity.RoleEntity{
		ID:   int64(roleID),
		Name: req.Name,
	}

	err = r.roleService.UpdateRole(ctx, roleEntity)
	if err != nil {
		if errors.Is(err, message.ErrRoleNotFound) {
			log.Errorf("[RoleHandler - 5] UpdateRole: %v", err)
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)
		}
		log.Errorf("[RoleHandler - 5] UpdateRole: %v", err)
		resp.Message = err.Error()
		resp.Data = nil
		return c.JSON(http.StatusInternalServerError, resp)
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}

// DeleteRoleByID godoc
// @Summary Delete role
// @Description Delete role by its ID
// @Tags roles
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Role ID"
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 409 {object} response.DefaultResponse "Conflict"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /admin/roles/{id} [delete]
func (r *roleHandler) DeleteRoleByID(c echo.Context) error {
	var (
		resp = response.DefaultResponse{}
		ctx  = c.Request().Context()
	)

	roleIDString := c.Param("id")
	if roleIDString == "" {
		log.Errorf("[RoleHandler - 1] DeleteRoleByID: missing role id")
		resp.Message = "missing role id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	roleID, err := strconv.Atoi(roleIDString)
	if err != nil {
		log.Errorf("[RoleHandler - 2] DeleteRoleByID: invalid role id")
		resp.Message = "invalid role id"
		resp.Data = nil
		return c.JSON(http.StatusBadRequest, resp)
	}

	err = r.roleService.DeleteRoleByID(ctx, int64(roleID))
	if err != nil {
		switch {
		case errors.Is(err, message.ErrRoleNotFound):
			log.Errorf("[RoleHandler - 3] DeleteRoleByID: %v", err)
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusNotFound, resp)

		case errors.Is(err, message.ErrRoleAssociated):
			log.Errorf("[RoleHandler - 4] DeleteRoleByID: %v", err)
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusConflict, resp)

		default:
			log.Errorf("[RoleHandler - 5] DeleteRoleByID: %v", err)
			resp.Message = err.Error()
			resp.Data = nil
			return c.JSON(http.StatusInternalServerError, resp)
		}
	}

	resp.Message = "Success"
	resp.Data = nil
	return c.JSON(http.StatusOK, resp)
}
