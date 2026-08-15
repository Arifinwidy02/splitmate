package session

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	CookieName         = "session"
	AccessTokenTTL     = 15 * time.Minute
	RefreshTokenTTL    = 7 * 24 * time.Hour
	DefaultTokenTTL    = 7 * 24 * time.Hour // Deprecated: use AccessTokenTTL or RefreshTokenTTL
	AccessTokenCookie  = "access_token"
	RefreshTokenCookie = "refresh_token"
)

type TokenService struct {
	secret           []byte
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
}

func NewTokenService(secret []byte, accessTokenTTL, refreshTokenTTL time.Duration) *TokenService {
	return &TokenService{
		secret:          secret,
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func NewTokenServiceWithDefaults(secret []byte) *TokenService {
	return &TokenService{
		secret:          secret,
		accessTokenTTL:  AccessTokenTTL,
		refreshTokenTTL: RefreshTokenTTL,
	}
}

func (t *TokenService) IssueAccessToken(userID uuid.UUID) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(t.accessTokenTTL)

	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		Issuer:    "splitmate",
		Audience:  []string{"splitmate-api"},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(t.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}

	return signed, expiresAt, nil
}

func (t *TokenService) IssueRefreshToken(userID uuid.UUID) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(t.refreshTokenTTL)

	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		Issuer:    "splitmate",
		Audience:  []string{"splitmate-refresh"},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(t.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign refresh token: %w", err)
	}

	return signed, expiresAt, nil
}

// Issue is deprecated: use IssueAccessToken or IssueRefreshToken
func (t *TokenService) Issue(userID uuid.UUID) (string, time.Time, error) {
	return t.IssueAccessToken(userID)
}

func (t *TokenService) Parse(raw string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}

	_, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		return t.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return uuid.Nil, err
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid token subject: %w", err)
	}

	return userID, nil
}

func (t *TokenService) ParseRefreshToken(raw string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}

	_, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		return t.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return uuid.Nil, err
	}

	// Validate that this is a refresh token (check audience)
	if len(claims.Audience) == 0 || claims.Audience[0] != "splitmate-refresh" {
		return uuid.Nil, fmt.Errorf("invalid refresh token audience")
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid token subject: %w", err)
	}

	return userID, nil
}
