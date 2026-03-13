package auth

import (
	"errors"
	"time"

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

// AccessTokenClaims represents the claims in an access token
type AccessTokenClaims struct {
	jwt.RegisteredClaims
	Type TokenType `json:"typ"`
}

// TokenService handles JWT token generation and validation
type TokenService struct {
	secret         []byte
	accessTokenTTL time.Duration
}

// NewTokenService creates a new TokenService
func NewTokenService(secret string, accessTokenTTL time.Duration) *TokenService {
	return &TokenService{
		secret:         []byte(secret),
		accessTokenTTL: accessTokenTTL,
	}
}

// GenerateAccessToken creates a new JWT access token for a user
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

// ValidateAccessToken validates a JWT access token and returns the user ID
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

// GetAccessTokenTTL returns the access token TTL in seconds
func (s *TokenService) GetAccessTokenTTL() int {
	return int(s.accessTokenTTL.Seconds())
}
