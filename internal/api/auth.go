package api

import "encoding/json"

// WhoamiResponse represents the current authenticated user.
type WhoamiResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// whoamiData is the GET /v1/auth_tokens envelope; only the user part is used.
type whoamiData struct {
	User WhoamiResponse `json:"user"`
}

// Whoami verifies credentials via GET /v1/auth_tokens, the one endpoint that ignores subdomain.
func (c *Client) Whoami() (*WhoamiResponse, error) {
	resp, err := c.Get("auth_tokens", nil)
	if err != nil {
		return nil, err
	}

	var data whoamiData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}
	return &data.User, nil
}
