package api

import "fmt"

// APIError is the base error type for all Avochato API errors.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

// AuthError is returned on 401 responses.
type AuthError struct{ APIError }

// ForbiddenError is returned on 403 responses.
type ForbiddenError struct{ APIError }

// NotFoundError is returned on 404 responses.
type NotFoundError struct{ APIError }

// RateLimitError is returned on 429 responses.
type RateLimitError struct {
	APIError
	RetryAfter int
}

// ServerError is returned on 5xx responses.
type ServerError struct{ APIError }

func newTypedError(statusCode int, message string, retryAfter int) error {
	base := APIError{StatusCode: statusCode, Message: message}
	switch statusCode {
	case 401:
		return &AuthError{base}
	case 403:
		return &ForbiddenError{base}
	case 404:
		return &NotFoundError{base}
	case 429:
		return &RateLimitError{APIError: base, RetryAfter: retryAfter}
	default:
		if statusCode >= 500 {
			return &ServerError{base}
		}
		return &base
	}
}
