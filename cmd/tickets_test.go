package cmd

import (
	"net/http"
	"strings"
	"testing"
)

// ticketEcho answers a ticket update with the status that was requested,
// or "Open" when the request would be ignored.
func ticketEcho(r *http.Request) (int, any) {
	status := r.Form.Get("status")
	if status != "Open" && status != "Closed" {
		status = "Open"
	}
	return 200, map[string]any{"tickets": []any{map[string]any{"id": "T1", "status": status, "user_id": r.Form.Get("user_id")}}}
}

func TestTicketsCloseAndOpen(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	f.on("PUT /v1/tickets/T1", ticketEcho)

	var tk map[string]any
	res := run(t, "", "tickets", "close", "T1")
	decode(t, res, &tk)
	if res.Failed || tk["status"] != "Closed" || f.last(t).Form.Get("status") != "Closed" {
		t.Fatalf("close: %+v", res)
	}
	res = run(t, "", "tickets", "open", "T1")
	decode(t, res, &tk)
	if res.Failed || tk["status"] != "Open" {
		t.Fatalf("open: %+v", res)
	}
}

func TestTicketsCloseFailsWhenStatusUnchanged(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	f.ok("PUT /v1/tickets/T1", map[string]any{"tickets": []any{map[string]any{"id": "T1", "status": "Open"}}})

	res := run(t, "", "tickets", "close", "T1")
	if !res.Failed || !strings.Contains(res.Stderr, "still") {
		t.Fatalf("expected failure, got %+v", res)
	}
}

func TestTicketsUpdateAndAssign(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	f.on("PUT /v1/tickets/T1", ticketEcho)

	if res := run(t, "", "tickets", "update", "T1", "--status", "closed", "--assign", "U2", "--unaddressed", "false"); res.Failed {
		t.Fatalf("update: %+v", res)
	}
	form := f.last(t).Form
	if form.Get("status") != "Closed" || form.Get("user_id") != "U2" || form.Get("unaddressed") != "false" {
		t.Fatalf("update form = %v", form)
	}
	if res := run(t, "", "tickets", "assign", "T1", "a@x.com"); res.Failed {
		t.Fatalf("assign: %+v", res)
	}
	if got := f.last(t).Form.Get("user_email"); got != "a@x.com" {
		t.Fatalf("user_email = %q", got)
	}
}

func TestTicketsWritesNeedInboxWhenSeveralSaved(t *testing.T) {
	f := newFakeAPI(t)
	twoInboxes(t, f)
	for _, args := range [][]string{
		{"tickets", "close", "T1"}, {"tickets", "open", "T1"},
		{"tickets", "assign", "T1", "U2"}, {"tickets", "update", "T1", "--status", "open"},
	} {
		if res := run(t, "", args...); !res.Failed {
			t.Errorf("%v should refuse without --account", args)
		}
	}
	if len(f.requests()) != 0 {
		t.Fatal("no request may be sent when the inbox is ambiguous")
	}
}

func TestTicketsListFilters(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	f.ok("GET /v1/users/_", map[string]any{"user": map[string]any{"id": "U9"}})
	f.ok("GET /v1/tickets", map[string]any{"tickets": []any{map[string]any{"id": "T1"}, map[string]any{"id": "T2"}}, "total_count": 2})

	res := run(t, "", "tickets", "list", "-q", "--status", "closed", "--unaddressed", "--user", "a@x.com", "--channel", "sms")
	if res.Failed || strings.Fields(res.Stdout)[1] != "T2" {
		t.Fatalf("list: %+v", res)
	}
	q := f.last(t).Form
	if q.Get("status") != "Closed" || q.Get("unaddressed") != "on" || q.Get("user_id") != "U9" || q.Get("channel") != "sms" {
		t.Fatalf("query = %v", q)
	}
}

func TestTicketsShow(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	f.ok("GET /v1/tickets/T1", map[string]any{"tickets": []any{map[string]any{"id": "T1", "status": "Open"}}})
	var tk map[string]any
	decode(t, run(t, "", "tickets", "show", "T1"), &tk)
	if tk["id"] != "T1" {
		t.Fatalf("show = %v", tk)
	}
	if res := run(t, "", "tickets", "show", "T404"); !res.Failed {
		t.Fatal("unknown ticket should fail")
	}
}
