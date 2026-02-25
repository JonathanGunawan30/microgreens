package service

import (
	"errors"
	"time"
	"user-service/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/gommon/log"
)

type JwtServiceInterface interface {
	GenerateToken(payload int64, role string) (string, error)
	ValidateToken(token string) (*jwt.Token, error)
}

type jwtService struct {
	secretKey string
	issuer    string
}

func NewJwtService(cfg *config.Config) JwtServiceInterface {
	return &jwtService{
		secretKey: cfg.App.JwtSecretKey,
		issuer:    cfg.App.JwtIssuer,
	}
}

func (j *jwtService) GenerateToken(userID int64, role string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"iss":     j.issuer,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(j.secretKey))
	if err != nil {
		log.Errorf("[GenerateToken] Error Generate JWT Token with error: %v", err)
		return "", err
	}

	return tokenString, nil
}

func (j *jwtService) ValidateToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(j.secretKey), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		log.Errorf("[ValidateToken] Error Validate JWT Token with error: %v", err)
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if claims["iss"] != j.issuer {
			return nil, errors.New("invalid issuer")
		}
		return token, nil
	}

	return nil, errors.New("invalid token")
}
