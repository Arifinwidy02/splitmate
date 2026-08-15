package session

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	// This assumes a test database is available
	// In a real setup, you'd use docker-testcontainers or similar
	connStr := "postgres://postgres:postgres@localhost:5432/splitmate_test?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), connStr)
	require.NoError(t, err)
	
	// Clean up refresh_tokens table
	_, err = pool.Exec(context.Background(), "TRUNCATE TABLE refresh_tokens CASCADE")
	require.NoError(t, err)
	
	return pool
}

func TestCreateRefreshToken(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewRepository(pool)
	userID := uuid.New()
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	token := "test_refresh_token_123"

	rt, err := repo.CreateRefreshToken(context.Background(), userID, token, expiresAt)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, rt.ID)
	assert.Equal(t, userID, rt.UserID)
	assert.Equal(t, expiresAt, rt.ExpiresAt)
	assert.Nil(t, rt.RevokedAt)
}

func TestFindRefreshToken(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewRepository(pool)
	userID := uuid.New()
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	token := "test_refresh_token_456"

	// Create token
	_, err := repo.CreateRefreshToken(context.Background(), userID, token, expiresAt)
	require.NoError(t, err)

	// Find token
	rt, err := repo.FindRefreshToken(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, userID, rt.UserID)
	assert.Equal(t, expiresAt, rt.ExpiresAt)

	// Find non-existent token
	_, err = repo.FindRefreshToken(context.Background(), "non_existent")
	assert.Error(t, err)
	assert.Equal(t, ErrRefreshTokenNotFound, err)
}

func TestRevokeRefreshToken(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewRepository(pool)
	userID := uuid.New()
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	token := "test_refresh_token_789"

	// Create token
	_, err := repo.CreateRefreshToken(context.Background(), userID, token, expiresAt)
	require.NoError(t, err)

	// Revoke token
	err = repo.RevokeRefreshToken(context.Background(), token)
	require.NoError(t, err)

	// Verify revoked
	rt, err := repo.FindRefreshToken(context.Background(), token)
	require.NoError(t, err)
	assert.NotNil(t, rt.RevokedAt)

	// Try to revoke again (should fail)
	err = repo.RevokeRefreshToken(context.Background(), token)
	assert.Error(t, err)
	assert.Equal(t, ErrRefreshTokenNotFound, err)
}

func TestValidateRefreshToken(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewRepository(pool)
	userID := uuid.New()
	
	// Test valid token
	validToken := "valid_token_123"
	validExpiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err := repo.CreateRefreshToken(context.Background(), userID, validToken, validExpiresAt)
	require.NoError(t, err)

	rt, err := repo.ValidateRefreshToken(context.Background(), validToken)
	require.NoError(t, err)
	assert.Equal(t, userID, rt.UserID)

	// Test revoked token
	revokedToken := "revoked_token_456"
	revokedExpiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err = repo.CreateRefreshToken(context.Background(), userID, revokedToken, revokedExpiresAt)
	require.NoError(t, err)
	err = repo.RevokeRefreshToken(context.Background(), revokedToken)
	require.NoError(t, err)

	_, err = repo.ValidateRefreshToken(context.Background(), revokedToken)
	assert.Error(t, err)
	assert.Equal(t, ErrRefreshTokenRevoked, err)

	// Test expired token
	expiredToken := "expired_token_789"
	expiredExpiresAt := time.Now().Add(-1 * time.Hour)
	_, err = repo.CreateRefreshToken(context.Background(), userID, expiredToken, expiredExpiresAt)
	require.NoError(t, err)

	_, err = repo.ValidateRefreshToken(context.Background(), expiredToken)
	assert.Error(t, err)
	assert.Equal(t, ErrRefreshTokenExpired, err)
}

func TestRevokeAllUserTokens(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewRepository(pool)
	userID := uuid.New()
	otherUserID := uuid.New()
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	// Create multiple tokens for the user
	_, err := repo.CreateRefreshToken(context.Background(), userID, "token1", expiresAt)
	require.NoError(t, err)
	_, err = repo.CreateRefreshToken(context.Background(), userID, "token2", expiresAt)
	require.NoError(t, err)
	
	// Create token for another user
	_, err = repo.CreateRefreshToken(context.Background(), otherUserID, "token3", expiresAt)
	require.NoError(t, err)

	// Revoke all user tokens
	err = repo.RevokeAllUserTokens(context.Background(), userID)
	require.NoError(t, err)

	// Verify user tokens are revoked
	_, err = repo.ValidateRefreshToken(context.Background(), "token1")
	assert.Error(t, err)
	_, err = repo.ValidateRefreshToken(context.Background(), "token2")
	assert.Error(t, err)

	// Verify other user's token is still valid
	rt, err := repo.ValidateRefreshToken(context.Background(), "token3")
	require.NoError(t, err)
	assert.Equal(t, otherUserID, rt.UserID)
}
