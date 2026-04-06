package service_test

import (
	"context"
	"errors"
	"notification-service/config"
	"notification-service/internal/core/domain/entity"
	"notification-service/internal/core/service"
	"notification-service/tests/mocks"
	"notification-service/utils/constant"
	"notification-service/utils/conv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func int64Ptr(i int64) *int64 {
	return &i
}

func TestNotificationService_SendEmailNotif(t *testing.T) {
	mockEmailSender := mocks.NewEmailSender(t)
	mockRepo := mocks.NewNotificationRepository(t)
	cfg := &config.Config{}

	svc := service.NewNotificationService(mockEmailSender, mockRepo, cfg)

	t.Run("Success", func(t *testing.T) {
		mockEmailSender.On("SendEmailNotif", "test@example.com", "Subject", "Body").Return(nil).Once()

		err := svc.SendEmailNotif("test@example.com", "Subject", "Body")

		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		mockEmailSender.On("SendEmailNotif", "test@example.com", "Subject", "Body").Return(errors.New("smtp error")).Once()

		err := svc.SendEmailNotif("test@example.com", "Subject", "Body")

		assert.Error(t, err)
		assert.Equal(t, "smtp error", err.Error())
	})
}

func TestNotificationService_ProcessNotification(t *testing.T) {
	mockEmailSender := mocks.NewEmailSender(t)
	mockRepo := mocks.NewNotificationRepository(t)
	cfg := &config.Config{}
	svc := service.NewNotificationService(mockEmailSender, mockRepo, cfg)

	ctx := context.Background()
	notif := entity.NotificationEntity{
		Message:          "Hello",
		NotificationType: constant.TypeEmail,
		ReceiverEmail:    conv.StrPtr("test@example.com"),
		Subject:          conv.StrPtr("Greeting"),
	}

	t.Run("Success Email", func(t *testing.T) {
		mockRepo.On("CreateNotification", ctx, mock.MatchedBy(func(n entity.NotificationEntity) bool {
			return *n.Status == constant.StatusPending
		})).Return(int64(1), nil).Once()

		mockEmailSender.On("SendEmailNotif", "test@example.com", "Greeting", "Hello").Return(nil).Once()

		mockRepo.On("UpdateStatus", ctx, int64(1), constant.StatusSent).Return(nil).Once()

		err := svc.ProcessNotification(ctx, notif)

		assert.NoError(t, err)
	})

	t.Run("Create Error", func(t *testing.T) {
		mockRepo.On("CreateNotification", ctx, mock.MatchedBy(func(n entity.NotificationEntity) bool {
			return *n.Status == constant.StatusPending
		})).Return(int64(0), errors.New("db error")).Once()

		err := svc.ProcessNotification(ctx, notif)

		assert.Error(t, err)
		assert.Equal(t, "db error", err.Error())
	})

	t.Run("Unsupported Type", func(t *testing.T) {
		invalidNotif := notif
		invalidNotif.NotificationType = "INVALID"

		mockRepo.On("CreateNotification", ctx, mock.MatchedBy(func(n entity.NotificationEntity) bool {
			return *n.Status == constant.StatusPending
		})).Return(int64(2), nil).Once()

		mockRepo.On("UpdateStatus", ctx, int64(2), constant.StatusFailed).Return(nil).Once()

		err := svc.ProcessNotification(ctx, invalidNotif)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported notification type")
	})

	t.Run("Send Error", func(t *testing.T) {
		mockRepo.On("CreateNotification", ctx, mock.MatchedBy(func(n entity.NotificationEntity) bool {
			return *n.Status == constant.StatusPending
		})).Return(int64(3), nil).Once()

		mockEmailSender.On("SendEmailNotif", "test@example.com", "Greeting", "Hello").Return(errors.New("send error")).Once()

		mockRepo.On("UpdateStatus", ctx, int64(3), constant.StatusFailed).Return(nil).Once()

		err := svc.ProcessNotification(ctx, notif)

		assert.Error(t, err)
		assert.Equal(t, "send error", err.Error())
	})

	t.Run("Email Nil Fields", func(t *testing.T) {
		nilEmailNotif := notif
		nilEmailNotif.ReceiverEmail = nil

		mockRepo.On("CreateNotification", ctx, mock.Anything).Return(int64(4), nil).Once()
		mockRepo.On("UpdateStatus", ctx, int64(4), constant.StatusFailed).Return(nil).Once()

		err := svc.ProcessNotification(ctx, nilEmailNotif)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "receiver email is nil")
	})

	t.Run("Success Push", func(t *testing.T) {
		pushNotif := entity.NotificationEntity{
			Message:          "Push",
			NotificationType: constant.TypePush,
			ReceiverID:       int64Ptr(1),
		}

		mockRepo.On("CreateNotification", ctx, mock.Anything).Return(int64(5), nil).Once()
		mockRepo.On("UpdateStatus", ctx, int64(5), constant.StatusSent).Return(nil).Once()

		err := svc.ProcessNotification(ctx, pushNotif)
		assert.NoError(t, err)
	})

	t.Run("Push Nil ReceiverID", func(t *testing.T) {
		pushNotif := entity.NotificationEntity{
			Message:          "Push",
			NotificationType: constant.TypePush,
			ReceiverID:       nil,
		}

		mockRepo.On("CreateNotification", ctx, mock.Anything).Return(int64(6), nil).Once()
		mockRepo.On("UpdateStatus", ctx, int64(6), constant.StatusFailed).Return(nil).Once()

		err := svc.ProcessNotification(ctx, pushNotif)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "receiver id is nil")
	})
}

