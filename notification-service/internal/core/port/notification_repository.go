package port

import (
	"context"
	"notification-service/internal/core/domain/entity"
)

type NotificationRepository interface {
	GetAll(ctx context.Context, query entity.NotifQueryString, userID int64) ([]entity.NotificationEntity, int64, error)
	GetByID(ctx context.Context, notifID, userID int64) (*entity.NotificationEntity, error)
	CreateNotification(ctx context.Context, notification entity.NotificationEntity) (int64, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	ReadNotificationByID(ctx context.Context, id, userID int64) error
	ReadAllNotifications(ctx context.Context, userID int64) error
}
