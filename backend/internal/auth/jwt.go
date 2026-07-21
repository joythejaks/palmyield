package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid access token")

type Claims struct {
	UserID        string `json:"user_id"`
	CooperativeID string `json:"cooperative_id"`
	Role          string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(secret, userID, cooperativeID, role string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:        userID,
		CooperativeID: cooperativeID,
		Role:          role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseAccessToken(secret, tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
