package cmd

import (
	"strings"
	"testing"
)

func TestColorsDisabledWhenNotATerminal(t *testing.T) {
	var got bool
	res := captureBanner(t, &got)
	if got {
		t.Fatal("colors should be off when stdout is a pipe")
	}
	if strings.Contains(res, "\033[") {
		t.Fatal("banner printed ANSI codes with colors off")
	}
	if !strings.Contains(res, "CLI") {
		t.Fatal("banner missing the CLI label")
	}
}

func TestColorsRespectEnv(t *testing.T) {
	for _, env := range [][2]string{{"NO_COLOR", "1"}, {"TERM", "dumb"}} {
		t.Setenv(env[0], env[1])
		if colorsEnabled() {
			t.Errorf("%s=%s should disable colors", env[0], env[1])
		}
	}
}

// captureBanner prints the banner with stdout redirected to a pipe.
func captureBanner(t *testing.T, enabled *bool) string {
	t.Helper()
	res := runFunc(t, func() {
		*enabled = colorsEnabled()
		printBanner()
	})
	return res
}
