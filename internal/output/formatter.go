package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// Mode controls how command output is rendered.
type Mode int

const (
	ModeHuman Mode = iota // TTY tables (default)
	ModeJSON              // Raw JSON
	ModeQuiet             // IDs only, one per line
)

// DetectMode resolves the output mode from flags, defaulting to JSON when stdout is not a terminal.
func DetectMode(jsonFlag, quietFlag bool) Mode {
	if jsonFlag {
		return ModeJSON
	}
	if quietFlag {
		return ModeQuiet
	}
	if !IsTerminal() {
		return ModeJSON
	}
	return ModeHuman
}

// IsTerminal reports whether stdout is an interactive terminal.
func IsTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// PrintJSON marshals v as indented JSON and writes it to stdout.
func PrintJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// PrintDone reports a completed action: JSON gets {"ok":true,...}, quiet the ID, humans the message.
func PrintDone(mode Mode, action, id, message string) error {
	switch mode {
	case ModeJSON:
		return PrintJSON(map[string]any{"ok": true, "action": action, "id": id})
	case ModeQuiet:
		fmt.Println(id)
	default:
		fmt.Println(Clean(message))
	}
	return nil
}

// failed records whether any error was printed, so the process can exit non-zero.
var failed bool

// PrintError writes an error message to stderr and marks the run as failed.
func PrintError(err error) {
	failed = true
	fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
}

// Failed reports whether PrintError was called during this run.
func Failed() bool {
	return failed
}

// ResetFailed clears the error flag; tests that run several commands in one process call it between runs.
func ResetFailed() {
	failed = false
}
