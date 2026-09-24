package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetProfileKeepsExistingDefault(t *testing.T) {
	creds := twoInboxes()
	creds.SetProfile("third", Profile{AuthID: "x"})
	if creds.DefaultAccount != "old-inbox" {
		t.Fatalf("default changed to %q", creds.DefaultAccount)
	}
}

func TestRemoveProfile(t *testing.T) {
	creds := twoInboxes()
	creds.SetProfile("third", Profile{AuthID: "x"})

	creds.RemoveProfile("old-inbox")
	if creds.DefaultAccount != "" {
		t.Fatalf("with 2 profiles left, default should be cleared, got %q", creds.DefaultAccount)
	}

	creds.DefaultAccount = "work"
	creds.RemoveProfile("work")
	if creds.DefaultAccount != "third" {
		t.Fatalf("with 1 profile left it should become default, got %q", creds.DefaultAccount)
	}

	creds.SetProfile("fourth", Profile{AuthID: "y"})
	creds.RemoveProfile("fourth")
	if creds.DefaultAccount != "third" {
		t.Fatalf("removing a non-default profile changed the default to %q", creds.DefaultAccount)
	}
}

func TestSaveAndLoadCredentials(t *testing.T) {
	withCreds(t, nil)
	if c, err := LoadCredentials(); c != nil || err != nil {
		t.Fatalf("missing file: got %v, %v", c, err)
	}
	if err := SaveCredentials(twoInboxes()); err != nil {
		t.Fatal(err)
	}
	path, _ := CredentialsPath()
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("credentials file mode = %v, %v", info.Mode().Perm(), err)
	}
	dir, _ := os.Stat(filepath.Dir(path))
	if dir.Mode().Perm() != 0o700 {
		t.Fatalf("credentials dir mode = %v", dir.Mode().Perm())
	}
	got, err := LoadCredentials()
	if err != nil || got.DefaultAccount != "old-inbox" || got.Accounts["work"].Subdomain != "new-inbox" {
		t.Fatalf("round trip = %+v, %v", got, err)
	}
}

func TestLoadCredentialsCorruptFile(t *testing.T) {
	withCreds(t, nil)
	path, _ := CredentialsPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	_ = os.WriteFile(path, []byte("{not json"), 0o600)
	if _, err := LoadCredentials(); err == nil {
		t.Fatal("expected a parse error, so callers never overwrite the file")
	}
}

func TestFindProfileAndNames(t *testing.T) {
	c := twoInboxes()
	if n, ok := c.FindProfile("work"); !ok || n != "work" {
		t.Errorf("by name: %q %v", n, ok)
	}
	if n, ok := c.FindProfile("new-inbox"); !ok || n != "work" {
		t.Errorf("by subdomain: %q %v", n, ok)
	}
	if _, ok := c.FindProfile("nope"); ok {
		t.Error("unknown profile matched")
	}
	if got := strings.Join(c.Names(), ","); got != "old-inbox,work" {
		t.Errorf("Names() = %s", got)
	}
}

func TestSubdomainOr(t *testing.T) {
	if got := (Profile{}).SubdomainOr("legacy"); got != "legacy" {
		t.Errorf("empty subdomain = %q", got)
	}
	if got := (Profile{Subdomain: "real"}).SubdomainOr("legacy"); got != "real" {
		t.Errorf("stored subdomain = %q", got)
	}
}
