package api

import (
	"errors"
	"testing"
)

func TestNewTypedError(t *testing.T) {
	cases := []struct {
		status int
		check  func(error) bool
	}{
		{401, func(e error) bool { var x *AuthError; return errors.As(e, &x) }},
		{403, func(e error) bool { var x *ForbiddenError; return errors.As(e, &x) }},
		{404, func(e error) bool { var x *NotFoundError; return errors.As(e, &x) }},
		{429, func(e error) bool { var x *RateLimitError; return errors.As(e, &x) && x.RetryAfter == 7 }},
		{500, func(e error) bool { var x *ServerError; return errors.As(e, &x) }},
		{503, func(e error) bool { var x *ServerError; return errors.As(e, &x) }},
		{400, func(e error) bool { var x *APIError; return errors.As(e, &x) }},
	}
	for _, c := range cases {
		err := newTypedError(c.status, "boom", 7)
		if !c.check(err) {
			t.Errorf("status %d: got %T", c.status, err)
		}
	}
}

func TestAPIErrorMessage(t *testing.T) {
	if got := newTypedError(404, "ticket not found", 0).Error(); got != "API error 404: ticket not found" {
		t.Fatalf("Error() = %q", got)
	}
}
