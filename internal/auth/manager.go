package auth

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type tokenRequester func(ctx context.Context, client *http.Client, clientID, clientSecret string) (TokenResponse, error)

// TokenManager caches an OPS OAuth token and refreshes it before expiry.
type TokenManager struct {
	client        *http.Client
	clientID      string
	clientSecret  string
	refreshWindow time.Duration

	mu        sync.Mutex
	token     string
	expiresAt time.Time

	request tokenRequester
}

func NewTokenManager(client *http.Client, clientID, clientSecret string, refreshWindow time.Duration) *TokenManager {
	if refreshWindow <= 0 {
		refreshWindow = 2 * time.Minute
	}
	return &TokenManager{
		client:        client,
		clientID:      strings.TrimSpace(clientID),
		clientSecret:  strings.TrimSpace(clientSecret),
		refreshWindow: refreshWindow,
		request:       RequestToken,
	}
}

func (m *TokenManager) Token(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	if m.token != "" && now.Add(m.refreshWindow).Before(m.expiresAt) {
		return m.token, nil
	}

	resp, err := m.request(ctx, m.client, m.clientID, m.clientSecret)
	if err != nil {
		return "", err
	}

	expiresInSeconds, parseErr := strconv.Atoi(strings.TrimSpace(resp.ExpiresIn))
	if parseErr != nil || expiresInSeconds <= 0 {
		expiresInSeconds = 20 * 60
	}

	m.token = strings.TrimSpace(resp.AccessToken)
	m.expiresAt = now.Add(time.Duration(expiresInSeconds) * time.Second)
	return m.token, nil
}
