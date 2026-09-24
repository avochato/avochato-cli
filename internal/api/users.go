package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// User represents an Avochato account user; CreatedAt and AddedAt are Unix epoch floats.
type User struct {
	ID             string  `json:"id"`
	Email          string  `json:"email"`
	Name           string  `json:"name"`
	ImageURL       string  `json:"image_url"`
	Role           string  `json:"role"`
	Enabled        bool    `json:"enabled"`
	CanMessage     bool    `json:"can_message"`
	CanCall        bool    `json:"can_call"`
	CanViewBilling bool    `json:"can_view_billing"`
	CanViewOrg     bool    `json:"can_view_org"`
	Phone          string  `json:"phone"`
	PhoneFormatted string  `json:"phone_formatted"`
	Require2FA     bool    `json:"require_2fa"`
	MFAAppValid    bool    `json:"mfa_app_valid"`
	PhoneValid     bool    `json:"phone_valid"`
	SignedInAt     string  `json:"signed_in_at"` // ISO 8601 string, unlike created_at/added_at which are Unix floats
	AddedAt        float64 `json:"added_at"`
	CreatedAt      float64 `json:"created_at"`
}

// ListUsers returns a page of users for the account.
func (c *Client) ListUsers(page, limit int) ([]User, error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	params.Set("limit", strconv.Itoa(limit))

	resp, err := c.Get("users", params)
	if err != nil {
		return nil, err
	}

	// Try direct array first, then wrapped object
	var users []User
	if err := json.Unmarshal(resp.Data, &users); err != nil {
		var wrapped struct {
			Users []User `json:"users"`
		}
		if err2 := json.Unmarshal(resp.Data, &wrapped); err2 != nil {
			return nil, fmt.Errorf("failed to parse users: %w", err)
		}
		users = wrapped.Users
	}
	return users, nil
}

// userTarget returns the path segment and params addressing one user; member routes accept email in place of ID.
func userTarget(idOrEmail string) (string, url.Values) {
	if strings.Contains(idOrEmail, "@") {
		return "_", url.Values{"email": {idOrEmail}}
	}
	return esc(idOrEmail), url.Values{}
}

// parseUser reads the {"user": {...}} envelope used by show, create, and update.
func parseUser(resp *APIResponse) (*User, error) {
	var wrapped struct {
		User User `json:"user"`
	}
	if err := json.Unmarshal(resp.Data, &wrapped); err != nil {
		return nil, fmt.Errorf("failed to parse user: %w", err)
	}
	return &wrapped.User, nil
}

// GetUser fetches a single user by hashid or email.
func (c *Client) GetUser(idOrEmail string) (*User, error) {
	seg, params := userTarget(idOrEmail)
	resp, err := c.Get("users/"+seg, params)
	if err != nil {
		return nil, err
	}
	return parseUser(resp)
}

// InviteUser adds a new user to the account.
func (c *Client) InviteUser(email, role, name, phone string) (*User, error) {
	body := url.Values{
		"email": {email},
		"role":  {role},
	}
	if name != "" {
		body.Set("name", name)
	}
	if phone != "" {
		body.Set("phone", phone)
	}

	resp, err := c.Post("users", body)
	if err != nil {
		return nil, err
	}
	return parseUser(resp)
}

// UpdateUser changes a user's role.
func (c *Client) UpdateUser(idOrEmail, role string) (*User, error) {
	seg, body := userTarget(idOrEmail)
	body.Set("role", role)
	resp, err := c.Put("users/"+seg, body)
	if err != nil {
		return nil, err
	}
	return parseUser(resp)
}

// RemoveUser removes a user from the account.
func (c *Client) RemoveUser(idOrEmail string) error {
	seg, params := userTarget(idOrEmail)
	_, err := c.Delete("users/"+seg, params)
	return err
}

// EnableUser re-enables a disabled user.
func (c *Client) EnableUser(idOrEmail string) error {
	return c.userAction(idOrEmail, "enable")
}

// DisableUser disables a user without removing them.
func (c *Client) DisableUser(idOrEmail string) error {
	return c.userAction(idOrEmail, "disable")
}

// ResetPassword sends a password reset email to the user.
func (c *Client) ResetPassword(idOrEmail string) error {
	return c.userAction(idOrEmail, "password_reset")
}

func (c *Client) userAction(idOrEmail, action string) error {
	seg, body := userTarget(idOrEmail)
	_, err := c.Post("users/"+seg+"/"+action, body)
	return err
}
