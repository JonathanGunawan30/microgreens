package handler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"notification-service/internal/adapter/handler"
	"notification-service/internal/core/domain/entity"
	"notification-service/tests/mocks"
	msg "notification-service/utils/message"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupTestEcho() *echo.Echo {
	return echo.New()
}

func TestNotificationHandler_GetAll(t *testing.T) {
	e := setupTestEcho()
	mockSvc := mocks.NewNotificationService(t)
	h := handler.NewNotificationHandler(mockSvc)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/notifications?page=1&limit=10", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", entity.JwtUserData{ID: 100})

		mockSvc.On("GetAll", mock.Anything, mock.MatchedBy(func(q entity.NotifQueryString) bool {
			return q.Page == 1 && q.Limit == 10
		}), int64(100)).Return([]entity.NotificationEntity{
			{
				ID:      1,
				Subject: strPtr("Subject"),
				Message: "Message",
				Status:  strPtr("SENT"),
				SendAt:  timePtr(time.Now()),
			},
		}, int64(1), int64(1), nil).Once()

		if assert.NoError(t, h.GetAll(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/notifications", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		// No user set in context

		if assert.NoError(t, h.GetAll(c)) {
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		}
	})

	t.Run("Internal Error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/notifications", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", entity.JwtUserData{ID: 100})

		mockSvc.On("GetAll", mock.Anything, mock.Anything, int64(100)).Return(nil, int64(0), int64(0), errors.New("error")).Once()

		if assert.NoError(t, h.GetAll(c)) {
			assert.Equal(t, http.StatusInternalServerError, rec.Code)
		}
	})

	t.Run("Bind Error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/notifications?page=abc", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		if assert.NoError(t, h.GetAll(c)) {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		}
	})
}

func TestNotificationHandler_GetByID(t *testing.T) {
	e := setupTestEcho()
	mockSvc := mocks.NewNotificationService(t)
	h := handler.NewNotificationHandler(mockSvc)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/notifications/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")
		c.Set("user", entity.JwtUserData{ID: 100})

		mockSvc.On("GetByID", mock.Anything, int64(1), int64(100)).Return(&entity.NotificationEntity{
			ID:               1,
			Subject:          strPtr("Subject"),
			Message:          "Message",
			Status:           strPtr("SENT"),
			NotificationType: "EMAIL",
			SendAt:           timePtr(time.Now()),
			ReadAt:           timePtr(time.Now()),
		}, nil).Once()

		if assert.NoError(t, h.GetByID(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("Invalid ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/notifications/abc", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")

		if assert.NoError(t, h.GetByID(c)) {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/notifications/1", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")
		c.Set("user", entity.JwtUserData{ID: 100})

		mockSvc.On("GetByID", mock.Anything, int64(1), int64(100)).Return(nil, msg.ErrNotifNotFound).Once()

		if assert.NoError(t, h.GetByID(c)) {
			assert.Equal(t, http.StatusNotFound, rec.Code)
		}
	})
}

func TestNotificationHandler_Read(t *testing.T) {
	e := setupTestEcho()
	mockSvc := mocks.NewNotificationService(t)
	h := handler.NewNotificationHandler(mockSvc)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/auth/notifications/1/read", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")
		c.Set("user", entity.JwtUserData{ID: 100})

		mockSvc.On("ReadNotificationByID", mock.Anything, int64(1), int64(100)).Return(nil).Once()

		if assert.NoError(t, h.Read(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/auth/notifications/1/read", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("1")
		c.Set("user", entity.JwtUserData{ID: 100})

		mockSvc.On("ReadNotificationByID", mock.Anything, int64(1), int64(100)).Return(msg.ErrNotifNotFound).Once()

		if assert.NoError(t, h.Read(c)) {
			assert.Equal(t, http.StatusNotFound, rec.Code)
		}
	})
}

func TestNotificationHandler_ReadAll(t *testing.T) {
	e := setupTestEcho()
	mockSvc := mocks.NewNotificationService(t)
	h := handler.NewNotificationHandler(mockSvc)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/auth/notifications/read-all", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("user", entity.JwtUserData{ID: 100})

		mockSvc.On("ReadAllNotifications", mock.Anything, int64(100)).Return(nil).Once()

		if assert.NoError(t, h.ReadAll(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})
}

func strPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}
