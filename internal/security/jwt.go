package security

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Claims struct {
	UserID      int64    `json:"user_id"`
	LoginName   string   `json:"login_name"`
	UserType    string   `json:"user_type"`
	ClientType  string   `json:"client_type"`
	Roles       []string `json:"roles,omitempty"`
	Permissions []string `json:"permissions,omitempty"` // runtime-hydrated; omitted from new JWTs
	jwt.RegisteredClaims
}

// NormalizeClientType maps legacy client labels onto the canonical set:
// admin | boss | employee | mobile | customer
func NormalizeClientType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "admin", "web":
		return "admin"
	case "boss":
		return "boss"
	case "mobile":
		return "mobile"
	case "customer", "portal", "shop":
		return "customer"
	case "employee", "pad", "mp_worker", "mp_sales", "front":
		return "employee"
	case "":
		return "admin"
	default:
		return "employee"
	}
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func IssueToken(secret string, ttlMin int, claims Claims) (string, error) {
	now := time.Now()
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ID:        uuid.NewString(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(ttlMin) * time.Minute)),
		Issuer:    "erp-api",
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

func ParseToken(secret, tokenStr string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	c, ok := tok.Claims.(*Claims)
	if !ok || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	return c, nil
}
