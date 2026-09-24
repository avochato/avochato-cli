package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/avochato/avochato-cli/internal/api"
	"github.com/avochato/avochato-cli/internal/config"
	"github.com/avochato/avochato-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication profiles",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Save credentials for an account",
	RunE:  runAuthLogin,
}

// loginCmd is the top-level shortcut for `auth login`.
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Save credentials for an account (shortcut for `auth login`)",
	RunE:  runAuthLogin,
}

var authWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the currently authenticated user",
	RunE:  runAuthWhoami,
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved profiles",
	RunE:  runAuthList,
}

var authUseCmd = &cobra.Command{
	Use:   "use <profile>",
	Short: "Set the default profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runAuthUse,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove saved credentials",
	RunE:  runAuthLogout,
}

func init() {
	authLoginCmd.Flags().String("profile", "", "Save under a named profile (default: account subdomain)")
	authLoginCmd.Flags().String("base-url", "", "Override base URL (e.g. https://staging.avochato.com)")
	authLoginCmd.Flags().Bool("default", false, "Make this profile the default")
	loginCmd.Flags().AddFlagSet(authLoginCmd.Flags())

	authLogoutCmd.Flags().String("profile", "", "Profile to remove (default: current default)")
	authLogoutCmd.Flags().Bool("all", false, "Remove all profiles")

	authCmd.AddCommand(authLoginCmd, authWhoamiCmd, authListCmd, authUseCmd, authLogoutCmd)
	rootCmd.AddCommand(authCmd, loginCmd)
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Auth ID:     ")
	authID, _ := reader.ReadString('\n')
	authID = strings.TrimSpace(authID)

	fmt.Print("Auth Secret: ")
	authSecret := readSecret(reader)

	fmt.Print("Account (subdomain): ")
	account, _ := reader.ReadString('\n')
	account = strings.TrimSpace(account)

	baseURL, _ := cmd.Flags().GetString("base-url")
	if baseURL == "" {
		baseURL = "https://www.avochato.com"
	}
	if err := config.ValidateBaseURL(baseURL); err != nil {
		return err
	}

	cfg := &config.Config{
		AuthID:     authID,
		AuthSecret: authSecret,
		Account:    account,
		BaseURL:    baseURL,
	}

	fmt.Print("Verifying credentials... ")
	client := api.NewClient(cfg)
	who, err := client.Whoami()
	if err != nil {
		fmt.Println("failed")
		output.PrintError(err)
		return nil
	}
	// /v1/auth_tokens does not check the subdomain, so make one call that does.
	if _, err := client.ListUsers(1, 1); err != nil {
		fmt.Println("failed")
		output.PrintError(fmt.Errorf("credentials are valid but cannot access inbox %q: %w", account, err))
		return nil
	}
	fmt.Println("OK")

	profileName, _ := cmd.Flags().GetString("profile")
	if profileName == "" {
		profileName = account
	}

	creds, err := config.LoadCredentials()
	if err != nil {
		// Never overwrite a file we could not parse; it holds the other profiles.
		return fmt.Errorf("could not read ~/.avochato/credentials.json (fix or remove it first): %w", err)
	}
	if creds == nil {
		creds = &config.Credentials{Accounts: make(map[string]config.Profile)}
	}
	creds.SetProfile(profileName, config.Profile{
		AuthID:     authID,
		AuthSecret: authSecret,
		BaseURL:    baseURL,
		Subdomain:  account,
	})
	if makeDefault, _ := cmd.Flags().GetBool("default"); makeDefault {
		creds.DefaultAccount = profileName
	}

	if err := config.SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Printf("Logged in as %s (%s)\n", output.Clean(who.Name), account)
	fmt.Printf("Profile %q saved to ~/.avochato/credentials.json\n", profileName)
	if creds.DefaultAccount != profileName {
		fmt.Printf("\nThe default profile is still %q. Commands without --account use that inbox.\n", creds.DefaultAccount)
		fmt.Printf("Run `avochato auth use %s` to make this the default.\n", profileName)
	}
	return nil
}

// readSecret reads a line without echoing it when stdin is a terminal.
func readSecret(reader *bufio.Reader) string {
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Println()
		if err == nil {
			return strings.TrimSpace(string(b))
		}
	}
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func runAuthWhoami(cmd *cobra.Command, args []string) error {
	jsonFlag, _ := rootCmd.PersistentFlags().GetBool("json")

	// clientFromCmd validates the base URL, so the secret is never sent over plain http.
	client, _, err := clientFromCmd(cmd)
	if err != nil {
		return err
	}
	cfg := client.Config()
	who, err := client.Whoami()
	if err != nil {
		output.PrintError(err)
		return nil
	}

	if output.DetectMode(jsonFlag, false) == output.ModeJSON {
		return output.PrintJSON(who)
	}

	fmt.Printf("Logged in as %s\n", output.Clean(who.Name))
	fmt.Printf("Email:   %s\n", output.Clean(who.Email))
	fmt.Printf("Account: %s\n", cfg.Account)
	if cfg.Profile != "" {
		fmt.Printf("Profile: %s\n", cfg.Profile)
	}
	return nil
}

func runAuthList(cmd *cobra.Command, args []string) error {
	creds, err := config.LoadCredentials()
	if err != nil || creds == nil {
		fmt.Println("No profiles saved. Run `avochato login` to get started.")
		return nil
	}

	def := creds.DefaultAccount
	if def == "" {
		def = "(none, pass --account or run `avochato auth use <profile>`)"
	}
	fmt.Printf("Default: %s\n\n", def)
	for _, name := range creds.Names() {
		marker := "  "
		if name == creds.DefaultAccount {
			marker = "* "
		}
		line := marker + name
		if sub := creds.Accounts[name].Subdomain; sub != "" && sub != name {
			line += " (" + sub + ")"
		}
		fmt.Println(line)
	}
	return nil
}

func runAuthUse(cmd *cobra.Command, args []string) error {
	profile := args[0]
	creds, err := config.LoadCredentials()
	if err != nil || creds == nil {
		return fmt.Errorf("no credentials found; run `avochato login` first")
	}
	name, ok := creds.FindProfile(profile)
	if !ok {
		return fmt.Errorf("profile %q not found", profile)
	}
	profile = name
	creds.DefaultAccount = profile
	if err := config.SaveCredentials(creds); err != nil {
		return err
	}
	fmt.Printf("Default profile set to %q\n", profile)
	return nil
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	all, _ := cmd.Flags().GetBool("all")

	creds, err := config.LoadCredentials()
	if err != nil || creds == nil {
		fmt.Println("No credentials to remove.")
		return nil
	}

	if all {
		if err := config.SaveCredentials(&config.Credentials{
			Accounts: make(map[string]config.Profile),
		}); err != nil {
			return err
		}
		fmt.Println("All profiles removed.")
		return nil
	}

	profile, _ := cmd.Flags().GetString("profile")
	if profile == "" {
		profile = creds.DefaultAccount
	}

	if _, ok := creds.Accounts[profile]; !ok {
		return fmt.Errorf("profile %q not found", profile)
	}
	creds.RemoveProfile(profile)

	if err := config.SaveCredentials(creds); err != nil {
		return fmt.Errorf("failed to update credentials: %w", err)
	}

	fmt.Printf("Logged out of %q\n", profile)
	if creds.DefaultAccount == "" && len(creds.Accounts) > 1 {
		fmt.Println("No default profile is set. Run `avochato auth use <profile>` or pass --account.")
	}
	return nil
}
