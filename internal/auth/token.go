package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
)

type TokenClaims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

type TokenService struct {
	secret []byte
}

func NewTokenService(secret string) *TokenService {
	return &TokenService{secret: []byte(secret)}
}

func (s *TokenService) CreateAccessToken(userID uuid.UUID) (string, error) {
	return s.createToken(userID, 15*time.Minute)
}

func (s *TokenService) CreateRefreshToken(userID uuid.UUID) (string, error) {
	return s.createToken(userID, 7*24*time.Hour)
}

func (s *TokenService) createToken(userID uuid.UUID, duration time.Duration) (string, error) {
	claims := TokenClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *TokenService) ValidateToken(tokenStr string) (uuid.UUID, error) {
	// Reject empty tokens early
	if strings.TrimSpace(tokenStr) == "" {
		return uuid.Nil, ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(tokenStr, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
	// Explicitly validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})

	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		// Validate that UserID is not empty
		if claims.UserID == uuid.Nil {
			return uuid.Nil, ErrInvalidToken
		}
		return claims.UserID, nil
	}

	return uuid.Nil, ErrInvalidToken
}
