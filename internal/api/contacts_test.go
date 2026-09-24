package api

import (
	"net/http"
	"testing"
)

func TestGetContactByPhoneRequiresExactMatch(t *testing.T) {
	c := testClient(t, respondWith(`{"status":200,"data":{"contacts":[
		{"id":"C1","phone":"+15550000001"},{"id":"C2","phone":"+16505551234"}]}}`))

	got, err := c.GetContact("+16505551234")
	if err != nil || got.ID != "C2" {
		t.Fatalf("GetContact = %+v, %v", got, err)
	}
	if _, err := c.GetContact("+15559999999"); err == nil {
		t.Fatal("expected not found instead of the first search result")
	}
}

func TestCreateContactParsesSingularEnvelope(t *testing.T) {
	c := testClient(t, respondWith(`{"status":200,"data":{"contact":{"id":"C7","phone":"+16505551234"}}}`))
	got, err := c.CreateContact(CreateContactParams{Phone: "+16505551234"})
	if err != nil || got.ID != "C7" {
		t.Fatalf("CreateContact = %+v, %v", got, err)
	}
}

func TestLooksLikePhone(t *testing.T) {
	for s, want := range map[string]bool{
		"+16505551234": true,
		"16505551234":  true,
		"1aB3xYz":      false,
		"rl6Bro9MNJ":   false,
		"+":            false,
		"":             false,
	} {
		if got := looksLikePhone(s); got != want {
			t.Errorf("looksLikePhone(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestListContactsQueryAndCursor(t *testing.T) {
	var q map[string]string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		q = map[string]string{}
		for k := range r.URL.Query() {
			q[k] = r.URL.Query().Get(k)
		}
		respondWith(`{"status":200,"data":{"contacts":[{"id":"C1"}],"size":1,"next_page":"123"}}`)(w, r)
	})

	page, err := c.ListContacts("", 50, "")
	if err != nil || page.NextPage != "123" || len(page.Contacts) != 1 {
		t.Fatalf("page = %+v, %v", page, err)
	}
	if q["query"] != "*" || q["limit"] != "50" || q["after_page"] != "" {
		t.Fatalf("first page query = %v", q)
	}
	if _, err := c.ListContacts("acme", 0, "123"); err != nil {
		t.Fatal(err)
	}
	if q["query"] != "acme" || q["after_page"] != "123" || q["limit"] != "" {
		t.Fatalf("second page query = %v", q)
	}
}

func TestGetContactByID(t *testing.T) {
	var path string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		respondWith(`{"status":200,"data":{"contacts":[{"id":"C1","name":"Jane"}]}}`)(w, r)
	})
	got, err := c.GetContact("C1")
	if err != nil || got.Name != "Jane" || path != "/v1/contacts/C1" {
		t.Fatalf("GetContact = %+v, %v (path %s)", got, err, path)
	}
}

func TestOptInOutPaths(t *testing.T) {
	var path string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		respondWith(`{"status":200,"data":{}}`)(w, r)
	})
	if err := c.OptInContact("C1"); err != nil || path != "/v1/contacts/C1/opt_in" {
		t.Fatalf("opt-in: %v %s", err, path)
	}
	if err := c.OptOutContact("C1"); err != nil || path != "/v1/contacts/C1/opt_out" {
		t.Fatalf("opt-out: %v %s", err, path)
	}
}

func TestPhoneDigits(t *testing.T) {
	for in, want := range map[string]string{"+1 (650) 555-1234": "16505551234", "6505551234": "16505551234", "+442071234567": "442071234567"} {
		if got := phoneDigits(in); got != want {
			t.Errorf("phoneDigits(%q) = %q, want %q", in, got, want)
		}
	}
}
