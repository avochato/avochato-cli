package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Contact represents an Avochato contact as returned by the v1 API.
type Contact struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Phone       string   `json:"phone"`
	Email       string   `json:"email"`
	Company     string   `json:"company"`
	Notes       string   `json:"notes"`
	OtherPhone  string   `json:"other_phone"`
	Street      string   `json:"street"`
	City        string   `json:"city"`
	State       string   `json:"state"`
	Zip         string   `json:"zip"`
	Country     string   `json:"country"`
	Tags        []string `json:"tags"`
	OptedOut    bool     `json:"opted_out"`
	DoubleOptIn bool     `json:"double_opted_in"`
	Muted       bool     `json:"muted"`
	Blocked     bool     `json:"blocked"`
	Visible     bool     `json:"visible"`
	UserID      string   `json:"user_id"`
	CreatedAt   float64  `json:"created_at"`
}

// ContactsPage is the paginated response from GET /v1/contacts.
type ContactsPage struct {
	Contacts []Contact `json:"contacts"`
	Size     int       `json:"size"`
	NextPage string    `json:"next_page"` // cursor for the next page; empty when on the last page
}

// ListContacts fetches a page of contacts. Pass afterCursor="" for the first page.
func (c *Client) ListContacts(query string, limit int, afterCursor string) (*ContactsPage, error) {
	params := url.Values{}
	if query != "" {
		params.Set("query", query)
	} else {
		params.Set("query", "*") // wildcard returns all contacts
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if afterCursor != "" {
		params.Set("after_page", afterCursor)
	}

	resp, err := c.Get("contacts", params)
	if err != nil {
		return nil, err
	}

	var page ContactsPage
	if err := json.Unmarshal(resp.Data, &page); err != nil {
		return nil, fmt.Errorf("failed to parse contacts: %w", err)
	}
	return &page, nil
}

// GetContact fetches a single contact by hashid or phone number.
func (c *Client) GetContact(idOrPhone string) (*Contact, error) {
	if looksLikePhone(idOrPhone) {
		return c.findContactByPhone(idOrPhone)
	}

	resp, err := c.Get("contacts/"+esc(idOrPhone), nil)
	if err != nil {
		return nil, err
	}

	// show returns { contacts: [...] }
	var wrapped struct {
		Contacts []Contact `json:"contacts"`
	}
	if err := json.Unmarshal(resp.Data, &wrapped); err != nil {
		return nil, fmt.Errorf("failed to parse contact: %w", err)
	}
	if len(wrapped.Contacts) == 0 {
		return nil, &NotFoundError{APIError{StatusCode: 404, Message: "contact not found"}}
	}
	return &wrapped.Contacts[0], nil
}

// findContactByPhone searches for the number and returns only an exact match, since search can return other contacts.
func (c *Client) findContactByPhone(phone string) (*Contact, error) {
	want := phoneDigits(phone)
	page, err := c.ListContacts(phone, 100, "")
	if err != nil {
		return nil, err
	}
	for i := range page.Contacts {
		if phoneDigits(page.Contacts[i].Phone) == want {
			return &page.Contacts[i], nil
		}
	}
	return nil, &NotFoundError{APIError{StatusCode: 404, Message: fmt.Sprintf("no contact with phone %s", phone)}}
}

// phoneDigits reduces a phone number to its digits, treating 10 digits as US (+1).
func phoneDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	d := b.String()
	if len(d) == 10 {
		d = "1" + d
	}
	return d
}

// looksLikePhone reports whether s is all digits (optional leading "+") rather than a contact hashid.
func looksLikePhone(s string) bool {
	digits := strings.TrimPrefix(s, "+")
	if digits == "" {
		return false
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// CreateContactParams holds the fields for creating a contact.
type CreateContactParams struct {
	Phone   string
	Name    string
	Email   string
	Company string
	Notes   string
}

// CreateContact creates or updates a contact via POST /v1/contacts.
func (c *Client) CreateContact(p CreateContactParams) (*Contact, error) {
	body := url.Values{"phone": {p.Phone}}
	if p.Name != "" {
		body.Set("name", p.Name)
	}
	if p.Email != "" {
		body.Set("email", p.Email)
	}
	if p.Company != "" {
		body.Set("company", p.Company)
	}
	if p.Notes != "" {
		body.Set("notes", p.Notes)
	}

	resp, err := c.Post("contacts", body)
	if err != nil {
		return nil, err
	}

	// ContactImportable responds {"contact": {...}} for a single upsert.
	var wrapped struct {
		Contact *Contact `json:"contact"`
	}
	if err := json.Unmarshal(resp.Data, &wrapped); err != nil {
		return nil, fmt.Errorf("failed to parse contact: %w", err)
	}
	if wrapped.Contact == nil {
		return nil, fmt.Errorf("no contact returned")
	}
	return wrapped.Contact, nil
}

// OptOut opts a contact out of receiving messages.
func (c *Client) OptOutContact(id string) error {
	_, err := c.Post(fmt.Sprintf("contacts/%s/opt_out", esc(id)), nil)
	return err
}

// OptIn opts a contact back in to receive messages.
func (c *Client) OptInContact(id string) error {
	_, err := c.Post(fmt.Sprintf("contacts/%s/opt_in", esc(id)), nil)
	return err
}
