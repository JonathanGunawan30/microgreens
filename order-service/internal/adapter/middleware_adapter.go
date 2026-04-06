package adapter

import (
	"bytes"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"order-service/config"
	"order-service/internal/adapter/handler/response"
	"order-service/internal/core/domain/entity"
	"order-service/utils/conv"
	"order-service/utils/jwt"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"github.com/redis/go-redis/v9"
)

type MiddlewareAdapter interface {
	CheckToken(secretKey string) echo.MiddlewareFunc
	DistanceCheck() echo.MiddlewareFunc
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
				log.Errorf("[CheckToken] Redis client is nil")
				return c.JSON(http.StatusUnauthorized, response.Error("invalid token"))
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

func (m *middlewareAdapter) DistanceCheck() echo.MiddlewareFunc {
	latRef, errRef1 := strconv.ParseFloat(m.cfg.App.LatitudeRef, 64)
	lngRef, errRef2 := strconv.ParseFloat(m.cfg.App.LongitudeRef, 64)

	if errRef1 != nil || errRef2 != nil {
		log.Panic("Config LatitudeRef/LongitudeRef invalid!")
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			bodyBytes, err := io.ReadAll(c.Request().Body)
			if err != nil {
				log.Errorf("[MIddlewareAdapter] DIstanceCheck: Failed to ready body: %v", err)
				return c.JSON(http.StatusBadRequest, response.Error("invalid request body"))
			}

			c.Request().Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			var payload struct {
				ShippingType string `json:"shipping_type"`
			}

			_ = json.Unmarshal(bodyBytes, &payload)

			if strings.EqualFold(payload.ShippingType, "pickup") {
				return next(c)
			}

			latParam := c.QueryParam("lat")
			lngParam := c.QueryParam("lng")
			if latParam == "" || lngParam == "" {
				log.Errorf("[MiddlewareAdapter - 1] DistanceCheck: %s", "missing or invalid lat or lng")
				return c.JSON(http.StatusBadRequest, response.Error("missing or invalid lat or lng"))
			}

			lat, err1 := strconv.ParseFloat(latParam, 64)
			lng, err2 := strconv.ParseFloat(lngParam, 64)

			if err1 != nil || err2 != nil {
				log.Errorf("[MiddlewareAdapter - 2] DistanceCheck: %s", "missing or invalid lat or lng")
				return c.JSON(http.StatusBadRequest, response.Error("missing or invalid lat or lng"))
			}

			distance := m.HaversineDistance(latRef, lngRef, lat, lng)
			if distance > float64(m.cfg.App.MaxDistance) {
				log.Errorf("[MiddlewareAdapter - 3] DistanceCheck: %s", "distance too far")
				return c.JSON(http.StatusBadRequest, response.Error("oops distance too far"))
			}

			return next(c)

		}
	}
}

func toRadians(d float64) float64 {
	return d * math.Pi / 180
}

func (m *middlewareAdapter) HaversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371

	dLat := toRadians(lat2 - lat1)
	dLng := toRadians(lng2 - lng1)

	radLat1 := toRadians(lat1)
	radLat2 := toRadians(lat2)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(radLat1)*math.Cos(radLat2)*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
