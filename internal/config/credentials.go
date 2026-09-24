package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// Profile holds the auth credentials and base URL for one account.
type Profile struct {
	AuthID     string `json:"auth_id"`
	AuthSecret string `json:"auth_secret"`
	BaseURL    string `json:"base_url,omitempty"`
	// Subdomain is the inbox subdomain; older profiles leave it empty and use the profile name.
	Subdomain string `json:"subdomain,omitempty"`
}

// SubdomainOr returns the profile's subdomain, or fallback if none is stored.
func (p Profile) SubdomainOr(fallback string) string {
	if p.Subdomain != "" {
		return p.Subdomain
	}
	return fallback
}

// Credentials is the full contents of ~/.avochato/credentials.json.
type Credentials struct {
	DefaultAccount string             `json:"default_account"`
	Accounts       map[string]Profile `json:"accounts"`
}

// SetProfile saves a profile and sets it as default if no default exists.
func (c *Credentials) SetProfile(name string, p Profile) {
	if c.Accounts == nil {
		c.Accounts = make(map[string]Profile)
	}
	c.Accounts[name] = p
	if c.DefaultAccount == "" {
		c.DefaultAccount = name
	}
}

// FindProfile returns the profile name matching key by name or by stored subdomain.
func (c *Credentials) FindProfile(key string) (string, bool) {
	if _, ok := c.Accounts[key]; ok {
		return key, true
	}
	for _, name := range c.Names() {
		if c.Accounts[name].Subdomain == key {
			return name, true
		}
	}
	return "", false
}

// Names returns the saved profile names in sorted order.
func (c *Credentials) Names() []string {
	names := make([]string, 0, len(c.Accounts))
	for name := range c.Accounts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RemoveProfile deletes a profile; a removed default is replaced only when exactly one profile remains.
func (c *Credentials) RemoveProfile(name string) {
	delete(c.Accounts, name)
	if c.DefaultAccount != name {
		return
	}
	c.DefaultAccount = ""
	if len(c.Accounts) == 1 {
		c.DefaultAccount = c.Names()[0]
	}
}

// CredentialsPath returns the path to the credentials file.
func CredentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".avochato", "credentials.json"), nil
}

// LoadCredentials reads ~/.avochato/credentials.json, returning nil, nil if it does not exist.
func LoadCredentials() (*Credentials, error) {
	path, err := CredentialsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}
	if creds.Accounts == nil {
		creds.Accounts = make(map[string]Profile)
	}
	return &creds, nil
}

// SaveCredentials atomically writes ~/.avochato/credentials.json with 0600 permissions.
func SaveCredentials(creds *Credentials) error {
	path, err := CredentialsPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}

	// A fresh temp file gets 0600 even if the old file was wider, and rename never leaves a partial file.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
