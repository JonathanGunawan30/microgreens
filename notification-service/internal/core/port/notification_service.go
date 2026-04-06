package port

import (
	"context"
	"notification-service/internal/core/domain/entity"
)

type NotificationService interface {
	SendEmailNotif(to string, subject string, body string) error
	ProcessNotification(ctx context.Context, notif entity.NotificationEntity) error
	GetAll(ctx context.Context, query entity.NotifQueryString, userID int64) ([]entity.NotificationEntity, int64, int64, error)
	GetByID(ctx context.Context, notifID, userID int64) (*entity.NotificationEntity, error)
	GetAdmin() entity.AdminEntity
	ReadNotificationByID(ctx context.Context, id, userID int64) error
	ReadAllNotifications(ctx context.Context, userID int64) error
}
