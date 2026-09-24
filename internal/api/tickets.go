package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Ticket represents an Avochato ticket (conversation) as returned by the v1 API.
type Ticket struct {
	ID          string  `json:"id"`
	UUID        string  `json:"uuid"`
	Contact     string  `json:"contact"` // contact hash_id
	UserID      string  `json:"user_id"` // assigned user hash_id
	Status      string  `json:"status"`  // "open" | "closed"
	Unaddressed bool    `json:"unaddressed"`
	Origin      string  `json:"origin"`
	Channel     string  `json:"channel"`
	HelperBot   bool    `json:"helper_bot"`
	Summary     string  `json:"summary"`
	CreatedAt   float64 `json:"created_at"`
}

// TicketsPage is the response from GET /v1/tickets.
type TicketsPage struct {
	Tickets    []Ticket `json:"tickets"`
	TotalCount int      `json:"total_count"`
	LastKey    string   `json:"last_key"` // cursor for next page; empty on last page
}

// ListTicketsParams holds all filter options for listing tickets.
type ListTicketsParams struct {
	Query       string
	Status      string // "open" | "closed"
	UserID      string // filter by assigned user (hash_id or email)
	Unaddressed string // "true" | "false"
	Order       string // newest | oldest | most_recent_activity | oldest_activity | best_match
	Limit       int
	After       string // cursor from previous LastKey
	Channel     string
}

// ListTickets fetches a page of tickets matching the given filters.
func (c *Client) ListTickets(p ListTicketsParams) (*TicketsPage, error) {
	params := url.Values{}
	if p.Query != "" {
		params.Set("query", p.Query)
	}
	if p.Status != "" {
		status, err := NormalizeStatus(p.Status)
		if err != nil {
			return nil, err
		}
		params.Set("status", status)
	}
	if p.UserID != "" {
		// The list filter only understands user hashids, so resolve emails first.
		userID := p.UserID
		if strings.Contains(userID, "@") {
			u, err := c.GetUser(userID)
			if err != nil {
				return nil, fmt.Errorf("could not find user %q: %w", userID, err)
			}
			userID = u.ID
		}
		params.Set("user_id", userID)
	}
	if p.Unaddressed == "true" {
		params.Set("unaddressed", "on") // FiltersHelper only checks for "on"
	}
	if p.Order != "" {
		params.Set("order", p.Order)
	}
	if p.Limit > 0 {
		params.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.After != "" {
		params.Set("after", p.After)
	}
	if p.Channel != "" {
		params.Set("channel", p.Channel)
	}

	resp, err := c.Get("tickets", params)
	if err != nil {
		return nil, err
	}

	var page TicketsPage
	if err := json.Unmarshal(resp.Data, &page); err != nil {
		return nil, fmt.Errorf("failed to parse tickets: %w", err)
	}
	return &page, nil
}

// GetTicket fetches a single ticket by hash_id or UUID.
func (c *Client) GetTicket(id string) (*Ticket, error) {
	params := url.Values{}

	resp, err := c.Get(fmt.Sprintf("tickets/%s", esc(id)), params)
	if err != nil {
		return nil, err
	}

	var wrapped struct {
		Tickets []Ticket `json:"tickets"`
	}
	if err := json.Unmarshal(resp.Data, &wrapped); err != nil {
		return nil, fmt.Errorf("failed to parse ticket: %w", err)
	}
	if len(wrapped.Tickets) == 0 {
		return nil, &NotFoundError{APIError{StatusCode: 404, Message: "ticket not found"}}
	}
	return &wrapped.Tickets[0], nil
}

// UpdateTicketParams holds the fields that can be updated on a ticket.
type UpdateTicketParams struct {
	// Assign sets the ticket owner. Use "unassign" to remove, "autoassign" to auto-assign.
	Assign string
	// Status sets the ticket status: "open" or "closed".
	Status string
	// Unaddressed marks the conversation as addressed (false) or unaddressed (true).
	Unaddressed string // "true" | "false" | ""
}

// UpdateTicket updates a ticket's owner, status, or addressed state.
func (c *Client) UpdateTicket(id string, p UpdateTicketParams) (*Ticket, error) {
	body := url.Values{}
	if p.Assign != "" {
		if strings.Contains(p.Assign, "@") {
			body.Set("user_email", p.Assign)
		} else {
			body.Set("user_id", p.Assign)
		}
	}
	var status string
	if p.Status != "" {
		var err error
		if status, err = NormalizeStatus(p.Status); err != nil {
			return nil, err
		}
		body.Set("status", status)
	}
	if p.Unaddressed != "" {
		body.Set("unaddressed", p.Unaddressed)
	}

	resp, err := c.Put(fmt.Sprintf("tickets/%s", esc(id)), body)
	if err != nil {
		return nil, err
	}

	var wrapped struct {
		Tickets []Ticket `json:"tickets"`
	}
	if err := json.Unmarshal(resp.Data, &wrapped); err != nil {
		return nil, fmt.Errorf("failed to parse ticket: %w", err)
	}
	if len(wrapped.Tickets) == 0 {
		return nil, &NotFoundError{APIError{StatusCode: 404, Message: "ticket not found"}}
	}
	t := &wrapped.Tickets[0]
	// The API responds 200 even when it ignores a status change, so check it.
	if status != "" && t.Status != status {
		return t, fmt.Errorf("ticket %s status is still %q (requested %q)", t.ID, t.Status, status)
	}
	return t, nil
}

// ticketStatuses mirrors Ticket::STATUS_TYPES; the API matches them case-sensitively.
var ticketStatuses = []string{"New", "Open", "Pending", "Closed"}

// NormalizeStatus maps user input like "closed" to the API's "Closed".
func NormalizeStatus(s string) (string, error) {
	for _, st := range ticketStatuses {
		if strings.EqualFold(s, st) {
			return st, nil
		}
	}
	return "", fmt.Errorf("invalid status %q (use new, open, pending, or closed)", s)
}
