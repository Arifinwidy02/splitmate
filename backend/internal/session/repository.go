package session

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
	ErrRefreshTokenRevoked   = errors.New("refresh token revoked")
	ErrRefreshTokenExpired   = errors.New("refresh token expired")
)

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (r *Repository) CreateRefreshToken(ctx context.Context, userID uuid.UUID, token string, expiresAt time.Time) (*RefreshToken, error) {
	tokenHash := hashToken(token)
	
	row := r.pool.QueryRow(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)
		 RETURNING id, user_id::text, token_hash, expires_at, created_at, revoked_at`,
		userID.String(), tokenHash, expiresAt)

	var rt RefreshToken
	var rawUserID string
	
	if err := row.Scan(&rt.ID, &rawUserID, &rt.TokenHash, &rt.ExpiresAt, &rt.CreatedAt, &rt.RevokedAt); err != nil {
		return nil, fmt.Errorf("create refresh token: %w", err)
	}
	
	userID, err := uuid.Parse(rawUserID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}
	rt.UserID = userID
	
	return &rt, nil
}

func (r *Repository) FindRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	tokenHash := hashToken(token)
	
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id::text, token_hash, expires_at, created_at, revoked_at
		 FROM refresh_tokens
		 WHERE token_hash = $1`,
		tokenHash)

	var rt RefreshToken
	var rawUserID string
	
	if err := row.Scan(&rt.ID, &rawUserID, &rt.TokenHash, &rt.ExpiresAt, &rt.CreatedAt, &rt.RevokedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("find refresh token: %w", err)
	}
	
	userID, err := uuid.Parse(rawUserID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}
	rt.UserID = userID
	
	return &rt, nil
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, token string) error {
	tokenHash := hashToken(token)
	
	now := time.Now()
	result, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens
		 SET revoked_at = $1
		 WHERE token_hash = $2 AND revoked_at IS NULL`,
		now, tokenHash)
	
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	
	if result.RowsAffected() == 0 {
		return ErrRefreshTokenNotFound
	}
	
	return nil
}

func (r *Repository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens
		 SET revoked_at = $1
		 WHERE user_id = $2 AND revoked_at IS NULL`,
		now, userID.String())
	
	if err != nil {
		return fmt.Errorf("revoke all user tokens: %w", err)
	}
	
	return nil
}

func (r *Repository) CleanupExpiredTokens(ctx context.Context) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM refresh_tokens
		 WHERE expires_at < $1 OR (revoked_at IS NOT NULL AND revoked_at < $2)`,
		time.Now(), time.Now().Add(-24*time.Hour))
	
	if err != nil {
		return fmt.Errorf("cleanup expired tokens: %w", err)
	}
	
	return nil
}

func (r *Repository) ValidateRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	rt, err := r.FindRefreshToken(ctx, token)
	if err != nil {
		return nil, err
	}
	
	if rt.RevokedAt != nil {
		return nil, ErrRefreshTokenRevoked
	}
	
	if time.Now().After(rt.ExpiresAt) {
		return nil, ErrRefreshTokenExpired
	}
	
	return rt, nil
}
