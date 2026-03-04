package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	epoerrors "github.com/smcronin/epo-cli/internal/errors"
)

const TokenURL = "https://ops.epo.org/3.2/auth/accesstoken"

type TokenResponse struct {
	IssuedAt        string `json:"issued_at"`
	ApplicationName string `json:"application_name"`
	Scope           string `json:"scope"`
	Status          string `json:"status"`
	ExpiresIn       string `json:"expires_in"`
	APIProductList  string `json:"api_product_list"`
	TokenType       string `json:"token_type"`
	AccessToken     string `json:"access_token"`
	Organization    string `json:"organization_name"`
	RefreshCount    string `json:"refresh_count"`
}

func RequestToken(ctx context.Context, client *http.Client, clientID, clientSecret string) (TokenResponse, error) {
	if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" {
		return TokenResponse{}, &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: "missing client ID or client secret",
			Hint:    "Use --client-id/--client-secret or set EPO_CLIENT_ID and EPO_CLIENT_SECRET",
		}
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResponse{}, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+basicAuth(clientID, clientSecret))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("execute token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return TokenResponse{}, &epoerrors.APIError{
			StatusCode: resp.StatusCode,
			Message:    "token request failed",
			Body:       shorten(strings.TrimSpace(string(body)), 500),
		}
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return TokenResponse{}, fmt.Errorf("parse token response: %w", err)
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return TokenResponse{}, &epoerrors.APIError{
			StatusCode: resp.StatusCode,
			Message:    "token response missing access_token",
			Body:       shorten(strings.TrimSpace(string(body)), 500),
		}
	}

	return token, nil
}

func basicAuth(clientID, clientSecret string) string {
	cred := clientID + ":" + clientSecret
	return base64.StdEncoding.EncodeToString([]byte(cred))
}

func shorten(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}
