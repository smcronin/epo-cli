package auth

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestTokenManagerCachesTokenUntilRefreshWindow(t *testing.T) {
	t.Parallel()

	m := NewTokenManager(&http.Client{}, "id", "secret", 2*time.Minute)
	callCount := 0
	m.request = func(ctx context.Context, client *http.Client, clientID, clientSecret string) (TokenResponse, error) {
		callCount++
		return TokenResponse{
			AccessToken: "token-1",
			ExpiresIn:   "1200",
		}, nil
	}

	got1, err := m.Token(context.Background())
	if err != nil {
		t.Fatalf("first token call error: %v", err)
	}
	got2, err := m.Token(context.Background())
	if err != nil {
		t.Fatalf("second token call error: %v", err)
	}
	if got1 != "token-1" || got2 != "token-1" {
		t.Fatalf("unexpected tokens: %q, %q", got1, got2)
	}
	if callCount != 1 {
		t.Fatalf("expected requester to be called once, got %d", callCount)
	}
}

func TestTokenManagerRefreshesInsideWindow(t *testing.T) {
	t.Parallel()

	m := NewTokenManager(&http.Client{}, "id", "secret", 2*time.Minute)
	callCount := 0
	m.request = func(ctx context.Context, client *http.Client, clientID, clientSecret string) (TokenResponse, error) {
		callCount++
		if callCount == 1 {
			return TokenResponse{
				AccessToken: "token-old",
				ExpiresIn:   "30",
			}, nil
		}
		return TokenResponse{
			AccessToken: "token-new",
			ExpiresIn:   "1200",
		}, nil
	}

	got1, err := m.Token(context.Background())
	if err != nil {
		t.Fatalf("first token call error: %v", err)
	}
	got2, err := m.Token(context.Background())
	if err != nil {
		t.Fatalf("second token call error: %v", err)
	}
	if got1 != "token-old" || got2 != "token-new" {
		t.Fatalf("expected token refresh, got %q then %q", got1, got2)
	}
	if callCount != 2 {
		t.Fatalf("expected requester to be called twice, got %d", callCount)
	}
}
