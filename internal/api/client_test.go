package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/avochato/avochato-cli/internal/config"
)

func testClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewClient(&config.Config{AuthID: "id", AuthSecret: "s3cret", Account: "inbox", BaseURL: srv.URL})
}

func okJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"status":200,"data":{"tickets":[{"id":"t1","status":"Closed"}]}}`)
}

func respondWith(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}
}

func TestIDsCannotChangeEndpointOrInjectParams(t *testing.T) {
	var gotPath, gotSubdomain string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_ = r.ParseForm()
		gotSubdomain = r.Form.Get("subdomain")
		okJSON(w)
	})

	if _, err := c.UpdateTicket("abc/../../users/u1?subdomain=other", UpdateTicketParams{Status: "closed"}); err != nil {
		t.Fatal(err)
	}
	if want := "/v1/tickets/abc%2F..%2F..%2Fusers%2Fu1%3Fsubdomain=other"; gotPath != want {
		t.Fatalf("path = %q, want %q", gotPath, want)
	}
	if gotSubdomain != "inbox" {
		t.Fatalf("subdomain = %q, want inbox", gotSubdomain)
	}
}

func TestDotSegmentIDsAreRejected(t *testing.T) {
	called := false
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		okJSON(w)
	})
	for _, id := range []string{"..", ".", ""} {
		if _, err := c.UpdateTicket(id, UpdateTicketParams{Status: "closed"}); err == nil {
			t.Errorf("id %q: expected error", id)
		}
	}
	if called {
		t.Fatal("request reached the server")
	}
}

func TestRetryOn429ResendsFormOnce(t *testing.T) {
	var bodies []string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"status":429,"errors":["slow down"]}`)
	})

	_, err := c.Post("messages", url.Values{"phone": {"+15555550100"}})
	if err == nil {
		t.Fatal("expected error after retry")
	}
	if len(bodies) != 2 {
		t.Fatalf("got %d requests, want 2 (one retry)", len(bodies))
	}
	for i, b := range bodies {
		if !strings.Contains(b, "auth_secret=s3cret") || !strings.Contains(b, "phone=") {
			t.Fatalf("request %d body missing form fields: %q", i, b)
		}
	}
}

func TestRedact(t *testing.T) {
	got := redact("https://x/v1/users?auth_id=a&auth_secret=s3cret&subdomain=b")
	if strings.Contains(got, "s3cret") || !strings.Contains(got, "auth_id=REDACTED&auth_secret=REDACTED&subdomain=b") {
		t.Fatalf("redact = %q", got)
	}
}

func TestRedirectsAreNotFollowed(t *testing.T) {
	leaked := false
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked = true }))
	t.Cleanup(other.Close)
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+"/v1/messages", http.StatusTemporaryRedirect)
	})
	if _, err := c.Post("messages", nil); err == nil {
		t.Fatal("expected error on redirect")
	}
	if leaked {
		t.Fatal("credentials were sent to the redirect target")
	}
}

func TestNetworkErrorHidesSecret(t *testing.T) {
	c := NewClient(&config.Config{AuthID: "id", AuthSecret: "s3cret", BaseURL: "http://127.0.0.1:1"})
	_, err := c.Get("users", nil)
	if err == nil || strings.Contains(err.Error(), "s3cret") {
		t.Fatalf("error leaks secret or is nil: %v", err)
	}
}

func TestStringErrorsAreReported(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"status":400,"errors":"+16505551234 is already opted out"}`)
	})
	err := c.OptOutContact("C1")
	if err == nil || !strings.Contains(err.Error(), "already opted out") {
		t.Fatalf("err = %v", err)
	}
}
