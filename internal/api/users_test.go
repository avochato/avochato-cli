package api

import (
	"net/http"
	"net/url"
	"testing"
)

func TestUsersEnvelopeAndEmailTarget(t *testing.T) {
	var path, email string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = r.ParseForm()
		email = r.Form.Get("email")
		respondWith(`{"status":200,"data":{"user":{"id":"U1","email":"a@x.com","role":"manager"}}}`)(w, r)
	})

	u, err := c.GetUser("U1")
	if err != nil || u.ID != "U1" || u.Role != "manager" {
		t.Fatalf("GetUser = %+v, %v", u, err)
	}
	if _, err := c.UpdateUser("a@x.com", "manager"); err != nil {
		t.Fatal(err)
	}
	if path != "/v1/users/_" || email != "a@x.com" {
		t.Fatalf("update by email hit %q with email=%q", path, email)
	}
	if err := c.DisableUser("a@x.com"); err != nil {
		t.Fatal(err)
	}
	if path != "/v1/users/_/disable" || email != "a@x.com" {
		t.Fatalf("disable by email hit %q with email=%q", path, email)
	}
	if err := c.RemoveUser("a@x.com"); err != nil {
		t.Fatal(err)
	}
	if path != "/v1/users/_" || email != "a@x.com" {
		t.Fatalf("remove by email hit %q with email=%q", path, email)
	}
}

func TestListUsers(t *testing.T) {
	var page string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		page = r.URL.Query().Get("page")
		respondWith(`{"status":200,"data":{"users":[{"id":"U1","email":"a@x.com","added_at":1700000000.1},{"id":"U2"}]}}`)(w, r)
	})
	users, err := c.ListUsers(3, 25)
	if err != nil {
		t.Fatal(err)
	}
	if page != "3" || len(users) != 2 || users[0].Email != "a@x.com" || users[0].AddedAt != 1700000000.1 {
		t.Fatalf("page=%s users=%+v", page, users)
	}
}

func TestGetUserByEmailUsesMemberRoute(t *testing.T) {
	var path, email string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		path, email = r.URL.Path, r.URL.Query().Get("email")
		respondWith(`{"status":200,"data":{"user":{"id":"U1","email":"a@x.com"}}}`)(w, r)
	})
	u, err := c.GetUser("a@x.com")
	if err != nil || u.ID != "U1" {
		t.Fatalf("GetUser = %+v, %v", u, err)
	}
	if path != "/v1/users/_" || email != "a@x.com" {
		t.Fatalf("hit %s with email=%q", path, email)
	}
}

func TestInviteUser(t *testing.T) {
	var form url.Values
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.PostForm
		respondWith(`{"status":200,"data":{"user":{"id":"U9","email":"new@x.com","role":"member"}}}`)(w, r)
	})
	u, err := c.InviteUser("new@x.com", "member", "New Person", "+16505551234")
	if err != nil || u.ID != "U9" || u.Role != "member" {
		t.Fatalf("InviteUser = %+v, %v", u, err)
	}
	for k, v := range map[string]string{"email": "new@x.com", "role": "member", "name": "New Person", "phone": "+16505551234"} {
		if form.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, form.Get(k), v)
		}
	}
}

func TestUserActionsByID(t *testing.T) {
	var method, path string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		respondWith(`{"status":200,"data":{}}`)(w, r)
	})
	for _, tc := range []struct {
		call       func() error
		method, to string
	}{
		{func() error { return c.EnableUser("U1") }, "POST", "/v1/users/U1/enable"},
		{func() error { return c.DisableUser("U1") }, "POST", "/v1/users/U1/disable"},
		{func() error { return c.ResetPassword("U1") }, "POST", "/v1/users/U1/password_reset"},
		{func() error { return c.RemoveUser("U1") }, "DELETE", "/v1/users/U1"},
	} {
		if err := tc.call(); err != nil {
			t.Fatal(err)
		}
		if method != tc.method || path != tc.to {
			t.Errorf("got %s %s, want %s %s", method, path, tc.method, tc.to)
		}
	}
}
