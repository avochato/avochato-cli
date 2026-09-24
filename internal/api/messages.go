package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// Message represents an Avochato message event as returned by the v1 API.
type Message struct {
	ID        string  `json:"event_id"` // Event hash_id (from Event#api_json)
	Direction string  `json:"direction"`
	Status    string  `json:"status"`
	Body      string  `json:"message"` // field is "message" not "body"
	From      string  `json:"from"`
	To        string  `json:"to"`
	SentAt    float64 `json:"sent_at"`
	ContactID string  `json:"contact_id"`
	TicketID  string  `json:"ticket_id"`
}

// SendMessageParams holds all options for sending a message.
type SendMessageParams struct {
	To           string // recipient phone (E.164), required
	Text         string // message body, required unless MediaURL set
	From         string // sender phone number (must exist in account)
	MediaURL     string // MMS attachment URL
	SendAsEmail  string // send on behalf of this user email
	SendAsUserID string // send on behalf of this user ID
	ScheduledFor int64  // Unix timestamp for scheduled send
	DelaySeconds int    // seconds from now to send
}

// SendMessage sends an outbound message via POST /v1/messages.
func (c *Client) SendMessage(p SendMessageParams) (*Message, error) {
	body := url.Values{
		"phone":   {p.To},
		"message": {p.Text},
	}
	if p.From != "" {
		body.Set("from", p.From)
	}
	if p.MediaURL != "" {
		body.Set("media_url", p.MediaURL)
	}
	if p.SendAsEmail != "" {
		body.Set("send_as_user_email", p.SendAsEmail)
	}
	if p.SendAsUserID != "" {
		body.Set("send_as_user_id", p.SendAsUserID)
	}
	if p.ScheduledFor > 0 {
		body.Set("scheduled_for", strconv.FormatInt(p.ScheduledFor, 10))
	}
	if p.DelaySeconds > 0 {
		body.Set("delay", strconv.Itoa(p.DelaySeconds))
	}

	resp, err := c.Post("messages", body)
	if err != nil {
		return nil, err
	}

	var wrapped struct {
		Message Message `json:"message"`
	}
	if err := json.Unmarshal(resp.Data, &wrapped); err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}
	return &wrapped.Message, nil
}

// ListMessages returns a page of message events for the account.
func (c *Client) ListMessages(page, limit int) ([]Message, error) {
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	params.Set("limit", strconv.Itoa(limit))

	resp, err := c.Get("messages", params)
	if err != nil {
		return nil, err
	}

	var wrapped struct {
		Messages []Message `json:"messages"`
	}
	if err := json.Unmarshal(resp.Data, &wrapped); err != nil {
		return nil, fmt.Errorf("failed to parse messages: %w", err)
	}
	return wrapped.Messages, nil
}
