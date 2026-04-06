package handler

import (
	"errors"
	"net/http"
	"notification-service/internal/adapter/handler/response"
	"notification-service/internal/core/domain/entity"
	"notification-service/internal/core/port"
	"notification-service/utils/conv"
	msg "notification-service/utils/message"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

type NotificationHandler struct {
	service port.NotificationService
}

func NewNotificationHandler(service port.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		service: service,
	}
}

// GetAll godoc
// @Summary Get all notifications
// @Description Get paginated list of notifications for authenticated user
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 10)"
// @Param order_by query string false "Order by field (default: created_at)"
// @Param order_type query string false "Order type ASC or DESC (default: desc)"
// @Success 200 {object} response.DefaultResponseWithPagination{data=[]response.ListResponse} "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /auth/notifications [get]
func (h *NotificationHandler) GetAll(c echo.Context) error {
	var (
		ctx               = c.Request().Context()
		resp              []response.ListResponse
		notificationQuery entity.NotifQueryString
	)

	if err := c.Bind(&notificationQuery); err != nil {
		return c.JSON(http.StatusBadRequest, response.Error("Invalid Request"))
	}

	if notificationQuery.Page <= 0 {
		notificationQuery.Page = 1
	}

	if notificationQuery.Limit <= 0 {
		notificationQuery.Limit = 10
	}

	if notificationQuery.OrderBy == "" {
		notificationQuery.OrderBy = "created_at"
	}

	if notificationQuery.OrderType == "" {
		notificationQuery.OrderType = "desc"
	}

	user, ok := c.Get("user").(entity.JwtUserData)
	if !ok || user.ID == 0 {
		return c.JSON(http.StatusUnauthorized, response.Error("Unauthorized"))
	}

	data, count, totalPages, err := h.service.GetAll(ctx, notificationQuery, user.ID)
	if err != nil {
		log.Errorf("[NotificationService] Error: %v", err)
		return c.JSON(http.StatusInternalServerError, response.Error("Internal Server Error"))
	}

	for _, notif := range data {
		var readAt *string
		if notif.ReadAt != nil {
			formatted := notif.ReadAt.Format("2006-01-02 15:04:05")
			readAt = &formatted
		}

		resp = append(resp, response.ListResponse{
			ID:      notif.ID,
			Subject: *notif.Subject,
			Message: notif.Message,
			Status:  *notif.Status,
			SendAt:  notif.SendAt.Format("2006-01-02 15:04:05"),
			ReadAt:  readAt,
		})
	}

	return c.JSON(http.StatusOK, response.SuccessWithPagination("Success", resp, notificationQuery.Page, count, notificationQuery.Limit, totalPages))

}

// GetByID godoc
// @Summary Get notification by ID
// @Description Get notification detail by ID for authenticated user
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Notification ID"
// @Success 200 {object} response.DefaultResponse{data=response.DetailResponse} "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /auth/notifications/{id} [get]
func (h *NotificationHandler) GetByID(c echo.Context) error {
	var (
		ctx        = c.Request().Context()
		respDetail response.DetailResponse
	)

	idStr := c.Param("id")
	notifID, err := conv.StringToInt64(idStr)
	if err != nil || notifID <= 0 {
		log.Errorf("[NotificationHandler] GetByID: %v", err)
		return c.JSON(http.StatusBadRequest, response.Error("Invalid Notification ID"))
	}

	user, ok := c.Get("user").(entity.JwtUserData)
	if !ok || user.ID == 0 {
		return c.JSON(http.StatusUnauthorized, response.Error("Unauthorized"))
	}

	notificationEntity, err := h.service.GetByID(ctx, notifID, user.ID)
	if err != nil {
		log.Errorf("[NotificationHandler] GetByID: %v", err)

		if errors.Is(err, msg.ErrNotifNotFound) {
			return c.JSON(http.StatusNotFound, response.Error("Notification Not Found"))
		}

		return c.JSON(http.StatusInternalServerError, response.Error("Internal Server Error"))
	}

	respDetail.ID = notificationEntity.ID
	respDetail.Message = notificationEntity.Message
	respDetail.NotificationType = notificationEntity.NotificationType
	respDetail.Status = *notificationEntity.Status
	respDetail.Subject = *notificationEntity.Subject
	respDetail.SendAt = notificationEntity.SendAt.Format("2006-01-02 15:04:05")
	respDetail.ReadAt = notificationEntity.ReadAt.Format("2006-01-02 15:04:05")

	return c.JSON(http.StatusOK, response.Success("Success", respDetail))
}

// Read godoc
// @Summary Mark notification as read
// @Description Mark a specific notification as read for authenticated user
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Notification ID"
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 400 {object} response.DefaultResponse "Bad Request"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /auth/notifications/{id}/read [patch]
func (h *NotificationHandler) Read(c echo.Context) error {
	ctx := c.Request().Context()

	idStr := c.Param("id")
	id, err := conv.StringToInt64(idStr)
	if err != nil || id <= 0 {
		log.Errorf("[Notification Handler] Read: %v", err)
		return c.JSON(http.StatusBadRequest, response.Error("Invalid Notification ID"))
	}

	user, ok := c.Get("user").(entity.JwtUserData)
	if !ok || user.ID == 0 {
		return c.JSON(http.StatusUnauthorized, response.Error("Unauthorized"))
	}

	err = h.service.ReadNotificationByID(ctx, id, user.ID)
	if err != nil {
		log.Errorf("[Notification Handler] Read: %v", err)

		if errors.Is(err, msg.ErrNotifNotFound) {
			return c.JSON(http.StatusNotFound, response.Error("Notification Not Found"))
		}

		return c.JSON(http.StatusInternalServerError, response.Error("Internal Server Error"))
	}

	return c.JSON(http.StatusOK, response.Success("Success", nil))
}

// ReadAll godoc
// @Summary Mark all notifications as read
// @Description Mark all notifications as read for authenticated user
// @Tags notifications
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.DefaultResponse "Success"
// @Failure 401 {object} response.DefaultResponse "Unauthorized"
// @Failure 404 {object} response.DefaultResponse "Not Found"
// @Failure 500 {object} response.DefaultResponse "Internal Server Error"
// @Router /auth/notifications/read-all [patch]
func (h *NotificationHandler) ReadAll(c echo.Context) error {
	ctx := c.Request().Context()

	user, ok := c.Get("user").(entity.JwtUserData)
	if !ok || user.ID == 0 {
		return c.JSON(http.StatusUnauthorized, response.Error("Unauthorized"))
	}

	err := h.service.ReadAllNotifications(ctx, user.ID)
	if err != nil {
		log.Errorf("[Notification Handler] ReadAll: %v", err)

		if errors.Is(err, msg.ErrNotifNotFound) {
			return c.JSON(http.StatusNotFound, response.Error("Notification Not Found"))
		}

		return c.JSON(http.StatusInternalServerError, response.Error("Internal Server Error"))
	}

	return c.JSON(http.StatusOK, response.Success("Success", nil))
}
