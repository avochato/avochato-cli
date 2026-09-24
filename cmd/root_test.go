package cmd

import (
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	rootCmd.Version = "1.2.3"
	t.Cleanup(func() { rootCmd.Version = "" })
	res := run(t, "", "--version")
	if res.Failed || !strings.Contains(res.Stdout, "1.2.3") {
		t.Fatalf("--version: %+v", res)
	}
}

func TestNoArgsPrintsBannerAndHelp(t *testing.T) {
	home(t, nil)
	res := run(t, "")
	for _, want := range []string{"CLI", "Available Commands", "messages", "login"} {
		if !strings.Contains(res.Stdout, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestUnknownCommandFailsOnce(t *testing.T) {
	res := run(t, "", "nope")
	if !res.Failed || strings.Count(res.Stderr, "unknown command") != 1 {
		t.Fatalf("got %+v", res)
	}
	if strings.Contains(res.Stderr, "Usage:") {
		t.Fatal("runtime errors should not print usage")
	}
}
