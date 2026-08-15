package session

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenService_IssueAccessToken(t *testing.T) {
	secret := []byte("test-secret-key-12345678901234567890")
	tokenService := NewTokenService(secret, 15*time.Minute, 7*24*time.Hour)
	
	userID := uuid.New()
	token, expiresAt, err := tokenService.IssueAccessToken(userID)
	
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, expiresAt.After(time.Now()))
	assert.True(t, expiresAt.Before(time.Now().Add(16*time.Minute)))
}

func TestTokenService_IssueRefreshToken(t *testing.T) {
	secret := []byte("test-secret-key-12345678901234567890")
	tokenService := NewTokenService(secret, 15*time.Minute, 7*24*time.Hour)
	
	userID := uuid.New()
	token, expiresAt, err := tokenService.IssueRefreshToken(userID)
	
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.True(t, expiresAt.After(time.Now()))
	assert.True(t, expiresAt.Before(time.Now().Add(8*24*time.Hour)))
}

func TestTokenService_ParseAccessToken(t *testing.T) {
	secret := []byte("test-secret-key-12345678901234567890")
	tokenService := NewTokenService(secret, 15*time.Minute, 7*24*time.Hour)
	
	userID := uuid.New()
	token, _, err := tokenService.IssueAccessToken(userID)
	require.NoError(t, err)
	
	// Parse valid token
	parsedUserID, err := tokenService.Parse(token)
	require.NoError(t, err)
	assert.Equal(t, userID, parsedUserID)
	
	// Parse invalid token
	_, err = tokenService.Parse("invalid.token.here")
	assert.Error(t, err)
}

func TestTokenService_ParseRefreshToken(t *testing.T) {
	secret := []byte("test-secret-key-12345678901234567890")
	tokenService := NewTokenService(secret, 15*time.Minute, 7*24*time.Hour)
	
	userID := uuid.New()
	token, _, err := tokenService.IssueRefreshToken(userID)
	require.NoError(t, err)
	
	// Parse valid refresh token
	parsedUserID, err := tokenService.ParseRefreshToken(token)
	require.NoError(t, err)
	assert.Equal(t, userID, parsedUserID)
	
	// Try to parse access token as refresh token (should fail)
	accessToken, _, err := tokenService.IssueAccessToken(userID)
	require.NoError(t, err)
	
	_, err = tokenService.ParseRefreshToken(accessToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid refresh token audience")
}

func TestService_CreateTokenPair(t *testing.T) {
	// This would require a database connection
	// For now, we'll test the logic without a real DB
	t.Skip("Requires database connection")
}

func TestService_RefreshAccessToken(t *testing.T) {
	// This would require a database connection
	// For now, we'll test the logic without a real DB
	t.Skip("Requires database connection")
}
