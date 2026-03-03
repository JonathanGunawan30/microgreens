package port

import (
	"context"
	"notification-service/internal/core/domain/entity"
)

type NotificationRepository interface {
	GetAll(ctx context.Context, query entity.NotifQueryString) ([]entity.NotificationEntity, int64, error)
	GetByID(ctx context.Context, notifID int64) (*entity.NotificationEntity, error)
	CreateNotification(ctx context.Context, notification entity.NotificationEntity) (int64, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
}
