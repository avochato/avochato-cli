package cmd

import (
	"net/http"
	"strings"
	"testing"
)

func TestContactsListFollowsCursorWithAll(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	f.on("GET /v1/contacts", func(r *http.Request) (int, any) {
		if r.URL.Query().Get("after_page") == "" {
			return 200, map[string]any{"contacts": []any{map[string]any{"id": "C1"}}, "next_page": "p2"}
		}
		return 200, map[string]any{"contacts": []any{map[string]any{"id": "C2"}}, "next_page": ""}
	})

	if res := run(t, "", "contacts", "list", "-q"); strings.Join(strings.Fields(res.Stdout), ",") != "C1" {
		t.Fatalf("single page: %q", res.Stdout)
	}
	if res := run(t, "", "contacts", "list", "-q", "--all"); strings.Join(strings.Fields(res.Stdout), ",") != "C1,C2" {
		t.Fatalf("all pages: %q", res.Stdout)
	}
}

func TestContactsShowByPhoneNeedsExactMatch(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	f.ok("GET /v1/contacts", map[string]any{"contacts": []any{
		map[string]any{"id": "C1", "phone": "+15550000001"},
		map[string]any{"id": "C2", "phone": "+16505551234"},
	}})

	var c map[string]any
	decode(t, run(t, "", "contacts", "show", "+16505551234"), &c)
	if c["id"] != "C2" {
		t.Fatalf("show = %v", c)
	}
	if res := run(t, "", "contacts", "show", "+15559999999"); !res.Failed {
		t.Fatal("a phone with no exact match must fail, not return another contact")
	}
}

func TestContactsCreate(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	f.ok("POST /v1/contacts", map[string]any{"contact": map[string]any{"id": "C9", "phone": "+16505551234", "name": "Jane"}})

	res := run(t, "", "contacts", "create", "--phone", "+16505551234", "--name", "Jane", "--email", "j@x.com", "--company", "Acme", "--notes", "vip")
	var c map[string]any
	decode(t, res, &c)
	if res.Failed || c["id"] != "C9" {
		t.Fatalf("create: %+v", res)
	}
	form := f.last(t).Form
	if form.Get("name") != "Jane" || form.Get("email") != "j@x.com" || form.Get("company") != "Acme" || form.Get("notes") != "vip" {
		t.Fatalf("form = %v", form)
	}
}

func TestContactsOptInOut(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	f.ok("POST /v1/contacts/C1/opt_in", map[string]any{})
	f.on("POST /v1/contacts/C1/opt_out", func(*http.Request) (int, any) {
		return 400, "+16505551234 is already opted out from receiving messages."
	})

	res := run(t, "", "contacts", "opt-in", "C1")
	if !res.Failed || !strings.Contains(res.Stderr, "--force") || len(f.requests()) != 0 {
		t.Fatalf("opt-in without --force must refuse: %+v", res)
	}
	var done map[string]any
	res = run(t, "", "contacts", "opt-in", "C1", "--force")
	decode(t, res, &done)
	if res.Failed || done["ok"] != true || done["action"] != "opt-in" || done["id"] != "C1" {
		t.Fatalf("opt-in: %+v", res)
	}
	res = run(t, "", "contacts", "opt-out", "C1")
	if !res.Failed || !strings.Contains(res.Stderr, "already opted out") {
		t.Fatalf("opt-out should report the API error: %+v", res)
	}
}

func TestContactsWritesNeedInboxWhenSeveralSaved(t *testing.T) {
	f := newFakeAPI(t)
	twoInboxes(t, f)
	for _, args := range [][]string{
		{"contacts", "create", "--phone", "+16505551234"}, {"contacts", "opt-in", "C1"}, {"contacts", "opt-out", "C1"},
	} {
		if res := run(t, "", args...); !res.Failed {
			t.Errorf("%v should refuse without --account", args)
		}
	}
	if len(f.requests()) != 0 {
		t.Fatal("no request may be sent when the inbox is ambiguous")
	}
}
