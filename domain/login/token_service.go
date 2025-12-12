package login

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var AccessTokenTTL = time.Minute * 15
var RefreshTokenTTL = time.Hour * 24 * 30

func GenerateAccessToken(employeeID string, role string) (string, error) {
	secret := os.Getenv("JWT_SECRET")

	claims := jwt.MapClaims{
		"sub":  employeeID,
		"role": role,
		"exp":  time.Now().Add(AccessTokenTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func GenerateRefreshTokenString() string {
	raw := uuid.New().String() + time.Now().String()
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func HashRefreshToken(rt string) string {
	sum := sha256.Sum256([]byte(rt))
	return hex.EncodeToString(sum[:])
}
