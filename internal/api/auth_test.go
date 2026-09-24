package api

import (
	"errors"
	"net/http"
	"testing"
)

func TestWhoamiReadsUserFromEnvelope(t *testing.T) {
	var path string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		respondWith(`{"status":200,"data":{"account":{"subdomain":"inbox"},"user":{"id":"U1","name":"Ada","email":"ada@x.com"}}}`)(w, r)
	})
	who, err := c.Whoami()
	if err != nil {
		t.Fatal(err)
	}
	if path != "/v1/auth_tokens" || who.ID != "U1" || who.Name != "Ada" || who.Email != "ada@x.com" {
		t.Fatalf("path=%s who=%+v", path, who)
	}
}

func TestWhoamiBadCredentials(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"status":401,"errors":["Unauthorized"]}`))
	})
	_, err := c.Whoami()
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("err = %T %v, want *AuthError", err, err)
	}
}
