package api

import (
	"net/http"
	"net/url"
	"testing"
)

func TestSendMessageParams(t *testing.T) {
	var form map[string]string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = map[string]string{}
		for k := range r.PostForm {
			form[k] = r.PostForm.Get(k)
		}
		respondWith(`{"status":200,"data":{"message":{"event_id":"E1","to":"+16505551234","from":"+15550001111","status":"queued"}}}`)(w, r)
	})

	msg, err := c.SendMessage(SendMessageParams{
		To: "+16505551234", Text: "hi", From: "+15550001111", MediaURL: "https://x/y.png",
		SendAsEmail: "a@x.com", DelaySeconds: 30, ScheduledFor: 1700000000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if msg.ID != "E1" || msg.Status != "queued" {
		t.Fatalf("message = %+v", msg)
	}
	want := map[string]string{
		"phone": "+16505551234", "message": "hi", "from": "+15550001111", "media_url": "https://x/y.png",
		"send_as_user_email": "a@x.com", "delay": "30", "scheduled_for": "1700000000", "subdomain": "inbox",
	}
	for k, v := range want {
		if form[k] != v {
			t.Errorf("%s = %q, want %q", k, form[k], v)
		}
	}
	for _, k := range []string{"send_as_user_id"} {
		if _, ok := form[k]; ok {
			t.Errorf("unset param %s was sent", k)
		}
	}
}

func TestSendMessageOmitsEmptyOptionalParams(t *testing.T) {
	var form url.Values
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.PostForm
		respondWith(`{"status":200,"data":{"message":{"event_id":"E1"}}}`)(w, r)
	})
	if _, err := c.SendMessage(SendMessageParams{To: "+16505551234", SendAsUserID: "U1"}); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"from", "media_url", "send_as_user_email", "delay", "scheduled_for"} {
		if _, ok := form[k]; ok {
			t.Errorf("unset param %s was sent", k)
		}
	}
	if form["send_as_user_id"][0] != "U1" {
		t.Errorf("send_as_user_id = %v", form["send_as_user_id"])
	}
}

func TestListMessages(t *testing.T) {
	var page, limit string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		page, limit = r.URL.Query().Get("page"), r.URL.Query().Get("limit")
		respondWith(`{"status":200,"data":{"messages":[{"event_id":"E1","message":"hi","sent_at":1700000000.5},{"event_id":"E2"}]}}`)(w, r)
	})
	msgs, err := c.ListMessages(2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if page != "2" || limit != "10" {
		t.Fatalf("page=%s limit=%s", page, limit)
	}
	if len(msgs) != 2 || msgs[0].Body != "hi" || msgs[0].SentAt != 1700000000.5 {
		t.Fatalf("messages = %+v", msgs)
	}
}

func TestSendMessageAPIError(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":400,"errors":["Phone number '+1' doesn't exist in this account"]}`))
	})
	if _, err := c.SendMessage(SendMessageParams{To: "+16505551234", Text: "hi", From: "+1"}); err == nil {
		t.Fatal("expected error")
	}
}
