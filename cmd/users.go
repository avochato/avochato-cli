package cmd

import (
	"fmt"
	"time"

	"github.com/avochato/avochato-cli/internal/api"
	"github.com/avochato/avochato-cli/internal/config"
	"github.com/avochato/avochato-cli/internal/output"
	"github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage account users",
}

var usersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users in the account",
	RunE:  runUsersList,
}

var usersShowCmd = &cobra.Command{
	Use:   "show <id-or-email>",
	Short: "Show a user's details",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersShow,
}

var usersInviteCmd = &cobra.Command{
	Use:   "invite <email>",
	Short: "Invite a new user to the account",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersInvite,
}

var usersUpdateCmd = &cobra.Command{
	Use:   "update <id-or-email>",
	Short: "Update a user's role",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersUpdate,
}

var usersRemoveCmd = &cobra.Command{
	Use:   "remove <id-or-email>",
	Short: "Remove a user from the account",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersRemove,
}

var usersEnableCmd = &cobra.Command{
	Use:   "enable <id-or-email>",
	Short: "Enable a disabled user",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersEnable,
}

var usersDisableCmd = &cobra.Command{
	Use:   "disable <id-or-email>",
	Short: "Disable a user without removing them",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersDisable,
}

var usersResetPasswordCmd = &cobra.Command{
	Use:   "reset-password <id-or-email>",
	Short: "Send a password reset email",
	Args:  cobra.ExactArgs(1),
	RunE:  runUsersResetPassword,
}

var validRoles = map[string]bool{"member": true, "manager": true, "owner": true}

func init() {
	usersListCmd.Flags().IntP("limit", "l", 30, "Number of results")
	usersListCmd.Flags().IntP("page", "p", 1, "Page number")
	usersListCmd.Flags().Bool("all", false, "Fetch all pages")

	usersInviteCmd.Flags().StringP("role", "r", "member", "Role: member, manager, or owner")
	usersInviteCmd.Flags().String("name", "", "Display name")
	usersInviteCmd.Flags().String("phone", "", "Phone number in E.164 format")

	usersUpdateCmd.Flags().StringP("role", "r", "", "New role: member, manager, or owner")
	_ = usersUpdateCmd.MarkFlagRequired("role")

	usersRemoveCmd.Flags().Bool("force", false, "Skip confirmation prompt")

	usersCmd.AddCommand(
		usersListCmd, usersShowCmd, usersInviteCmd,
		usersUpdateCmd, usersRemoveCmd,
		usersEnableCmd, usersDisableCmd, usersResetPasswordCmd,
	)
	rootCmd.AddCommand(usersCmd)
}

// clientFromCmd builds an API client from the root persistent flags.
func clientFromCmd(cmd *cobra.Command) (*api.Client, *output.Mode, error) {
	account, _ := cmd.Root().PersistentFlags().GetString("account")
	cfg, err := config.Resolve(account, "")
	if err != nil {
		return nil, nil, err
	}
	if err := config.ValidateBaseURL(cfg.BaseURL); err != nil {
		return nil, nil, err
	}
	if cfg.AuthID == "" {
		if cfg.ProfileCount > 0 {
			return nil, nil, fmt.Errorf("no default profile is set; pass --account <inbox> or run `avochato auth use <profile>`")
		}
		return nil, nil, fmt.Errorf("not authenticated; run `avochato login` first")
	}
	jsonFlag, _ := cmd.Root().PersistentFlags().GetBool("json")
	quietFlag, _ := cmd.Root().PersistentFlags().GetBool("quiet")
	mode := output.DetectMode(jsonFlag, quietFlag)
	return api.NewClient(cfg), &mode, nil
}

// writeClientFromCmd is clientFromCmd for commands that change data; it requires a named inbox when several are saved.
func writeClientFromCmd(cmd *cobra.Command) (*api.Client, *output.Mode, error) {
	client, mode, err := clientFromCmd(cmd)
	if err != nil {
		return nil, nil, err
	}
	if err := client.Config().RequireExplicitInbox(); err != nil {
		return nil, nil, err
	}
	return client, mode, nil
}

func runUsersList(cmd *cobra.Command, args []string) error {
	client, modePtr, err := clientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}
	mode := *modePtr

	limit, _ := cmd.Flags().GetInt("limit")
	page, _ := cmd.Flags().GetInt("page")
	all, _ := cmd.Flags().GetBool("all")

	// The API ignores limit and pages 25 at a time, so page until enough and trim client-side.
	var users []api.User
	for {
		pageUsers, err := client.ListUsers(page, limit)
		if err != nil {
			output.PrintError(err)
			return nil
		}
		users = append(users, pageUsers...)
		if len(pageUsers) == 0 || (!all && len(users) >= limit) {
			break
		}
		page++
	}
	if !all && len(users) > limit {
		users = users[:limit]
	}

	switch mode {
	case output.ModeJSON:
		return output.PrintJSON(users)
	case output.ModeQuiet:
		for _, u := range users {
			fmt.Println(u.ID)
		}
	default:
		// The list endpoint does not return role or enabled; see `users show`.
		t := output.NewTable([]string{"ID", "Email", "Name"})
		for _, u := range users {
			t.Append([]string{u.ID, u.Email, u.Name})
		}
		t.Render()
	}
	return nil
}

