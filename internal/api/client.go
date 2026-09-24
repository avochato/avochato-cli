package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/avochato/avochato-cli/internal/config"
)

// UserAgent is sent with every request; cmd sets it to include the build version.
var UserAgent = "avochato-cli/dev"

// maxBody caps how much of a response body is read.
const maxBody = 10 << 20

// Client is the Avochato v1 API HTTP client.
type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewClient creates a new API client from a resolved Config.
func NewClient(cfg *config.Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			// Never follow redirects: a 307/308 would resend the credential body to another host.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

// Config returns the resolved configuration the client was built with.
func (c *Client) Config() *config.Config {
	return c.cfg
}

// APIResponse is the standard Avochato v1 response envelope.
type APIResponse struct {
	Status int             `json:"status"`
	Data   json.RawMessage `json:"data"`
	Errors errorList       `json:"errors"`
	Page   int             `json:"page"`
	Limit  int             `json:"limit"`
}

// errorList accepts "errors" as either an array of strings or a single string.
type errorList []string

func (e *errorList) UnmarshalJSON(b []byte) error {
	var one string
	if err := json.Unmarshal(b, &one); err == nil {
		if one != "" {
			*e = errorList{one}
		}
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err != nil {
		return err
	}
	*e = many
	return nil
}

// authParams returns the auth + subdomain params injected into every request.
func (c *Client) authParams() url.Values {
	p := url.Values{}
	p.Set("auth_id", c.cfg.AuthID)
	p.Set("auth_secret", c.cfg.AuthSecret)
	if c.cfg.Account != "" {
		p.Set("subdomain", c.cfg.Account)
	}
	return p
}

// Get performs an authenticated GET to /v1/<path>.
func (c *Client) Get(path string, params url.Values) (*APIResponse, error) {
	if params == nil {
		params = url.Values{}
	}
	for k, v := range c.authParams() {
		params[k] = v
	}
	return c.do("GET", path, "?"+params.Encode(), "")
}

// Post performs an authenticated POST to /v1/<path>.
func (c *Client) Post(path string, body url.Values) (*APIResponse, error) {
	if body == nil {
		body = url.Values{}
	}
	for k, v := range c.authParams() {
		body[k] = v
	}
	return c.do("POST", path, "", body.Encode())
}

// Put performs an authenticated PUT to /v1/<path>.
func (c *Client) Put(path string, body url.Values) (*APIResponse, error) {
	if body == nil {
		body = url.Values{}
	}
	for k, v := range c.authParams() {
		body[k] = v
	}
	return c.do("PUT", path, "", body.Encode())
}

// Delete performs an authenticated DELETE to /v1/<path>.
func (c *Client) Delete(path string, params url.Values) (*APIResponse, error) {
	if params == nil {
		params = url.Values{}
	}
	for k, v := range c.authParams() {
		params[k] = v
	}
	return c.do("DELETE", path, "?"+params.Encode(), "")
}

// esc escapes a caller-supplied ID as one path segment so it cannot reach another endpoint.
func esc(id string) string {
	return url.PathEscape(id)
}

// maxRetryAfter caps how long a 429 Retry-After header can make us sleep.
const maxRetryAfter = 60

// secretParam matches the auth_id and auth_secret values in a URL or form body.
var secretParam = regexp.MustCompile(`(auth_(?:id|secret)=)[^&\s]*`)

// redact hides credentials in debug and error output.
func redact(s string) string {
	return secretParam.ReplaceAllString(s, "${1}REDACTED")
}

func (c *Client) do(method, path, query, form string) (*APIResponse, error) {
	for _, seg := range strings.Split(path, "/") {
		if seg == "" || seg == "." || seg == ".." {
			return nil, fmt.Errorf("invalid API path %q", path)
		}
	}
	u := fmt.Sprintf("%s/v1/%s%s", c.cfg.BaseURL, path, query)
	return c.send(method, u, form, true)
}

func (c *Client) send(method, u, form string, retry bool) (*APIResponse, error) {
	var body io.Reader
	if form != "" {
		body = strings.NewReader(form)
	}
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, fmt.Errorf("invalid request: %s", redact(err.Error()))
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// The error embeds the request URL, which carries credentials on GET/DELETE.
		return nil, fmt.Errorf("network error: %s", redact(err.Error()))
	}
	defer resp.Body.Close()

	// Auto-retry once on 429; the form string is reused since the first reader was consumed.
	if resp.StatusCode == 429 && retry {
		retryAfter := 10
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if n, err := strconv.Atoi(ra); err == nil && n >= 0 {
				retryAfter = min(n, maxRetryAfter)
			}
		}
		time.Sleep(time.Duration(retryAfter) * time.Second)
		return c.send(method, u, form, false)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if os.Getenv("AVOCHATO_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "[debug] %s %s → %d\n%s\n", method, redact(u), resp.StatusCode, string(raw))
	}

	var apiResp APIResponse
	if err := json.Unmarshal(raw, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response (status %d): %w", resp.StatusCode, err)
	}

	if resp.StatusCode >= 400 {
		msg := "unknown error"
		if len(apiResp.Errors) > 0 {
			msg = strings.Join(apiResp.Errors, "; ")
		}
		retryAfter := 0
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			retryAfter, _ = strconv.Atoi(ra)
		}
		return nil, newTypedError(resp.StatusCode, msg, retryAfter)
	}

	return &apiResp, nil
}
