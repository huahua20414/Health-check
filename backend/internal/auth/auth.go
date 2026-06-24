package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"health-checkup/backend/internal/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type Claims struct {
	SessionID string `json:"sid"`
	UserID    uint   `json:"uid"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

func IssueToken(ctx context.Context, redisClient *redis.Client, secret string, ttl time.Duration, user models.User) (string, error) {
	sessionID, err := randomID()
	if err != nil {
		return "", err
	}
	expiresAt := time.Now().Add(ttl)
	claims := Claims{
		SessionID: sessionID,
		UserID:    user.ID,
		Role:      user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.Itoa(int(user.ID)),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	key := SessionKey(sessionID)
	if err := redisClient.Set(ctx, key, strconv.Itoa(int(user.ID)), ttl).Err(); err != nil {
		return "", err
	}
	return token, nil
}

func ParseToken(tokenText, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenText, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func SessionKey(sessionID string) string {
	return "session:" + sessionID
}

func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
