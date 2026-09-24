package api

import (
	"errors"
	"net/http"
	"net/url"
	"testing"
)

func TestUpdateTicketSendsCapitalizedStatus(t *testing.T) {
	var got string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		got = r.Form.Get("status")
		respondWith(`{"status":200,"data":{"tickets":[{"id":"t1","status":"Closed"}]}}`)(w, r)
	})
	if _, err := c.UpdateTicket("t1", UpdateTicketParams{Status: "closed"}); err != nil {
		t.Fatal(err)
	}
	if got != "Closed" {
		t.Fatalf("status param = %q, want Closed", got)
	}
}

func TestUpdateTicketFailsWhenStatusUnchanged(t *testing.T) {
	c := testClient(t, respondWith(`{"status":200,"data":{"tickets":[{"id":"t1","status":"Open"}]}}`))
	if _, err := c.UpdateTicket("t1", UpdateTicketParams{Status: "closed"}); err == nil {
		t.Fatal("expected error when the API leaves the status unchanged")
	}
	if _, err := c.UpdateTicket("t1", UpdateTicketParams{Status: "archived"}); err == nil {
		t.Fatal("expected error for an invalid status")
	}
}

func TestListTicketsFilters(t *testing.T) {
	var q map[string]string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/users/_" {
			respondWith(`{"status":200,"data":{"user":{"id":"U9","email":"a@x.com"}}}`)(w, r)
			return
		}
		q = map[string]string{}
		for k := range r.URL.Query() {
			q[k] = r.URL.Query().Get(k)
		}
		respondWith(`{"status":200,"data":{"tickets":[]}}`)(w, r)
	})
	if _, err := c.ListTickets(ListTicketsParams{Status: "closed", Unaddressed: "true", UserID: "a@x.com"}); err != nil {
		t.Fatal(err)
	}
	if q["status"] != "Closed" || q["unaddressed"] != "on" || q["user_id"] != "U9" || q["user_email"] != "" {
		t.Fatalf("unexpected query: %v", q)
	}
}

func TestGetTicket(t *testing.T) {
	var path string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		respondWith(`{"status":200,"data":{"tickets":[{"id":"T1","status":"Open","unaddressed":true}]}}`)(w, r)
	})
	tk, err := c.GetTicket("T1")
	if err != nil || tk.ID != "T1" || !tk.Unaddressed {
		t.Fatalf("GetTicket = %+v, %v", tk, err)
	}
	if path != "/v1/tickets/T1" {
		t.Fatalf("path = %s", path)
	}
}

func TestGetTicketNotFound(t *testing.T) {
	c := testClient(t, respondWith(`{"status":200,"data":{"tickets":[]}}`))
	_, err := c.GetTicket("T404")
	var nf *NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("err = %T %v, want *NotFoundError", err, err)
	}
}

func TestUpdateTicketAssignAndUnaddressed(t *testing.T) {
	var form url.Values
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = r.PostForm
		respondWith(`{"status":200,"data":{"tickets":[{"id":"T1","status":"Open"}]}}`)(w, r)
	})
	if _, err := c.UpdateTicket("T1", UpdateTicketParams{Assign: "a@x.com", Unaddressed: "false"}); err != nil {
		t.Fatal(err)
	}
	if form.Get("user_email") != "a@x.com" || form.Get("unaddressed") != "false" || form.Get("status") != "" {
		t.Fatalf("form = %v", form)
	}
	if _, err := c.UpdateTicket("T1", UpdateTicketParams{Assign: "unassign"}); err != nil {
		t.Fatal(err)
	}
	if form.Get("user_id") != "unassign" {
		t.Fatalf("user_id = %q", form.Get("user_id"))
	}
}

func TestNormalizeStatus(t *testing.T) {
	for in, want := range map[string]string{"closed": "Closed", "OPEN": "Open", "pending": "Pending", "New": "New"} {
		if got, err := NormalizeStatus(in); err != nil || got != want {
			t.Errorf("NormalizeStatus(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := NormalizeStatus("archived"); err == nil {
		t.Error("expected error for archived")
	}
}
