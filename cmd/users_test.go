package cmd

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/avochato/avochato-cli/internal/config"
)

func TestUsersListPagesAndTrims(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	f.on("GET /v1/users", func(r *http.Request) (int, any) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		var users []map[string]any
		for i := (page-1)*25 + 1; i <= page*25 && i <= 60; i++ {
			users = append(users, map[string]any{"id": fmt.Sprintf("U%d", i)})
		}
		return 200, map[string]any{"users": users}
	})

	if res := run(t, "", "users", "list", "-q"); len(strings.Fields(res.Stdout)) != 30 {
		t.Fatalf("default limit: %d ids", len(strings.Fields(res.Stdout)))
	}
	if res := run(t, "", "users", "list", "-q", "--all"); len(strings.Fields(res.Stdout)) != 60 {
		t.Fatalf("--all: %d ids", len(strings.Fields(res.Stdout)))
	}
}

func TestUsersShowByEmail(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	f.ok("GET /v1/users/_", map[string]any{"user": map[string]any{"id": "U1", "email": "a@x.com", "role": "manager"}})

	var u map[string]any
	decode(t, run(t, "", "users", "show", "a@x.com"), &u)
	if u["role"] != "manager" || f.last(t).Form.Get("email") != "a@x.com" {
		t.Fatalf("show = %v", u)
	}
}

func TestUsersInviteAndUpdate(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	f.ok("POST /v1/users", map[string]any{"user": map[string]any{"id": "U9", "role": "member"}})
	f.ok("PUT /v1/users/_", map[string]any{"user": map[string]any{"id": "U9", "role": "manager"}})

	if res := run(t, "", "users", "invite", "new@x.com", "--name", "New"); res.Failed {
		t.Fatalf("invite: %+v", res)
	}
	if form := f.last(t).Form; form.Get("email") != "new@x.com" || form.Get("role") != "member" || form.Get("name") != "New" {
		t.Fatalf("invite form = %v", form)
	}
	if res := run(t, "", "users", "update", "new@x.com", "--role", "manager"); res.Failed {
		t.Fatalf("update: %+v", res)
	}
	if res := run(t, "", "users", "invite", "x@x.com", "--role", "admin"); !res.Failed || !strings.Contains(res.Stderr, "invalid role") {
		t.Fatalf("invalid role should fail: %+v", res)
	}
}

func TestUsersRemoveNeedsForceWithoutTerminal(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	f.ok("DELETE /v1/users/_", map[string]any{})

	res := run(t, "", "users", "remove", "a@x.com")
	if !res.Failed || !strings.Contains(res.Stderr, "--force") || len(f.requests()) != 0 {
		t.Fatalf("remove without --force must refuse: %+v", res)
	}
	var done map[string]any
	res = run(t, "", "users", "remove", "a@x.com", "--force")
	decode(t, res, &done)
	if res.Failed || done["action"] != "remove" || f.last(t).Method != "DELETE" {
		t.Fatalf("remove --force: %+v", res)
	}
}

func TestUsersEnableDisableReset(t *testing.T) {
	f := newFakeAPI(t)
	oneInbox(t, f)
	for _, a := range []string{"enable", "disable", "password_reset"} {
		f.ok("POST /v1/users/U1/"+a, map[string]any{})
	}
	for cmd, path := range map[string]string{"enable": "enable", "disable": "disable", "reset-password": "password_reset"} {
		var done map[string]any
		res := run(t, "", "users", cmd, "U1")
		decode(t, res, &done)
		if res.Failed || done["action"] != cmd || f.last(t).Path != "/v1/users/U1/"+path {
			t.Errorf("%s: %+v", cmd, res)
		}
	}
}

func TestClientErrorsWhenNotLoggedIn(t *testing.T) {
	home(t, nil)
	res := run(t, "", "users", "list")
	if !res.Failed || !strings.Contains(res.Stderr, "avochato login") {
		t.Fatalf("got %+v", res)
	}

	f := newFakeAPI(t)
	home(t, &config.Credentials{Accounts: map[string]config.Profile{
		"a": {AuthID: "1", AuthSecret: "1", BaseURL: f.URL}, "b": {AuthID: "2", AuthSecret: "2", BaseURL: f.URL},
	}})
	res = run(t, "", "users", "list")
	if !res.Failed || !strings.Contains(res.Stderr, "no default profile") {
		t.Fatalf("got %+v", res)
	}
}

func TestPlainHTTPBaseURLRefused(t *testing.T) {
	home(t, &config.Credentials{DefaultAccount: "a", Accounts: map[string]config.Profile{
		"a": {AuthID: "1", AuthSecret: "1", BaseURL: "http://api.example.com"},
	}})
	if res := run(t, "", "users", "list"); !res.Failed || !strings.Contains(res.Stderr, "https") {
		t.Fatalf("got %+v", res)
	}
}
