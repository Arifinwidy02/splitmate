package session

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	tokens     *TokenService
	repository *Repository
}

func NewService(tokens *TokenService, repository *Repository) *Service {
	return &Service{
		tokens:     tokens,
		repository: repository,
	}
}

type TokenPair struct {
	AccessToken  string
	AccessExpiresAt time.Time
	RefreshToken string
	RefreshExpiresAt time.Time
}

func (s *Service) CreateTokenPair(ctx context.Context, userID uuid.UUID) (*TokenPair, error) {
	// Revoke any existing refresh tokens for this user
	if err := s.repository.RevokeAllUserTokens(ctx, userID); err != nil {
		return nil, fmt.Errorf("revoke existing tokens: %w", err)
	}
	
	// Generate new tokens
	accessToken, accessExpiresAt, err := s.tokens.IssueAccessToken(userID)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}
	
	refreshToken, refreshExpiresAt, err := s.tokens.IssueRefreshToken(userID)
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}
	
	// Store refresh token in database
	if _, err := s.repository.CreateRefreshToken(ctx, userID, refreshToken, refreshExpiresAt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}
	
	return &TokenPair{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}

func (s *Service) RefreshAccessToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Validate refresh token
	rt, err := s.repository.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) || 
		   errors.Is(err, ErrRefreshTokenRevoked) || 
		   errors.Is(err, ErrRefreshTokenExpired) {
			return nil, err
		}
		return nil, fmt.Errorf("validate refresh token: %w", err)
	}
	
	// Revoke the used refresh token (one-time use)
	if err := s.repository.RevokeRefreshToken(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("revoke refresh token: %w", err)
	}
	
	// Generate new token pair
	return s.CreateTokenPair(ctx, rt.UserID)
}

func (s *Service) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	return s.repository.RevokeRefreshToken(ctx, refreshToken)
}

func (s *Service) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	return s.repository.RevokeAllUserTokens(ctx, userID)
}

func (s *Service) CleanupExpiredTokens(ctx context.Context) error {
	return s.repository.CleanupExpiredTokens(ctx)
}

func (s *Service) ParseAccessToken(token string) (uuid.UUID, error) {
	return s.tokens.Parse(token)
}