func TestNotificationService_GetAll(t *testing.T) {
	mockEmailSender := mocks.NewEmailSender(t)
	mockRepo := mocks.NewNotificationRepository(t)
	cfg := &config.Config{}
	svc := service.NewNotificationService(mockEmailSender, mockRepo, cfg)

	ctx := context.Background()
	query := entity.NotifQueryString{Limit: 10, Page: 1}
	userID := int64(100)

	t.Run("Success", func(t *testing.T) {
		expectedData := []entity.NotificationEntity{{ID: 1}}
		mockRepo.On("GetAll", ctx, query, userID).Return(expectedData, int64(15), nil).Once()

		data, totalRows, totalPages, err := svc.GetAll(ctx, query, userID)

		assert.NoError(t, err)
		assert.Equal(t, expectedData, data)
		assert.Equal(t, int64(15), totalRows)
		assert.Equal(t, int64(2), totalPages) // 15 / 10 = 1.5 -> 2
	})

	t.Run("Zero Limit", func(t *testing.T) {
		queryZero := entity.NotifQueryString{Limit: 0, Page: 1}
		mockRepo.On("GetAll", ctx, queryZero, userID).Return([]entity.NotificationEntity{}, int64(15), nil).Once()

		_, _, totalPages, err := svc.GetAll(ctx, queryZero, userID)

		assert.NoError(t, err)
		assert.Equal(t, int64(0), totalPages)
	})

	t.Run("Error", func(t *testing.T) {
		mockRepo.On("GetAll", ctx, query, userID).Return(nil, int64(0), errors.New("db error")).Once()

		data, totalRows, totalPages, err := svc.GetAll(ctx, query, userID)

		assert.Error(t, err)
		assert.Nil(t, data)
		assert.Equal(t, int64(0), totalRows)
		assert.Equal(t, int64(0), totalPages)
	})
}

func TestNotificationService_GetByID(t *testing.T) {
	mockEmailSender := mocks.NewEmailSender(t)
	mockRepo := mocks.NewNotificationRepository(t)
	cfg := &config.Config{}
	svc := service.NewNotificationService(mockEmailSender, mockRepo, cfg)

	ctx := context.Background()
	notifID := int64(1)
	userID := int64(100)

	t.Run("Success", func(t *testing.T) {
		expectedNotif := &entity.NotificationEntity{ID: 1}
		mockRepo.On("GetByID", ctx, notifID, userID).Return(expectedNotif, nil).Once()

		data, err := svc.GetByID(ctx, notifID, userID)

		assert.NoError(t, err)
		assert.Equal(t, expectedNotif, data)
	})

	t.Run("Error", func(t *testing.T) {
		mockRepo.On("GetByID", ctx, notifID, userID).Return(nil, errors.New("not found")).Once()

		data, err := svc.GetByID(ctx, notifID, userID)

		assert.Error(t, err)
		assert.Nil(t, data)
	})
}

func TestNotificationService_GetAdmin(t *testing.T) {
	cfg := &config.Config{
		App: config.App{
			AdminID:    1,
			AdminEmail: "admin@example.com",
		},
	}
	svc := service.NewNotificationService(nil, nil, cfg)

	admin := svc.GetAdmin()

	assert.Equal(t, int64(1), admin.ID)
	assert.Equal(t, "admin@example.com", admin.Email)
}

func TestNotificationService_ReadNotificationByID(t *testing.T) {
	mockRepo := mocks.NewNotificationRepository(t)
	svc := service.NewNotificationService(nil, mockRepo, nil)

	ctx := context.Background()
	id := int64(1)
	userID := int64(100)

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("ReadNotificationByID", ctx, id, userID).Return(nil).Once()
		err := svc.ReadNotificationByID(ctx, id, userID)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		mockRepo.On("ReadNotificationByID", ctx, id, userID).Return(errors.New("db error")).Once()
		err := svc.ReadNotificationByID(ctx, id, userID)
		assert.Error(t, err)
	})
}

func TestNotificationService_ReadAllNotifications(t *testing.T) {
	mockRepo := mocks.NewNotificationRepository(t)
	svc := service.NewNotificationService(nil, mockRepo, nil)

	ctx := context.Background()
	userID := int64(100)

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("ReadAllNotifications", ctx, userID).Return(nil).Once()
		err := svc.ReadAllNotifications(ctx, userID)
		assert.NoError(t, err)
	})

	t.Run("Error", func(t *testing.T) {
		mockRepo.On("ReadAllNotifications", ctx, userID).Return(errors.New("db error")).Once()
		err := svc.ReadAllNotifications(ctx, userID)
		assert.Error(t, err)
	})
}
