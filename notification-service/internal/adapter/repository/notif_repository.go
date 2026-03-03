package repository

import (
	"context"
	"errors"
	"notification-service/internal/core/domain/entity"
	"notification-service/internal/core/domain/model"
	"notification-service/internal/core/port"
	msg "notification-service/utils/message"
	"strings"
	"time"

	"gorm.io/gorm"
)

type notifRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) port.NotificationRepository {
	return &notifRepository{db: db}
}

func (n *notifRepository) GetAll(ctx context.Context, query entity.NotifQueryString) ([]entity.NotificationEntity, int64, error) {
	var (
		modelNotif []model.Notification
		count      int64
	)

	db := n.db.WithContext(ctx).Model(&model.Notification{})

	if query.Search != "" {
		search := "%" + query.Search + "%"
		db = db.Where(
			"subject ILIKE ? OR message ILIKE ? OR status ILIKE ?",
			search, search, search,
		)
	}

	if query.UserID != 0 {
		db = db.Where("receiver_id = ?", query.UserID)
	}

	if query.IsRead {
		db = db.Where("read_at IS NOT NULL")
	}

	if err := db.Count(&count).Error; err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.Limit

	allowedOrder := map[string]string{
		"id":         "id",
		"subject":    "subject",
		"status":     "status",
		"send_at":    "send_at",
		"created_at": "created_at",
	}

	orderBy := "created_at"

	if val, ok := allowedOrder[query.OrderBy]; ok {
		orderBy = val
	}

	orderType := "DESC"
	if strings.ToUpper(query.OrderType) == "ASC" {
		orderType = "ASC"
	}

	order := orderBy + " " + orderType

	if err := db.
		Order(order).
		Limit(int(query.Limit)).
		Offset(int(offset)).
		Find(&modelNotif).Error; err != nil {
		return nil, 0, err
	}

	result := make([]entity.NotificationEntity, 0, len(modelNotif))

	for _, notif := range modelNotif {
		result = append(result, entity.NotificationEntity{
			ID:      notif.ID,
			Subject: notif.Subject,
			Status:  notif.Status,
			SendAt:  notif.SendAt,
		})
	}

	return result, count, nil
}

func (n *notifRepository) GetByID(ctx context.Context, notifID int64) (*entity.NotificationEntity, error) {
	modelNotif := model.Notification{}
	if err := n.db.WithContext(ctx).Select("id", "subject", "status", "send_at", "read_at", "message", "notification_type").
		Where("id = ?", notifID).First(&modelNotif).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, msg.ErrNotifNotFound
		}
		return nil, err
	}

	return &entity.NotificationEntity{
		ID:               modelNotif.ID,
		Message:          modelNotif.Message,
		NotificationType: modelNotif.NotificationType,
		Status:           modelNotif.Status,
		Subject:          modelNotif.Subject,
		SendAt:           modelNotif.SendAt,
		ReadAt:           modelNotif.ReadAt,
	}, nil
}

func (n *notifRepository) CreateNotification(ctx context.Context, notification entity.NotificationEntity) (int64, error) {
	now := time.Now()
	modelNotif := model.Notification{
		Message:          notification.Message,
		NotificationType: notification.NotificationType,
		Status:           notification.Status,
		ReceiverID:       notification.ReceiverID,
		Subject:          notification.Subject,
		ReceiverEmail:    notification.ReceiverEmail,
		SendAt:           &now,
		ReadAt:           notification.ReadAt,
	}

	if err := n.db.WithContext(ctx).Create(&modelNotif).Error; err != nil {
		return 0, err
	}

	return modelNotif.ID, nil
}

func (n *notifRepository) UpdateStatus(ctx context.Context, id int64, status string) error {
	return n.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("id = ?", id).
		Update("status", status).Error
}
