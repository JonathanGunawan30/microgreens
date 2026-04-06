package adapter

import (
	"net/http"
	"payment-service/config"
	"payment-service/internal/adapter/handler/response"
	"payment-service/internal/core/domain/entity"
	"payment-service/utils/conv"
	"payment-service/utils/jwt"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
)

type MiddlewareAdapter interface {
	CheckToken(secretKey string) echo.MiddlewareFunc
}

type middlewareAdapter struct {
	cfg   *config.Config
	redis *redis.Client
}

func NewMiddlewareAdapter(cfg *config.Config, redisClient *redis.Client) MiddlewareAdapter {
	return &middlewareAdapter{
		cfg:   cfg,
		redis: redisClient,
	}
}

func (m *middlewareAdapter) CheckToken(secretKey string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			respErr := response.DefaultResponse{}
			respErr.Data = nil
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				log.Errorf("[CheckToken-1] Missing token in header")
				return c.JSON(http.StatusUnauthorized, response.Error("missing token"))
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			claims, err := jwt.ValidateToken(tokenString, secretKey)
			if err != nil {
				log.Errorf("[CheckToken-2] Invalid token: %v", err)
				return c.JSON(http.StatusUnauthorized, response.Error("invalid token"))
			}

			userIDRaw, ok := claims["user_id"]
			if !ok || userIDRaw == nil {
				log.Errorf("[CheckToken-2b] user_id missing in JWT claims")
				return c.JSON(http.StatusUnauthorized, response.Error("invalid token"))
			}

			userIDFloat, ok := userIDRaw.(float64)
			if !ok {
				log.Errorf("[CheckToken-2c] user_id is not number")
				return c.JSON(http.StatusUnauthorized, response.Error("invalid token"))
			}

			userID := int64(userIDFloat)
			sessionKey := "session:" + strconv.FormatInt(userID, 10)

			if m.redis == nil {
				log.Errorf("[CheckToken-redis-nil] Redis client is nil")
				return c.JSON(http.StatusInternalServerError, response.Error("internal server error"))
			}

			session, err := m.redis.HGetAll(c.Request().Context(), sessionKey).Result()
			if err != nil || len(session) == 0 {
				log.Errorf("[CheckToken-3] Invalid token: %v", err)
				return c.JSON(http.StatusUnauthorized, response.Error("invalid token"))
			}

			if session["token"] != tokenString {
				log.Errorf("[CheckToken-4] Token revoked")
				return c.JSON(http.StatusUnauthorized, response.Error("token revoked"))
			}

			log.Infof("[CheckToken-5] Valid token")

			userIDStr := session["user_id"]
			userID, _ = conv.StringToInt64(userIDStr)

			createdAt, _ := time.Parse(time.RFC3339, session["created_at"])

			jwtUserData := entity.JwtUserData{
				ID:        userID,
				Name:      session["name"],
				Email:     session["email"],
				RoleName:  session["role_name"],
				LoggedIn:  session["logged_in"] == "true",
				Token:     session["token"],
				CreatedAt: createdAt,
			}

			c.Set("user", jwtUserData)

			currentPath := c.Request().URL.Path
			if strings.HasPrefix(currentPath, "/admin") {
				if jwtUserData.RoleName != "Super Admin" {
					log.Errorf("[CheckToken] Unauthorized access to admin path: %s by role: %s", currentPath, jwtUserData.RoleName)
					return c.JSON(http.StatusForbidden, response.Error("unauthorized access"))
				}
			}

			return next(c)

		}
	}
}
