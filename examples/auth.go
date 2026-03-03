// Example: OAuth2 token retrieval for OPS API
// Register at https://developers.epo.org to get credentials.

package examples

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	TokenURL = "https://ops.epo.org/3.2/auth/accesstoken"
	OPSURL   = "https://ops.epo.org/rest-services"
)

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Status      string `json:"status"`
}

// OPSClient is a token-managing HTTP client for OPS.
type OPSClient struct {
	clientID     string
	clientSecret string
	token        string
	tokenExpiry  time.Time
	mu           sync.Mutex
	httpClient   *http.Client
}

func NewOPSClient(clientID, clientSecret string) *OPSClient {
	return &OPSClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// GetToken returns a valid token, refreshing if needed.
func (c *OPSClient) GetToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Token valid for > 60s remaining — reuse it
	if c.token != "" && time.Until(c.tokenExpiry) > 60*time.Second {
		return c.token, nil
	}

	return c.refreshToken()
}

func (c *OPSClient) refreshToken() (string, error) {
	creds := base64.StdEncoding.EncodeToString(
		[]byte(fmt.Sprintf("%s:%s", c.clientID, c.clientSecret)),
	)

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+creds)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		return "", fmt.Errorf("token error %d: %s - %s",
			resp.StatusCode, errResp["message"], errResp["description"])
	}

	var tok TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}

	// Parse expiry (seconds)
	var expirySeconds int
	fmt.Sscanf(tok.ExpiresIn, "%d", &expirySeconds)

	c.token = tok.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(expirySeconds) * time.Second)

	return c.token, nil
}

// Get makes an authenticated GET request to OPS.
func (c *OPSClient) Get(path string, accept string) (*http.Response, error) {
	token, err := c.GetToken()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", OPSURL+"/"+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// If token expired mid-session, refresh and retry once
	if resp.StatusCode == http.StatusForbidden {
		resp.Body.Close()
		c.mu.Lock()
		c.token = "" // force refresh
		c.mu.Unlock()

		token, err = c.GetToken()
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return c.httpClient.Do(req)
	}

	return resp, nil
}
