package errors

import "fmt"

const (
	ExitOK           = 0
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
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Body == "" {
		return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error (%d): %s - %s", e.StatusCode, e.Message, e.Body)
}