func runUsersShow(cmd *cobra.Command, args []string) error {
	client, modePtr, err := clientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}

	user, err := client.GetUser(args[0])
	if err != nil {
		output.PrintError(err)
		return nil
	}

	if *modePtr == output.ModeJSON {
		return output.PrintJSON(user)
	}

	fmt.Printf("User: %s\n\n", user.Email)
	t := output.NewTable([]string{"Field", "Value"})
	t.Append([]string{"ID", user.ID})
	t.Append([]string{"Email", user.Email})
	t.Append([]string{"Name", user.Name})
	t.Append([]string{"Role", user.Role})
	t.Append([]string{"Enabled", output.BoolStr(user.Enabled)})
	t.Append([]string{"Can Message", output.BoolStr(user.CanMessage)})
	t.Append([]string{"Can Call", output.BoolStr(user.CanCall)})
	t.Append([]string{"2FA Required", output.BoolStr(user.Require2FA)})
	if user.PhoneFormatted != "" {
		t.Append([]string{"Phone", user.PhoneFormatted})
	}
	if user.AddedAt > 0 {
		t.Append([]string{"Added At", time.Unix(int64(user.AddedAt), 0).UTC().Format("2006-01-02 15:04:05 UTC")})
	}
	if user.CreatedAt > 0 {
		t.Append([]string{"Created At", time.Unix(int64(user.CreatedAt), 0).UTC().Format("2006-01-02 15:04:05 UTC")})
	}
	t.Render()
	return nil
}

func runUsersInvite(cmd *cobra.Command, args []string) error {
	client, modePtr, err := writeClientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}

	role, _ := cmd.Flags().GetString("role")
	name, _ := cmd.Flags().GetString("name")
	phone, _ := cmd.Flags().GetString("phone")

	if !validRoles[role] {
		output.PrintError(fmt.Errorf("invalid role %q; must be one of: member, manager, owner", role))
		return nil
	}

	user, err := client.InviteUser(args[0], role, name, phone)
	if err != nil {
		output.PrintError(err)
		return nil
	}

	if *modePtr == output.ModeJSON {
		return output.PrintJSON(user)
	}

	fmt.Printf("Invited %s\nRole:    %s\nUser ID: %s\n", user.Email, user.Role, user.ID)
	return nil
}

func runUsersUpdate(cmd *cobra.Command, args []string) error {
	client, modePtr, err := writeClientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}

	role, _ := cmd.Flags().GetString("role")
	if !validRoles[role] {
		output.PrintError(fmt.Errorf("invalid role %q; must be one of: member, manager, owner", role))
		return nil
	}

	user, err := client.UpdateUser(args[0], role)
	if err != nil {
		output.PrintError(err)
		return nil
	}

	if *modePtr == output.ModeJSON {
		return output.PrintJSON(user)
	}

	fmt.Printf("Updated %s\nRole changed to: %s\n", user.Email, user.Role)
	return nil
}

func runUsersRemove(cmd *cobra.Command, args []string) error {
	client, modePtr, err := writeClientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}

	idOrEmail := args[0]
	ok, err := confirm(cmd, fmt.Sprintf("Remove %s from this account?", idOrEmail), "remove "+idOrEmail)
	if err != nil {
		output.PrintError(err)
		return nil
	}
	if !ok {
		fmt.Println("Cancelled.")
		return nil
	}

	if err := client.RemoveUser(idOrEmail); err != nil {
		output.PrintError(err)
		return nil
	}

	return output.PrintDone(*modePtr, "remove", idOrEmail, "Removed "+idOrEmail)
}

func runUsersEnable(cmd *cobra.Command, args []string) error {
	client, modePtr, err := writeClientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}
	if err := client.EnableUser(args[0]); err != nil {
		output.PrintError(err)
		return nil
	}
	return output.PrintDone(*modePtr, "enable", args[0], "Enabled "+args[0])
}

func runUsersDisable(cmd *cobra.Command, args []string) error {
	client, modePtr, err := writeClientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}
	if err := client.DisableUser(args[0]); err != nil {
		output.PrintError(err)
		return nil
	}
	return output.PrintDone(*modePtr, "disable", args[0], "Disabled "+args[0])
}

func runUsersResetPassword(cmd *cobra.Command, args []string) error {
	client, modePtr, err := writeClientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}
	if err := client.ResetPassword(args[0]); err != nil {
		output.PrintError(err)
		return nil
	}
	return output.PrintDone(*modePtr, "reset-password", args[0], "Password reset email sent to "+args[0])
}
