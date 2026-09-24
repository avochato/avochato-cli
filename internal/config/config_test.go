package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withCreds points HOME at a temp dir holding creds and clears the env variables Resolve reads.
func withCreds(t *testing.T, creds *Credentials) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	for _, k := range []string{"AVOCHATO_PROFILE", "AVOCHATO_AUTH_ID", "AVOCHATO_AUTH_SECRET", "AVOCHATO_ACCOUNT", "AVOCHATO_BASE_URL"} {
		t.Setenv(k, "")
	}
	if creds != nil {
		if err := SaveCredentials(creds); err != nil {
			t.Fatal(err)
		}
	}
}

func twoInboxes() *Credentials {
	return &Credentials{
		DefaultAccount: "old-inbox",
		Accounts: map[string]Profile{
			"old-inbox": {AuthID: "old-id", AuthSecret: "old-secret"},
			"work":      {AuthID: "new-id", AuthSecret: "new-secret", Subdomain: "new-inbox", BaseURL: "https://staging.avochato.com"},
		},
	}
}

func TestResolveDefaultProfileIsNotExplicit(t *testing.T) {
	withCreds(t, twoInboxes())
	cfg, err := Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Account != "old-inbox" || cfg.AuthID != "old-id" || cfg.Explicit || cfg.ProfileCount != 2 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	err = cfg.RequireExplicitInbox()
	if err == nil || !strings.Contains(err.Error(), `default is "old-inbox"`) {
		t.Fatalf("expected ambiguity error naming the default, got %v", err)
	}
}

func TestResolveAccountFlagSelectsProfileCredentials(t *testing.T) {
	withCreds(t, twoInboxes())
	for _, key := range []string{"work", "new-inbox"} {
		cfg, err := Resolve(key, "")
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Profile != "work" || cfg.AuthID != "new-id" || cfg.Account != "new-inbox" ||
			cfg.BaseURL != "https://staging.avochato.com" || !cfg.Explicit {
			t.Fatalf("--account %s: unexpected config: %+v", key, cfg)
		}
		if err := cfg.RequireExplicitInbox(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResolveAccountFlagUnknownSubdomainKeepsDefaultCredentials(t *testing.T) {
	withCreds(t, twoInboxes())
	cfg, err := Resolve("third-inbox", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AuthID != "old-id" || cfg.Account != "third-inbox" || !cfg.Explicit {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestResolveProfileEnv(t *testing.T) {
	withCreds(t, twoInboxes())
	t.Setenv("AVOCHATO_PROFILE", "work")
	cfg, err := Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AuthID != "new-id" || cfg.Account != "new-inbox" || !cfg.Explicit {
		t.Fatalf("unexpected config: %+v", cfg)
	}

	t.Setenv("AVOCHATO_PROFILE", "missing")
	if _, err := Resolve("", ""); err == nil {
		t.Fatal("expected error for unknown AVOCHATO_PROFILE")
	}
}

func TestResolveAccountEnvIsExplicit(t *testing.T) {
	withCreds(t, twoInboxes())
	t.Setenv("AVOCHATO_ACCOUNT", "new-inbox")
	cfg, err := Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Account != "new-inbox" || !cfg.Explicit {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestBaseURLEnvOnlyAppliesToEnvCredentials(t *testing.T) {
	withCreds(t, twoInboxes())
	t.Setenv("AVOCHATO_BASE_URL", "https://attacker.example")
	cfg, err := Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != defaultBaseURL {
		t.Fatalf("saved profile secret would be sent to %q", cfg.BaseURL)
	}

	t.Setenv("AVOCHATO_AUTH_ID", "env-id")
	t.Setenv("AVOCHATO_AUTH_SECRET", "env-secret")
	cfg, err = Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL != "https://attacker.example" || cfg.AuthID != "env-id" || cfg.Profile != "" {
		t.Fatalf("env credentials should use env base URL: %+v", cfg)
	}
}

func TestSaveCredentialsFixesPermissions(t *testing.T) {
	withCreds(t, nil)
	path, _ := CredentialsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SaveCredentials(twoInboxes()); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil || fi.Mode().Perm() != 0600 {
		t.Fatalf("mode = %v, err = %v", fi.Mode(), err)
	}
}

func TestRequireExplicitInboxSingleProfile(t *testing.T) {
	withCreds(t, &Credentials{
		DefaultAccount: "only",
		Accounts:       map[string]Profile{"only": {AuthID: "id", AuthSecret: "secret"}},
	})
	cfg, err := Resolve("", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.RequireExplicitInbox(); err != nil {
		t.Fatalf("single profile should not need --account: %v", err)
	}
}

func TestValidateBaseURL(t *testing.T) {
	for raw, ok := range map[string]bool{
		"https://www.avochato.com":    true,
		"http://localhost:3000":       true,
		"http://127.0.0.1:8765":       true,
		"http://staging.avochato.com": false,
		"ftp://www.avochato.com":      false,
		"www.avochato.com":            false,
	} {
		if err := ValidateBaseURL(raw); (err == nil) != ok {
			t.Errorf("ValidateBaseURL(%q) error = %v, want ok=%v", raw, err, ok)
		}
	}
}
