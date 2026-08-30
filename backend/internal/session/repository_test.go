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
	connStr := "postgres://postgres:postgres@localhost:5432/splitmate_test?sslmode=disable"
	pool, err := pgxpool.New(context.Background(), connStr)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(), "TRUNCATE TABLE refresh_tokens, users CASCADE")
	require.NoError(t, err)

	return pool
}

func createTestUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	email := "test-" + uuid.New().String() + "@example.com"
	_, err := pool.Exec(context.Background(),
		"INSERT INTO users (id, name, email, password_hash, avatar_url, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, now(), now())",
		userID, "Test User", email, nil, nil)
	require.NoError(t, err)
	return userID
}

func TestCreateRefreshToken(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewRepository(pool)
	userID := createTestUser(t, pool)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	token := "test_refresh_token_123"

	rt, err := repo.CreateRefreshToken(context.Background(), userID, token, expiresAt)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, rt.ID)
	assert.Equal(t, userID, rt.UserID)
	assert.True(t, expiresAt.Truncate(time.Second).Equal(rt.ExpiresAt.Truncate(time.Second)))
	assert.Nil(t, rt.RevokedAt)
}

func TestFindRefreshToken(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewRepository(pool)
	userID := createTestUser(t, pool)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	token := "test_refresh_token_456"

	// Create token
	_, err := repo.CreateRefreshToken(context.Background(), userID, token, expiresAt)
	require.NoError(t, err)

	// Find token
	rt, err := repo.FindRefreshToken(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, userID, rt.UserID)
	assert.True(t, expiresAt.Truncate(time.Second).Equal(rt.ExpiresAt.Truncate(time.Second)))

	// Find non-existent token
	_, err = repo.FindRefreshToken(context.Background(), "non_existent")
	assert.Error(t, err)
	assert.Equal(t, ErrRefreshTokenNotFound, err)
}

func TestRevokeRefreshToken(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	repo := NewRepository(pool)
	userID := createTestUser(t, pool)
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
	userID := createTestUser(t, pool)

	// Test valid token
	validToken := "valid_token_123"
	validExpiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err := repo.CreateRefreshToken(context.Background(), userID, validToken, validExpiresAt)
	require.NoError(t, err)

	rt, err := repo.ValidateRefreshToken(context.Background(), validToken)
	require.NoError(t, err)
	assert.Equal(t, userID, rt.UserID)

	// Revoke valid token so we can create another for the same user
	err = repo.RevokeRefreshToken(context.Background(), validToken)
	require.NoError(t, err)

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
	userID := createTestUser(t, pool)
	otherUserID := createTestUser(t, pool)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	// Create and revoke token1
	_, err := repo.CreateRefreshToken(context.Background(), userID, "token1", expiresAt)
	require.NoError(t, err)
	err = repo.RevokeRefreshToken(context.Background(), "token1")
	require.NoError(t, err)

	// Create and revoke token2
	_, err = repo.CreateRefreshToken(context.Background(), userID, "token2", expiresAt)
	require.NoError(t, err)
	err = repo.RevokeRefreshToken(context.Background(), "token2")
	require.NoError(t, err)

	// Create active token3 for user
	_, err = repo.CreateRefreshToken(context.Background(), userID, "token3", expiresAt)
	require.NoError(t, err)

	// Create token for another user
	_, err = repo.CreateRefreshToken(context.Background(), otherUserID, "token4", expiresAt)
	require.NoError(t, err)

	// Revoke all user tokens (should revoke token3)
	err = repo.RevokeAllUserTokens(context.Background(), userID)
	require.NoError(t, err)

	// Verify user tokens are revoked
	_, err = repo.ValidateRefreshToken(context.Background(), "token3")
	assert.Error(t, err)

	// Verify other user's token is still valid
	rt, err := repo.ValidateRefreshToken(context.Background(), "token4")
	require.NoError(t, err)
	assert.Equal(t, otherUserID, rt.UserID)
}