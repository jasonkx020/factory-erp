package security

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const TokenTypeMQTT = "MQTT"

// MqttClaims short-lived CONNECT password for NanoMQ HTTP auth.
type MqttClaims struct {
	TokenType    string   `json:"typ"`
	Username     string   `json:"username"`
	ClientID     string   `json:"client_id"`
	Tenant       string   `json:"tenant"`
	Roles        []string `json:"roles"`
	UserID       int64    `json:"user_id"`
	jwt.RegisteredClaims
}

func IssueMqttToken(secret string, ttlSec int64, userID int64, username, clientID, tenant string, roles []string) (token string, expiresIn int64, err error) {
	username = strings.TrimSpace(username)
	clientID = strings.TrimSpace(clientID)
	tenant = strings.TrimSpace(tenant)
	if username == "" || clientID == "" {
		return "", 0, fmt.Errorf("mqtt token requires username and client_id")
	}
	if ttlSec <= 0 {
		ttlSec = 43200
	}
	now := time.Now()
	claims := MqttClaims{
		TokenType: TokenTypeMQTT,
		Username:  username,
		ClientID:  clientID,
		Tenant:    tenant,
		Roles:     roles,
		UserID:    userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(ttlSec) * time.Second)),
			Issuer:    "erp-api",
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString([]byte(secret))
	if err != nil {
		return "", 0, err
	}
	return signed, ttlSec, nil
}

func ParseMqttToken(secret, tokenStr string) (*MqttClaims, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &MqttClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	c, ok := tok.Claims.(*MqttClaims)
	if !ok || !tok.Valid {
		return nil, errors.New("invalid mqtt token")
	}
	if c.TokenType != TokenTypeMQTT {
		return nil, errors.New("not mqtt token")
	}
	return c, nil
}
