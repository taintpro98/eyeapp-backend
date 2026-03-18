package auth

import (
	"errors"
	"time"

	"github.com/alumieye/eyeapp-backend/internal/config"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInvalidTokenType = errors.New("invalid token type")
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type AccessTokenClaims struct {
	jwt.RegisteredClaims
	Type TokenType `json:"typ"`
}

type TokenService struct {
	secret         []byte
	accessTokenTTL time.Duration
}

func NewTokenService(cfg *config.Config) *TokenService {
	return &TokenService{
		secret:         []byte(cfg.JWTSecret),
		accessTokenTTL: cfg.AccessTokenTTL,
	}
}

func (s *TokenService) GenerateAccessToken(userID string) (string, error) {
	now := time.Now()
	claims := AccessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenTTL)),
		},
		Type: TokenTypeAccess,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *TokenService) ValidateAccessToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", ErrExpiredToken
		}
		return "", ErrInvalidToken
	}

	claims, ok := token.Claims.(*AccessTokenClaims)
	if !ok || !token.Valid {
		return "", ErrInvalidToken
	}

	if claims.Type != TokenTypeAccess {
		return "", ErrInvalidTokenType
	}

	return claims.Subject, nil
}

func (s *TokenService) GetAccessTokenTTL() int {
	return int(s.accessTokenTTL.Seconds())
}
