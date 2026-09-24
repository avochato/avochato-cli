package output

import (
	"io"
	"os"
	"testing"
)

// capture returns what fn writes to stdout and stderr.
func capture(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()
	fn()
	wOut.Close()
	wErr.Close()
	o, _ := io.ReadAll(rOut)
	e, _ := io.ReadAll(rErr)
	return string(o), string(e)
}

func TestDetectMode(t *testing.T) {
	if DetectMode(true, true) != ModeJSON {
		t.Error("--json should win over --quiet")
	}
	if DetectMode(false, true) != ModeQuiet {
		t.Error("--quiet should give quiet mode")
	}
	// Under `go test`, stdout is not a terminal, so output defaults to JSON.
	var got Mode
	capture(t, func() { got = DetectMode(false, false) })
	if got != ModeJSON {
		t.Errorf("piped stdout mode = %v, want JSON", got)
	}
}

func TestPrintJSON(t *testing.T) {
	out, _ := capture(t, func() { _ = PrintJSON(map[string]int{"a": 1}) })
	if out != "{\n  \"a\": 1\n}\n" {
		t.Fatalf("PrintJSON = %q", out)
	}
}

func TestPrintDone(t *testing.T) {
	cases := map[Mode]string{
		ModeJSON:  "{\n  \"action\": \"opt-in\",\n  \"id\": \"C1\",\n  \"ok\": true\n}\n",
		ModeQuiet: "C1\n",
		ModeHuman: "Opted in C1\n",
	}
	for mode, want := range cases {
		out, _ := capture(t, func() { _ = PrintDone(mode, "opt-in", "C1", "Opted in C1") })
		if out != want {
			t.Errorf("mode %v: got %q, want %q", mode, out, want)
		}
	}
}

func TestPrintErrorSetsFailed(t *testing.T) {
	failed = false
	t.Cleanup(func() { failed = false })
	if Failed() {
		t.Fatal("Failed() true before any error")
	}
	out, errOut := capture(t, func() { PrintError(io.ErrUnexpectedEOF) })
	if out != "" || errOut != "Error: unexpected EOF\n" {
		t.Fatalf("stdout=%q stderr=%q", out, errOut)
	}
	if !Failed() {
		t.Fatal("Failed() should be true after PrintError")
	}
	ResetFailed()
	if Failed() {
		t.Fatal("ResetFailed() should clear the flag")
	}
}

func TestIsTerminalFalseForPipe(t *testing.T) {
	var got bool
	capture(t, func() { got = IsTerminal() })
	if got {
		t.Fatal("a pipe is not a terminal")
	}
}
