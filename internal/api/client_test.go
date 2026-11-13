package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	epoerrors "github.com/smcronin/epo-cli/internal/errors"
)

type staticTokenProvider struct {
	token string
}

func (p staticTokenProvider) Token(ctx context.Context) (string, error) {
	return p.token, nil
}

func TestClientRetriesAndSucceeds(t *testing.T) {
	t.Parallel()

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") != "Bearer abc123" {
			t.Fatalf("missing auth header, got %q", r.Header.Get("Authorization"))
		}

		if requests == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"temporary"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Throttling-Control", "busy (retrieval=yellow:100)")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := NewClient(srv.Client(), staticTokenProvider{token: "abc123"})
	client.SetBaseURL(srv.URL)
	client.maxRetries = 2
	client.backoffBase = 1 * time.Millisecond
	client.backoffCap = 2 * time.Millisecond
	client.jitterMax = 0

	resp, err := client.Do(context.Background(), Request{
		Method: http.MethodGet,
		Path:   "/published-data/publication/epodoc/EP1000000/biblio",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if requests != 2 {
		t.Fatalf("expected 2 requests, got %d", requests)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}
	if resp.Metadata.Throttle.System != "busy" {
		t.Fatalf("unexpected throttle system: %q", resp.Metadata.Throttle.System)
	}
}

func TestClientReturnsAPIErrorOnFinalFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer srv.Close()

	client := NewClient(srv.Client(), staticTokenProvider{token: "abc123"})
	client.SetBaseURL(srv.URL)
	client.maxRetries = 1
	client.backoffBase = 1 * time.Millisecond
	client.backoffCap = 1 * time.Millisecond
	client.jitterMax = 0

	_, err := client.Do(context.Background(), Request{
		Method: http.MethodGet,
		Path:   "/published-data/publication/epodoc/EP1000000/biblio",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClientReturnsRateLimitWhenRetryAfterExceedsDeadline(t *testing.T) {
	t.Parallel()

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Retry-After", "60000")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`<error><code>429</code></error>`))
	}))
	defer srv.Close()

	client := NewClient(srv.Client(), staticTokenProvider{token: "abc123"})
	client.SetBaseURL(srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	started := time.Now()
	_, err := client.Do(ctx, Request{
		Method: http.MethodGet,
		Path:   "/published-data/search",
	})
	elapsed := time.Since(started)

	var apiErr *epoerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected the 429 API error, got %v", err)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("unexpected status code: %d", apiErr.StatusCode)
	}
	if apiErr.RetryAfter != 60*time.Second {
		t.Fatalf("expected Retry-After of 60s on the error, got %s", apiErr.RetryAfter)
	}
	if requests != 1 {
		t.Fatalf("expected no retry when the wait cannot fit the deadline, got %d requests", requests)
	}
	if elapsed > 400*time.Millisecond {
		t.Fatalf("expected an immediate return, took %s", elapsed)
	}
}

func TestClientDoesNotRetryQuotaRejection(t *testing.T) {
	t.Parallel()

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("X-Rejection-Reason", "RegisteredQuotaPerWeek")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<error><code>403</code><message>This request has been rejected due to the violation of Fair Use policy</message></error>`))
	}))
	defer srv.Close()

	client := NewClient(srv.Client(), staticTokenProvider{token: "abc123"})
	client.SetBaseURL(srv.URL)
	client.backoffBase = 1 * time.Millisecond
	client.backoffCap = 1 * time.Millisecond
	client.jitterMax = 0

	_, err := client.Do(context.Background(), Request{
		Method: http.MethodGet,
		Path:   "/published-data/search",
	})

	var apiErr *epoerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected the 403 API error, got %v", err)
	}
	if apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("unexpected status code: %d", apiErr.StatusCode)
	}
	if apiErr.RejectionReason != "RegisteredQuotaPerWeek" {
		t.Fatalf("expected the X-Rejection-Reason value on the error, got %q", apiErr.RejectionReason)
	}
	if requests != 1 {
		t.Fatalf("expected a quota 403 without Retry-After not to be retried, got %d requests", requests)
	}
}

func TestClientRetriesForbiddenWithRetryAfter(t *testing.T) {
	t.Parallel()

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.Header().Set("Retry-After", "5")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`<error><code>403</code></error>`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := NewClient(srv.Client(), staticTokenProvider{token: "abc123"})
	client.SetBaseURL(srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Do(ctx, Request{
		Method: http.MethodGet,
		Path:   "/published-data/search",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}
	if requests != 2 {
		t.Fatalf("expected one retry after the Retry-After wait, got %d requests", requests)
	}
}

func TestClientDoesNotRetryForbiddenWhenRetryAfterExceedsDeadline(t *testing.T) {
	t.Parallel()

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Retry-After", "60000")
		w.Header().Set("X-Rejection-Reason", "IndividualQuotaPerHour")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<error><code>403</code></error>`))
	}))
	defer srv.Close()

	client := NewClient(srv.Client(), staticTokenProvider{token: "abc123"})
	client.SetBaseURL(srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err := client.Do(ctx, Request{
		Method: http.MethodGet,
		Path:   "/published-data/search",
	})

	var apiErr *epoerrors.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected the 403 API error, got %v", err)
	}
	if apiErr.RetryAfter != 60*time.Second {
		t.Fatalf("expected Retry-After of 60s on the error, got %s", apiErr.RetryAfter)
	}
	if requests != 1 {
		t.Fatalf("expected no retry when the wait cannot fit the deadline, got %d requests", requests)
	}
}
