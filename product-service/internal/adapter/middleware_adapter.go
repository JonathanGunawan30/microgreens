package adapter

import (
	"net/http"
	"product-service/config"
	"product-service/internal/adapter/handler/response"
	"product-service/internal/core/domain/entity"
	"product-service/utils/conv"
	"product-service/utils/jwt"
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
				respErr.Message = "missing token"
				respErr.Data = nil
				return c.JSON(http.StatusUnauthorized, respErr)
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			claims, err := jwt.ValidateToken(tokenString, secretKey)
			if err != nil {
				log.Errorf("[CheckToken-2] Invalid token: %v", err)
				respErr.Message = "invalid token"
				respErr.Data = nil
				return c.JSON(http.StatusUnauthorized, respErr)
			}

			userIDRaw, ok := claims["user_id"]
			if !ok || userIDRaw == nil {
				log.Errorf("[CheckToken-2b] user_id missing in JWT claims")
				respErr.Message = "invalid token"
				respErr.Data = nil
				return c.JSON(http.StatusUnauthorized, respErr)
			}

			userIDFloat, ok := userIDRaw.(float64)
			if !ok {
				log.Errorf("[CheckToken-2c] user_id is not number")
				respErr.Message = "invalid token"
				respErr.Data = nil
				return c.JSON(http.StatusUnauthorized, respErr)
			}

			userID := int64(userIDFloat)
			sessionKey := "session:" + strconv.FormatInt(userID, 10)

			session, err := m.redis.HGetAll(c.Request().Context(), sessionKey).Result()
			if err != nil || len(session) == 0 {
				log.Errorf("[CheckToken-3] Invalid token: %v", err)
				respErr.Message = "invalid token"
				respErr.Data = nil
				return c.JSON(http.StatusUnauthorized, respErr)
			}

			if session["token"] != tokenString {
				log.Errorf("[CheckToken-4] Token revoked")
				respErr.Message = "token revoked"
				respErr.Data = nil
				return c.JSON(http.StatusUnauthorized, respErr)
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
					respErr.Message = "unauthorized access"
					respErr.Data = nil
					return c.JSON(http.StatusForbidden, respErr)
				}
			}

			return next(c)

		}
	}
}
