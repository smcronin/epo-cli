package errors

import (
	"fmt"
	"time"
)

const (
	ExitGeneralError = 1
	ExitUsageError   = 2
	ExitAuthFailure  = 3
	ExitNotFound     = 4
	ExitRateLimited  = 5
	ExitServerError  = 6
)

type CLIError struct {
	Code    int    `json:"code"`
	Type    string `json:"type"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
	// Reason is the raw X-Rejection-Reason header value when OPS rejected the
	// request for quota / fair-use reasons. It is an opaque OPS string.
	Reason string `json:"reason,omitempty"`
}

func (e *CLIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Hint == "" {
		return e.Message
	}
	return fmt.Sprintf("%s (%s)", e.Message, e.Hint)
}

type APIError struct {
	StatusCode int
	Message    string
	Body       string
	// RetryAfter is the wait OPS requested via the Retry-After header, if any.
	RetryAfter time.Duration
	// RejectionReason is the raw X-Rejection-Reason header value, if any.
	RejectionReason string
	// Service names the upstream that answered: "" or ServiceOPS for OPS,
	// ServiceEPS for the European Publication Server (which takes no
	// credentials).
	Service string
}

const (
	ServiceOPS = "OPS"
	ServiceEPS = "EPS"
)

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Body == "" {
		return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error (%d): %s - %s", e.StatusCode, e.Message, e.Body)
}
