package config

import (
	"fmt"
	"net/url"
	"os"
)

const defaultBaseURL = "https://www.avochato.com"

// Config holds the resolved runtime configuration for one API call.
type Config struct {
	AuthID     string
	AuthSecret string
	Account    string
	BaseURL    string

	// Profile is the saved profile the credentials came from ("" if none).
	Profile string
	// Explicit is true when the inbox was chosen for this invocation rather than defaulted.
	Explicit bool
	// ProfileCount is the number of saved profiles.
	ProfileCount int
}

// Resolve layers defaults, the credentials file, environment variables, and CLI flags.
func Resolve(cliAccount, cliBaseURL string) (*Config, error) {
	cfg := &Config{BaseURL: defaultBaseURL}

	creds, err := LoadCredentials()
	if err != nil {
		return nil, err
	}
	if creds != nil {
		cfg.ProfileCount = len(creds.Accounts)

		profileName := creds.DefaultAccount
		if ev := os.Getenv("AVOCHATO_PROFILE"); ev != "" {
			name, ok := creds.FindProfile(ev)
			if !ok {
				return nil, fmt.Errorf("AVOCHATO_PROFILE=%q does not match a saved profile (see `avochato auth list`)", ev)
			}
			profileName = name
			cfg.Explicit = true
		}
		// --account names a saved profile, or else an inbox subdomain for the current credentials.
		if cliAccount != "" {
			if name, ok := creds.FindProfile(cliAccount); ok {
				profileName = name
				cfg.Explicit = true
				cliAccount = ""
			}
		}
		if p, ok := creds.Accounts[profileName]; ok {
			cfg.AuthID = p.AuthID
			cfg.AuthSecret = p.AuthSecret
			cfg.Account = p.SubdomainOr(profileName)
			cfg.Profile = profileName
			if p.BaseURL != "" {
				cfg.BaseURL = p.BaseURL
			}
		}
	}

	envID, envSecret := os.Getenv("AVOCHATO_AUTH_ID"), os.Getenv("AVOCHATO_AUTH_SECRET")
	if envID != "" {
		cfg.AuthID = envID
	}
	if envSecret != "" {
		cfg.AuthSecret = envSecret
	}
	if v := os.Getenv("AVOCHATO_ACCOUNT"); v != "" {
		cfg.Account = v
		cfg.Explicit = true
	}
	// AVOCHATO_BASE_URL only applies to env credentials, so saved secrets can't be pointed at another host.
	if v := os.Getenv("AVOCHATO_BASE_URL"); v != "" && envID != "" && envSecret != "" {
		cfg.BaseURL = v
		cfg.Profile = ""
	}

	if cliAccount != "" {
		cfg.Account = cliAccount
		cfg.Explicit = true
	}
	if cliBaseURL != "" {
		cfg.BaseURL = cliBaseURL
	}

	return cfg, nil
}

// ValidateBaseURL rejects base URLs that would send credentials in cleartext (http is localhost-only).
func ValidateBaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid base URL %q", raw)
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		switch u.Hostname() {
		case "localhost", "127.0.0.1", "::1":
			return nil
		}
	}
	return fmt.Errorf("base URL %q must use https (http is only allowed for localhost)", raw)
}

// RequireExplicitInbox errors when several profiles are saved and none was chosen for this command.
func (c *Config) RequireExplicitInbox() error {
	if c.Explicit || c.ProfileCount <= 1 {
		return nil
	}
	return fmt.Errorf("%d inboxes are saved and none was chosen for this command (default is %q); "+
		"pass --account <inbox> or set AVOCHATO_PROFILE", c.ProfileCount, c.Account)
}
