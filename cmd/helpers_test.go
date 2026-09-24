package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/avochato/avochato-cli/internal/config"
	"github.com/avochato/avochato-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// request is one call the fake API received.
type request struct {
	Method string
	Path   string
	Form   url.Values // query and form body merged
}

// route answers one "METHOD /v1/path" with a status and a JSON "data" body.
type route func(r *http.Request) (status int, data any)

// fakeAPI is an in-process stand-in for the Avochato V1 API.
type fakeAPI struct {
	URL    string
	mu     sync.Mutex
	routes map[string]route
	reqs   []request
}

func newFakeAPI(t *testing.T) *fakeAPI {
	t.Helper()
	f := &fakeAPI{routes: map[string]route{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		f.mu.Lock()
		f.reqs = append(f.reqs, request{Method: r.Method, Path: r.URL.Path, Form: r.Form})
		h, ok := f.routes[r.Method+" "+r.URL.Path]
		f.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"status":404,"errors":["no route"]}`)
			return
		}
		status, data := h(r)
		w.WriteHeader(status)
		body := map[string]any{"status": status}
		if status >= 400 {
			body["errors"] = data
		} else {
			body["data"] = data
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)
	f.URL = srv.URL
	return f
}

// on registers a handler for "METHOD /v1/path".
func (f *fakeAPI) on(methodPath string, h route) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.routes[methodPath] = h
}

// ok registers a fixed 200 response.
func (f *fakeAPI) ok(methodPath string, data any) {
	f.on(methodPath, func(*http.Request) (int, any) { return 200, data })
}

// requests returns the calls received so far.
func (f *fakeAPI) requests() []request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]request(nil), f.reqs...)
}

// last returns the most recent call, failing if there was none.
func (f *fakeAPI) last(t *testing.T) request {
	t.Helper()
	reqs := f.requests()
	if len(reqs) == 0 {
		t.Fatal("no request reached the API")
	}
	return reqs[len(reqs)-1]
}

// home points HOME at a temp dir with the given saved profiles (nil for none)
// and clears the AVOCHATO_* environment.
func home(t *testing.T, creds *config.Credentials) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	for _, k := range []string{"AVOCHATO_PROFILE", "AVOCHATO_AUTH_ID", "AVOCHATO_AUTH_SECRET", "AVOCHATO_ACCOUNT", "AVOCHATO_BASE_URL", "AVOCHATO_DEBUG"} {
		t.Setenv(k, "")
	}
	if creds != nil {
		if err := config.SaveCredentials(creds); err != nil {
			t.Fatal(err)
		}
	}
}

// oneInbox saves a single "acme" profile pointing at the fake API.
func oneInbox(t *testing.T, f *fakeAPI) {
	home(t, &config.Credentials{
		DefaultAccount: "acme",
		Accounts:       map[string]config.Profile{"acme": {AuthID: "id1", AuthSecret: "sec1", BaseURL: f.URL, Subdomain: "acme"}},
	})
}

// twoInboxes saves "acme" (default) and "support" profiles.
func twoInboxes(t *testing.T, f *fakeAPI) {
	home(t, &config.Credentials{
		DefaultAccount: "acme",
		Accounts: map[string]config.Profile{
			"acme":    {AuthID: "id1", AuthSecret: "sec1", BaseURL: f.URL, Subdomain: "acme"},
			"support": {AuthID: "id2", AuthSecret: "sec2", BaseURL: f.URL, Subdomain: "support"},
		},
	})
}

// result is the outcome of one CLI invocation.
type result struct {
	Stdout, Stderr string
	Failed         bool // the process would exit non-zero
}

// run executes the CLI in-process with args and the given stdin, the way
// Execute does, but without exiting.
func run(t *testing.T, stdin string, args ...string) result {
	t.Helper()
	resetFlags(rootCmd)
	output.ResetFailed()

	oldIn, oldOut, oldErr := os.Stdin, os.Stdout, os.Stderr
	inR, inW, _ := os.Pipe()
	outR, outW, _ := os.Pipe()
	errR, errW, _ := os.Pipe()
	os.Stdin, os.Stdout, os.Stderr = inR, outW, errW
	go func() { _, _ = io.WriteString(inW, stdin); inW.Close() }()

	var outBuf, errBuf strings.Builder
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(&outBuf, outR) }()
	go func() { defer wg.Done(); _, _ = io.Copy(&errBuf, errR) }()

	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	if err != nil {
		_, _ = io.WriteString(errW, "Error: "+err.Error()+"\n")
	}

	outW.Close()
	errW.Close()
	wg.Wait()
	os.Stdin, os.Stdout, os.Stderr = oldIn, oldOut, oldErr
	return result{Stdout: outBuf.String(), Stderr: errBuf.String(), Failed: err != nil || output.Failed()}
}

// resetFlags restores every flag to its default, since cobra keeps flag
// values between Execute calls in one process.
func resetFlags(c *cobra.Command) {
	reset := func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	}
	c.Flags().VisitAll(reset)
	c.PersistentFlags().VisitAll(reset)
	for _, sub := range c.Commands() {
		resetFlags(sub)
	}
}

// decode parses JSON stdout into v.
func decode(t *testing.T, res result, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(res.Stdout), v); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, res.Stdout)
	}
}

// runFunc captures what fn writes to stdout.
func runFunc(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	var buf strings.Builder
	done := make(chan struct{})
	go func() { _, _ = io.Copy(&buf, r); close(done) }()
	fn()
	w.Close()
	<-done
	os.Stdout = old
	return buf.String()
}
