package cmd

import (
	"bufio"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/avochato/avochato-cli/internal/config"
)

// loginAPI answers the two calls login makes.
func loginAPI(t *testing.T) *fakeAPI {
	f := newFakeAPI(t)
	f.ok("GET /v1/auth_tokens", map[string]any{"user": map[string]any{"id": "U1", "name": "Ada", "email": "ada@x.com"}})
	f.ok("GET /v1/users", map[string]any{"users": []any{}})
	return f
}

func loadCreds(t *testing.T) *config.Credentials {
	t.Helper()
	c, err := config.LoadCredentials()
	if err != nil || c == nil {
		t.Fatalf("credentials: %v, %v", c, err)
	}
	return c
}

func TestLoginSavesProfile(t *testing.T) {
	f := loginAPI(t)
	home(t, nil)

	res := run(t, "id1\nsec1\nacme\n", "login", "--base-url", f.URL)
	if res.Failed || !strings.Contains(res.Stdout, "Logged in as Ada (acme)") {
		t.Fatalf("login: %+v", res)
	}
	c := loadCreds(t)
	p := c.Accounts["acme"]
	if c.DefaultAccount != "acme" || p.AuthID != "id1" || p.AuthSecret != "sec1" || p.Subdomain != "acme" || p.BaseURL != f.URL {
		t.Fatalf("saved = %+v", c)
	}
	if got := f.last(t).Form.Get("subdomain"); got != "acme" {
		t.Fatalf("inbox check used subdomain %q", got)
	}
}

func TestLoginSecondInboxWarnsAboutDefault(t *testing.T) {
	f := loginAPI(t)
	oneInbox(t, f)

	res := run(t, "id2\nsec2\nsupport\n", "auth", "login", "--base-url", f.URL)
	if res.Failed || !strings.Contains(res.Stdout, `default profile is still "acme"`) {
		t.Fatalf("login: %+v", res)
	}
	if loadCreds(t).DefaultAccount != "acme" {
		t.Fatal("default must not change without --default")
	}

	res = run(t, "id2\nsec2\nsupport\n", "login", "--base-url", f.URL, "--default", "--profile", "help")
	if res.Failed || loadCreds(t).DefaultAccount != "help" || loadCreds(t).Accounts["help"].Subdomain != "support" {
		t.Fatalf("login --default --profile: %+v", res)
	}
}

func TestLoginRejectsBadCredentialsAndInbox(t *testing.T) {
	f := newFakeAPI(t)
	f.on("GET /v1/auth_tokens", func(*http.Request) (int, any) { return 401, []string{"Unauthorized"} })
	home(t, nil)
	if res := run(t, "bad\nbad\nacme\n", "login", "--base-url", f.URL); !res.Failed {
		t.Fatal("bad credentials should fail")
	}

	f = newFakeAPI(t)
	f.ok("GET /v1/auth_tokens", map[string]any{"user": map[string]any{"name": "Ada"}})
	f.on("GET /v1/users", func(*http.Request) (int, any) { return 401, []string{"Unauthorized"} })
	res := run(t, "id1\nsec1\ntypo-inbox\n", "login", "--base-url", f.URL)
	if !res.Failed || !strings.Contains(res.Stderr, `cannot access inbox "typo-inbox"`) {
		t.Fatalf("bad inbox: %+v", res)
	}
	if c, _ := config.LoadCredentials(); c != nil {
		t.Fatal("nothing may be saved after a failed login")
	}
}

func TestLoginRefusesHTTPAndCorruptFile(t *testing.T) {
	home(t, nil)
	if res := run(t, "a\nb\nc\n", "login", "--base-url", "http://api.example.com"); !res.Failed || !strings.Contains(res.Stderr, "https") {
		t.Fatalf("http base url: %+v", res)
	}

	f := loginAPI(t)
	path, _ := config.CredentialsPath()
	_ = os.MkdirAll(strings.TrimSuffix(path, "/credentials.json"), 0o700)
	_ = os.WriteFile(path, []byte("{broken"), 0o600)
	res := run(t, "id1\nsec1\nacme\n", "login", "--base-url", f.URL)
	if !res.Failed || !strings.Contains(res.Stderr, "could not read") {
		t.Fatalf("corrupt file: %+v", res)
	}
	if b, _ := os.ReadFile(path); string(b) != "{broken" {
		t.Fatal("login overwrote an unreadable credentials file")
	}
}

func TestLoginShortcutHasSameFlags(t *testing.T) {
	for _, name := range []string{"base-url", "profile", "default"} {
		if loginCmd.Flags().Lookup(name) == nil {
			t.Errorf("login is missing --%s", name)
		}
	}
}

func TestWhoami(t *testing.T) {
	f := loginAPI(t)
	oneInbox(t, f)
	var who map[string]any
	res := run(t, "", "auth", "whoami", "--json")
	decode(t, res, &who)
	if res.Failed || who["email"] != "ada@x.com" {
		t.Fatalf("whoami: %+v", res)
	}
}

func TestAuthListUseLogout(t *testing.T) {
	f := newFakeAPI(t)
	twoInboxes(t, f)

	res := run(t, "", "auth", "list")
	if !strings.Contains(res.Stdout, "* acme") || !strings.Contains(res.Stdout, "  support") {
		t.Fatalf("list: %q", res.Stdout)
	}
	if res := run(t, "", "auth", "use", "support"); res.Failed || loadCreds(t).DefaultAccount != "support" {
		t.Fatalf("use: %+v", res)
	}
	if res := run(t, "", "auth", "use", "nope"); !res.Failed {
		t.Fatal("use of an unknown profile should fail")
	}
	if res := run(t, "", "auth", "logout", "--profile", "nope"); !res.Failed {
		t.Fatal("logout of an unknown profile should fail")
	}
	if res := run(t, "", "auth", "logout"); res.Failed {
		t.Fatalf("logout: %+v", res)
	}
	c := loadCreds(t)
	if _, ok := c.Accounts["support"]; ok || c.DefaultAccount != "acme" {
		t.Fatalf("after logout = %+v", c)
	}
	if res := run(t, "", "auth", "logout", "--all"); res.Failed || len(loadCreds(t).Accounts) != 0 {
		t.Fatalf("logout --all: %+v", res)
	}
}

func TestReadSecretFromPipe(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("  s3cret  \nnext\n"))
	if got := readSecret(r); got != "s3cret" {
		t.Fatalf("readSecret = %q", got)
	}
}
