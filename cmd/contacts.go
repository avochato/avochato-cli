package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/avochato/avochato-cli/internal/api"
	"github.com/avochato/avochato-cli/internal/output"
	"github.com/spf13/cobra"
)

var contactsCmd = &cobra.Command{
	Use:   "contacts",
	Short: "Manage contacts",
}

var contactsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List contacts in the account",
	RunE:  runContactsList,
}

var contactsShowCmd = &cobra.Command{
	Use:   "show <id-or-phone>",
	Short: "Show a contact's details",
	Args:  cobra.ExactArgs(1),
	RunE:  runContactsShow,
}

var contactsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create or update a contact",
	RunE:  runContactsCreate,
}

var contactsOptOutCmd = &cobra.Command{
	Use:   "opt-out <id>",
	Short: "Opt a contact out of receiving messages",
	Args:  cobra.ExactArgs(1),
	RunE:  runContactsOptOut,
}

var contactsOptInCmd = &cobra.Command{
	Use:   "opt-in <id>",
	Short: "Opt a contact back in to receive messages",
	Args:  cobra.ExactArgs(1),
	RunE:  runContactsOptIn,
}

func init() {
	contactsListCmd.Flags().StringP("query", "s", "", "Search query (default: all contacts)")
	contactsListCmd.Flags().IntP("limit", "l", 50, "Number of results per page (max 100)")
	contactsListCmd.Flags().String("after", "", "Cursor for next page (from previous next_page)")
	contactsListCmd.Flags().Bool("all", false, "Fetch all pages automatically")

	contactsCreateCmd.Flags().StringP("phone", "t", "", "Phone number in E.164 format (required)")
	contactsCreateCmd.Flags().StringP("name", "n", "", "Contact name")
	contactsCreateCmd.Flags().StringP("email", "e", "", "Email address")
	contactsCreateCmd.Flags().StringP("company", "c", "", "Company name")
	contactsCreateCmd.Flags().String("notes", "", "Notes")
	_ = contactsCreateCmd.MarkFlagRequired("phone")

	contactsOptInCmd.Flags().Bool("force", false, "Skip confirmation prompt")

	contactsCmd.AddCommand(contactsListCmd, contactsShowCmd, contactsCreateCmd, contactsOptOutCmd, contactsOptInCmd)
	rootCmd.AddCommand(contactsCmd)
}

func runContactsList(cmd *cobra.Command, args []string) error {
	client, modePtr, err := clientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}
	mode := *modePtr

	query, _ := cmd.Flags().GetString("query")
	limit, _ := cmd.Flags().GetInt("limit")
	after, _ := cmd.Flags().GetString("after")
	all, _ := cmd.Flags().GetBool("all")

	var allContacts []api.Contact
	var nextCursor string
	cursor := after
	for {
		page, err := client.ListContacts(query, limit, cursor)
		if err != nil {
			output.PrintError(err)
			return nil
		}
		allContacts = append(allContacts, page.Contacts...)

		if !all || page.NextPage == "" {
			nextCursor = page.NextPage
			break
		}
		cursor = page.NextPage
	}

	switch mode {
	case output.ModeJSON:
		return output.PrintJSON(allContacts)
	case output.ModeQuiet:
		for _, c := range allContacts {
			fmt.Println(c.ID)
		}
	default:
		t := output.NewTable([]string{"ID", "Name", "Phone", "Email", "Company", "Opted Out"})
		for _, c := range allContacts {
			t.Append([]string{c.ID, c.Name, c.Phone, c.Email, c.Company, output.BoolStr(c.OptedOut)})
		}
		t.Render()
		// In single-page mode, surface the cursor so the user can paginate manually
		if !all && nextCursor != "" {
			fmt.Printf("\n(next page: --after %q)\n", nextCursor)
		}
	}
	return nil
}

func runContactsShow(cmd *cobra.Command, args []string) error {
	client, modePtr, err := clientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}

	contact, err := client.GetContact(args[0])
	if err != nil {
		output.PrintError(err)
		return nil
	}

	if *modePtr == output.ModeJSON {
		return output.PrintJSON(contact)
	}

	fmt.Printf("Contact: %s\n\n", contact.Phone)
	t := output.NewTable([]string{"Field", "Value"})
	t.Append([]string{"ID", contact.ID})
	t.Append([]string{"Name", contact.Name})
	t.Append([]string{"Phone", contact.Phone})
	if contact.Email != "" {
		t.Append([]string{"Email", contact.Email})
	}
	if contact.Company != "" {
		t.Append([]string{"Company", contact.Company})
	}
	if contact.Notes != "" {
		t.Append([]string{"Notes", contact.Notes})
	}
	t.Append([]string{"Opted Out", output.BoolStr(contact.OptedOut)})
	t.Append([]string{"Muted", output.BoolStr(contact.Muted)})
	t.Append([]string{"Blocked", output.BoolStr(contact.Blocked)})
	if len(contact.Tags) > 0 {
		t.Append([]string{"Tags", strings.Join(contact.Tags, ", ")})
	}
	if contact.CreatedAt > 0 {
		t.Append([]string{"Created At", time.Unix(int64(contact.CreatedAt), 0).UTC().Format("2006-01-02 15:04:05 UTC")})
	}
	t.Render()
	return nil
}

func runContactsCreate(cmd *cobra.Command, args []string) error {
	client, modePtr, err := writeClientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}

	phone, _ := cmd.Flags().GetString("phone")
	name, _ := cmd.Flags().GetString("name")
	email, _ := cmd.Flags().GetString("email")
	company, _ := cmd.Flags().GetString("company")
	notes, _ := cmd.Flags().GetString("notes")

	contact, err := client.CreateContact(api.CreateContactParams{
		Phone:   phone,
		Name:    name,
		Email:   email,
		Company: company,
		Notes:   notes,
	})
	if err != nil {
		output.PrintError(err)
		return nil
	}

	if *modePtr == output.ModeJSON {
		return output.PrintJSON(contact)
	}

	fmt.Printf("Contact saved\nID:    %s\nPhone: %s\nName:  %s\n", contact.ID, contact.Phone, contact.Name)
	return nil
}

func runContactsOptOut(cmd *cobra.Command, args []string) error {
	client, modePtr, err := writeClientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}
	if err := client.OptOutContact(args[0]); err != nil {
		output.PrintError(err)
		return nil
	}
	return output.PrintDone(*modePtr, "opt-out", args[0], "Opted out "+args[0])
}

func runContactsOptIn(cmd *cobra.Command, args []string) error {
	client, modePtr, err := writeClientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}
	// Opt-in is a compliance action, so it needs the same confirmation as removing a user.
	ok, err := confirm(cmd, fmt.Sprintf("Opt %s back in? Only do this if the contact asked to receive messages again.", args[0]), "opt in "+args[0])
	if err != nil {
		output.PrintError(err)
		return nil
	}
	if !ok {
		fmt.Println("Cancelled.")
		return nil
	}
	if err := client.OptInContact(args[0]); err != nil {
		output.PrintError(err)
		return nil
	}
	return output.PrintDone(*modePtr, "opt-in", args[0], "Opted in "+args[0])
}
