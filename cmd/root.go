package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/avochato/avochato-cli/internal/api"
	"github.com/avochato/avochato-cli/internal/output"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "avochato",
	Short: "Official CLI for the Avochato messaging platform",
	Long: `avochato is the official command-line interface for the Avochato
messaging platform. Manage users, contacts, send messages, handle tickets,
and automate workflows from your terminal, scripts, or AI agents.`,
	// Errors are printed once by Execute; usage is only shown for --help.
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		printBanner()
		_ = cmd.Help()
	},
}

func Execute(version string) {
	rootCmd.Version = version
	api.UserAgent = "avochato-cli/" + version
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	// Commands report errors via output.PrintError and return nil, so exit non-zero here.
	if output.Failed() {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("account", "a", "", "Inbox to use: a saved profile name or an account subdomain (overrides the default profile)")
	rootCmd.PersistentFlags().BoolP("json", "j", false, "Output raw JSON")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Output IDs only (one per line)")
}

// confirm asks the user to approve an action; without a terminal it refuses unless --force was passed.
func confirm(cmd *cobra.Command, prompt, action string) (bool, error) {
	if force, _ := cmd.Flags().GetBool("force"); force {
		return true, nil
	}
	if !output.IsTerminal() {
		return false, fmt.Errorf("refusing to %s without confirmation; pass --force", action)
	}
	fmt.Printf("%s [y/N] ", prompt)
	var answer string
	fmt.Scanln(&answer)
	return strings.ToLower(answer) == "y", nil
}
