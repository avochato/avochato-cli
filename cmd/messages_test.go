package cmd

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestMessagesSendRefusesWithoutInboxWhenSeveralSaved(t *testing.T) {
	f := newFakeAPI(t)
	twoInboxes(t, f)

	res := run(t, "", "messages", "send", "--to", "+16505551234", "--text", "hi")
	if !res.Failed || !strings.Contains(res.Stderr, `default is "acme"`) {
		t.Fatalf("expected refusal, got %+v", res)
	}
	if len(f.requests()) != 0 {
		t.Fatal("no request may be sent when the inbox is ambiguous")
	}
}

func TestMessagesSendUsesChosenInbox(t *testing.T) {
	f := newFakeAPI(t)
	twoInboxes(t, f)
	f.ok("POST /v1/messages", map[string]any{"message": map[string]any{"event_id": "E1", "to": "+16505551234", "status": "queued"}})

	res := run(t, "", "--account", "support", "messages", "send", "--to", "+16505551234", "--text", "hi")
	if res.Failed {
		t.Fatalf("send failed: %+v", res)
	}
	req := f.last(t)
	if req.Form.Get("subdomain") != "support" || req.Form.Get("auth_id") != "id2" || req.Form.Get("message") != "hi" {
		t.Fatalf("request = %+v", req)
	}
	if !strings.Contains(res.Stderr, "Sending from inbox: support") {
		t.Fatalf("stderr = %q", res.Stderr)
	}
	var msg map[string]any
	decode(t, res, &msg)
	if msg["event_id"] != "E1" {
		t.Fatalf("stdout = %s", res.Stdout)
	}
}

func TestMessagesSendProfileEnvChoosesInbox(t *testing.T) {
	f := newFakeAPI(t)
	twoInboxes(t, f)
	f.ok("POST /v1/messages", map[string]any{"message": map[string]any{"event_id": "E1"}})
	t.Setenv("AVOCHATO_PROFILE", "support")

	if res := run(t, "", "messages", "send", "--to", "+16505551234", "--text", "hi"); res.Failed {
		t.Fatalf("send failed: %+v", res)
	}
	if got := f.last(t).Form.Get("subdomain"); got != "support" {
		t.Fatalf("subdomain = %q", got)
	}
}

func TestMessagesSendNeedsTextOrMedia(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	res := run(t, "", "messages", "send", "--to", "+16505551234")
	if !res.Failed || !strings.Contains(res.Stderr, "--text, --media-url") {
		t.Fatalf("got %+v", res)
	}

	f.ok("POST /v1/messages", map[string]any{"message": map[string]any{"event_id": "E2"}})
	if res := run(t, "", "messages", "send", "--to", "+16505551234", "--media-url", "https://x/y.png"); res.Failed {
		t.Fatalf("media-only send failed: %+v", res)
	}
	if f.last(t).Form.Get("media_url") != "https://x/y.png" {
		t.Fatal("media_url not sent")
	}
}

func TestMessagesSendRequiresTo(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	if res := run(t, "", "messages", "send", "--text", "hi"); !res.Failed {
		t.Fatal("missing --to should fail")
	}
}

// pagedMessages serves total messages, 25 per page, ignoring limit like the API.
func pagedMessages(total int) route {
	return func(r *http.Request) (int, any) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		var msgs []map[string]any
		for i := (page-1)*25 + 1; i <= page*25 && i <= total; i++ {
			msgs = append(msgs, map[string]any{"event_id": fmt.Sprintf("E%d", i)})
		}
		return 200, map[string]any{"messages": msgs}
	}
}

func TestMessagesListLimitAndAll(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	f.on("GET /v1/messages", pagedMessages(60))

	for _, tc := range []struct {
		args []string
		want int
	}{
		{[]string{"messages", "list", "-q"}, 20},
		{[]string{"messages", "list", "-q", "--limit", "10"}, 10},
		{[]string{"messages", "list", "-q", "--limit", "40"}, 40},
		{[]string{"messages", "list", "-q", "--all"}, 60},
	} {
		res := run(t, "", tc.args...)
		if got := len(strings.Fields(res.Stdout)); res.Failed || got != tc.want {
			t.Errorf("%v: got %d ids (failed=%v)", tc.args, got, res.Failed)
		}
	}
}

func TestMessagesListJSON(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	f.on("GET /v1/messages", pagedMessages(3))
	var msgs []map[string]any
	decode(t, run(t, "", "messages", "list", "--json"), &msgs)
	if len(msgs) != 3 {
		t.Fatalf("got %d messages", len(msgs))
	}
}
