package service

import (
	"context"
	"fmt"
	"math"
	"notification-service/internal/core/domain/entity"
	"notification-service/internal/core/port"
	"notification-service/utils/constant"
	"notification-service/utils/conv"
	"notification-service/utils/websocket"
	"strings"
)

type NotificationService struct {
	emailSender port.EmailSender
	repo        port.NotificationRepository
}

func NewNotificationService(emailSender port.EmailSender, repo port.NotificationRepository) *NotificationService {
	return &NotificationService{
		emailSender: emailSender,
		repo:        repo,
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
		return fmt.Errorf("websocket connection not found")
	}

	msg := map[string]any{
		"type":    notif.NotificationType,
		"subject": notif.Subject,
		"message": notif.Message,
		"send_at": notif.SendAt,
	}

	return conn.WriteJSON(msg)
}

func (n *NotificationService) GetAll(ctx context.Context, query entity.NotifQueryString) ([]entity.NotificationEntity, int64, int64, error) {
	data, totalRows, err := n.repo.GetAll(ctx, query)
	if err != nil {
		return nil, 0, 0, err
	}

	totalPages := int64(math.Ceil(float64(totalRows) / float64(query.Limit)))

	return data, totalRows, totalPages, nil
}

func (n *NotificationService) GetByID(ctx context.Context, notifID int64) (*entity.NotificationEntity, error) {
	return n.repo.GetByID(ctx, notifID)
}
