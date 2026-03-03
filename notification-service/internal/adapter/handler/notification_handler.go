package handler

import (
	"errors"
	"net/http"
	"notification-service/internal/adapter/handler/response"
	"notification-service/internal/core/domain/entity"
	"notification-service/internal/core/service"
	"notification-service/utils/conv"
	msg "notification-service/utils/message"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

type NotificationHandler struct {
	service *service.NotificationService
}

func NewNotificationHandler(service *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		service: service,
	}
}

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

	data, count, totalPages, err := h.service.GetAll(ctx, notificationQuery)
	if err != nil {
		log.Errorf("[NotificationService] Error: %v", err)
		return c.JSON(http.StatusInternalServerError, response.Error("Internal Server Error"))
	}

	for _, notif := range data {
		resp = append(resp, response.ListResponse{
			ID:      notif.ID,
			Subject: *notif.Subject,
			Status:  *notif.Status,
			SendAt:  notif.SendAt.Format("2006-01-02 15:04:05"),
		})
	}

	return c.JSON(http.StatusOK, response.SuccessWithPagination("Success", resp, notificationQuery.Page, count, notificationQuery.Limit, totalPages))

}

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

	notificationEntity, err := h.service.GetByID(ctx, notifID)
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
