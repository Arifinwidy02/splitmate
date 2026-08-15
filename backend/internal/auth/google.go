package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	googleTokenEndpoint    = "https://oauth2.googleapis.com/token"
	googleAuthEndpoint     = "https://accounts.google.com/o/oauth2/v2/auth"
	googleUserInfoEndpoint = "https://www.googleapis.com/oauth2/v2/userinfo"
	googleScope            = "openid email profile"
)

type GoogleProfile struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Picture   string `json:"picture"`
	AvatarURL *string
}

type googleClient interface {
	ExchangeCode(ctx context.Context, code, redirectURL, clientID, clientSecret string) (string, error)
	FetchProfile(ctx context.Context, accessToken string) (*GoogleProfile, error)
}

type googleHTTPClient struct {
	http *http.Client
}

func newGoogleHTTPClient() *googleHTTPClient {
	return &googleHTTPClient{http: &http.Client{Timeout: 10 * time.Second}}
}

func (c *googleHTTPClient) ExchangeCode(ctx context.Context, code, redirectURL, clientID, clientSecret string) (string, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("redirect_uri", redirectURL)
	form.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, googleTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("exchange google code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("google token endpoint returned status %d: %s", resp.StatusCode, truncate(string(body), 256))
	}

	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &token); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if token.AccessToken == "" {
		return "", fmt.Errorf("google token response missing access_token")
	}

	return token.AccessToken, nil
}

func (c *googleHTTPClient) FetchProfile(ctx context.Context, accessToken string) (*GoogleProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch google profile: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read userinfo response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo endpoint returned status %d: %s", resp.StatusCode, truncate(string(body), 256))
	}

	var profile GoogleProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("decode userinfo response: %w", err)
	}
	if profile.ID == "" {
		return nil, fmt.Errorf("google userinfo response missing id")
	}
	if profile.Picture != "" {
		profile.AvatarURL = &profile.Picture
	}

	return &profile, nil
}

func googleAuthURL(clientID, redirectURL, state string) string {
	q := url.Values{}
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURL)
	q.Set("response_type", "code")
	q.Set("scope", googleScope)
	q.Set("state", state)

	return googleAuthEndpoint + "?" + q.Encode()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
