package service

import (
	"testing"
	"user-service/config"
	service2 "user-service/internal/core/service"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestJwtService(t *testing.T) {
	cfg := &config.Config{
		App: config.App{
			JwtSecretKey: "secret",
			JwtIssuer:    "test-issuer",
		},
	}
	jwtSvc := service2.NewJwtService(cfg)

	t.Run("Generate and Validate Token", func(t *testing.T) {
		userID := int64(123)
		role := "admin"

		tokenString, err := jwtSvc.GenerateToken(userID, role)
		assert.NoError(t, err)
		assert.NotEmpty(t, tokenString)

		token, err := jwtSvc.ValidateToken(tokenString)
		assert.NoError(t, err)
		assert.True(t, token.Valid)

		claims, ok := token.Claims.(jwt.MapClaims)
		assert.True(t, ok)
		assert.Equal(t, float64(userID), claims["user_id"])
		assert.Equal(t, role, claims["role"])
		assert.Equal(t, "test-issuer", claims["iss"])
	})

	t.Run("Invalid Token", func(t *testing.T) {
		token, err := jwtSvc.ValidateToken("invalid-token")
		assert.Error(t, err)
		assert.Nil(t, token)
	})

	t.Run("Invalid Issuer", func(t *testing.T) {
		otherCfg := &config.Config{
			App: config.App{
				JwtSecretKey: "secret",
				JwtIssuer:    "other-issuer",
			},
		}
		otherJwtSvc := service2.NewJwtService(otherCfg)
		tokenString, _ := otherJwtSvc.GenerateToken(1, "user")

		token, err := jwtSvc.ValidateToken(tokenString)
		assert.Error(t, err)
		assert.Equal(t, "invalid issuer", err.Error())
		assert.Nil(t, token)
	})
}
