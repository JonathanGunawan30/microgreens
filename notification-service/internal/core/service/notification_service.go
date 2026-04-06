package service

import (
	"context"
	"fmt"
	"math"
	"notification-service/config"
	"notification-service/internal/core/domain/entity"
	"notification-service/internal/core/port"
	"notification-service/utils/constant"
	"notification-service/utils/conv"
	"notification-service/utils/websocket"
	"strings"

	"github.com/labstack/gommon/log"
)

type NotificationService struct {
	emailSender port.EmailSender
	repo        port.NotificationRepository
	cfg         *config.Config
}

func NewNotificationService(emailSender port.EmailSender, repo port.NotificationRepository, cfg *config.Config) *NotificationService {
	return &NotificationService{
		emailSender: emailSender,
		repo:        repo,
		cfg:         cfg,
	}
}

func (n *NotificationService) SendEmailNotif(to string, subject string, body string) error {
	return n.emailSender.SendEmailNotif(to, subject, body)
}

func (n *NotificationService) ProcessNotification(ctx context.Context, notif entity.NotificationEntity) error {
	notif.Status = conv.StrPtr(constant.StatusPending)

	id, err := n.repo.CreateNotification(ctx, notif)
	if err != nil {
		return err
	}

	var sendErr error

	switch strings.ToUpper(notif.NotificationType) {

	case constant.TypeEmail:
		sendErr = n.sendEmail(notif)

	case constant.TypePush:
		sendErr = n.sendPush(notif)

	default:
		sendErr = fmt.Errorf("unsupported notification type")
	}

	if sendErr != nil {
		_ = n.repo.UpdateStatus(ctx, id, constant.StatusFailed)
		return sendErr
	}

	return n.repo.UpdateStatus(ctx, id, constant.StatusSent)
}

func (n *NotificationService) sendEmail(notif entity.NotificationEntity) error {
	if notif.ReceiverEmail == nil {
		return fmt.Errorf("receiver email is nil")
	}

	if notif.Subject == nil {
		return fmt.Errorf("subject is nil")
	}

	return n.emailSender.SendEmailNotif(
		*notif.ReceiverEmail,
		*notif.Subject,
		notif.Message,
	)
}

func (n *NotificationService) sendPush(notif entity.NotificationEntity) error {
	if notif.ReceiverID == nil {
		return fmt.Errorf("receiver id is nil")
	}

	conn := websocket.GetWebSocketConn(*notif.ReceiverID)
	if conn == nil {
		log.Infof("User %d is offline, skipping real-time push", *notif.ReceiverID)
		return nil
	}

	msg := map[string]any{
		"type":    notif.NotificationType,
		"subject": notif.Subject,
		"message": notif.Message,
		"send_at": notif.SendAt,
	}

	return conn.WriteJSON(msg)
}

func (n *NotificationService) GetAll(ctx context.Context, query entity.NotifQueryString, userID int64) ([]entity.NotificationEntity, int64, int64, error) {
	data, totalRows, err := n.repo.GetAll(ctx, query, userID)
	if err != nil {
		return nil, 0, 0, err
	}

	var totalPages int64
	if query.Limit > 0 {
		totalPages = int64(math.Ceil(float64(totalRows) / float64(query.Limit)))
	}

	return data, totalRows, totalPages, nil
}

func (n *NotificationService) GetByID(ctx context.Context, notifID, userID int64) (*entity.NotificationEntity, error) {
	return n.repo.GetByID(ctx, notifID, userID)
}

func (n *NotificationService) GetAdmin() entity.AdminEntity {
	return entity.AdminEntity{
		ID:    n.cfg.App.AdminID,
		Email: n.cfg.App.AdminEmail,
	}
}

func (n *NotificationService) ReadNotificationByID(ctx context.Context, id, userID int64) error {
	return n.repo.ReadNotificationByID(ctx, id, userID)
}

func (n *NotificationService) ReadAllNotifications(ctx context.Context, userID int64) error {
	return n.repo.ReadAllNotifications(ctx, userID)
}
