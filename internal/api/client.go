package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/smcronin/epo-cli/internal/throttle"
)

const DefaultBaseURL = "https://ops.epo.org/rest-services"

type TokenProvider interface {
	Token(ctx context.Context) (string, error)
}

type Request struct {
	Method      string
	Path        string
	Query       url.Values
	Headers     map[string]string
	Body        []byte
	ContentType string
	Accept      string
}

type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Metadata   throttle.Metadata
}

type Client struct {
	httpClient    *http.Client
	baseURL       string
	tokenProvider TokenProvider
	maxRetries    int
	backoffBase   time.Duration
	backoffCap    time.Duration
	jitterMax     time.Duration
}

func NewClient(httpClient *http.Client, tokenProvider TokenProvider) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		httpClient:    httpClient,
		baseURL:       DefaultBaseURL,
		tokenProvider: tokenProvider,
		maxRetries:    3,
		backoffBase:   500 * time.Millisecond,
		backoffCap:    8 * time.Second,
		jitterMax:     250 * time.Millisecond,
	}
}

func (c *Client) SetBaseURL(baseURL string) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return
	}
	c.baseURL = strings.TrimRight(baseURL, "/")
}

func (c *Client) Do(ctx context.Context, req Request) (Response, error) {
	if c.tokenProvider == nil {
		return Response{}, fmt.Errorf("token provider is not configured")
	}

	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}

	fullURL, err := c.buildURL(req.Path, req.Query)
	if err != nil {
		return Response{}, err
	}

	token, err := c.tokenProvider.Token(ctx)
	if err != nil {
		return Response{}, err
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, method, fullURL, bytes.NewReader(req.Body))
		if err != nil {
			return Response{}, fmt.Errorf("build request: %w", err)
		}
		httpReq.Header.Set("Authorization", "Bearer "+token)
		httpReq.Header.Set("Accept", defaultIfEmpty(req.Accept, "application/json"))
		if len(req.Body) > 0 {
			httpReq.Header.Set("Content-Type", defaultIfEmpty(req.ContentType, "application/json"))
		}
		for k, v := range req.Headers {
			if strings.TrimSpace(k) == "" {
				continue
			}
			httpReq.Header.Set(k, v)
		}

		httpResp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = err
			if attempt < c.maxRetries {
				if sleepErr := c.sleep(ctx, c.retryDelay(attempt, 0)); sleepErr != nil {
					return Response{}, sleepErr
				}
				continue
			}
			return Response{}, fmt.Errorf("execute request: %w", err)
		}

		body, readErr := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if readErr != nil {
			return Response{}, fmt.Errorf("read response body: %w", readErr)
		}

		metadata := throttle.ParseHeaders(httpResp.Header)
		resp := Response{
			StatusCode: httpResp.StatusCode,
			Headers:    httpResp.Header.Clone(),
			Body:       body,
			Metadata:   metadata,
		}

		if httpResp.StatusCode >= 200 && httpResp.StatusCode < 300 {
			return resp, nil
		}

		lastErr = &epoerrors.APIError{
			StatusCode: httpResp.StatusCode,
			Message:    "OPS request failed",
			Body:       shortBody(body, 1000),
		}

		if !shouldRetry(httpResp.StatusCode) || attempt >= c.maxRetries {
			return resp, lastErr
		}

		delay := c.retryDelay(attempt, metadata.RetryAfter)
		if sleepErr := c.sleep(ctx, delay); sleepErr != nil {
			return Response{}, sleepErr
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("request failed")
	}
	return Response{}, lastErr
}

func (c *Client) buildURL(requestPath string, query url.Values) (string, error) {
	base, err := url.Parse(strings.TrimRight(c.baseURL, "/"))
	if err != nil {
		return "", fmt.Errorf("parse base URL: %w", err)
	}

	cleanPath := path.Clean("/" + strings.TrimSpace(requestPath))
	base.Path = strings.TrimRight(base.Path, "/") + cleanPath
	if query != nil {
		base.RawQuery = query.Encode()
	}
	return base.String(), nil
}

func (c *Client) retryDelay(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}
	delay := c.backoffBase * time.Duration(1<<attempt)
	if delay > c.backoffCap {
		delay = c.backoffCap
	}
	if c.jitterMax > 0 {
		delay += time.Duration(rand.Int63n(int64(c.jitterMax)))
	}
	return delay
}

func (c *Client) sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func shouldRetry(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= http.StatusInternalServerError
}

func defaultIfEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func shortBody(body []byte, max int) string {
	s := strings.TrimSpace(string(body))
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}
