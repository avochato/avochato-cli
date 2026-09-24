package output

import (
	"strings"
	"testing"
)

func TestBoolStr(t *testing.T) {
	if BoolStr(true) != "yes" || BoolStr(false) != "no" {
		t.Fatal("BoolStr should render yes/no")
	}
}

func TestNewTableRendersToStdout(t *testing.T) {
	out, _ := capture(t, func() {
		tb := NewTable([]string{"ID", "Name"})
		tb.Append([]string{"C1", "Jane Doe"})
		tb.Render()
	})
	for _, want := range []string{"ID", "NAME", "C1", "Jane Doe"} {
		if !strings.Contains(out, want) {
			t.Errorf("table output missing %q:\n%s", want, out)
		}
	}
}
