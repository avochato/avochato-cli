package cmd

import (
	"fmt"
	"time"

	"github.com/avochato/avochato-cli/internal/api"
	"github.com/avochato/avochato-cli/internal/output"
	"github.com/spf13/cobra"
)

var ticketsCmd = &cobra.Command{
	Use:   "tickets",
	Short: "Manage tickets (conversations)",
}

var ticketsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tickets",
	RunE:  runTicketsList,
}

var ticketsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a ticket's details",
	Args:  cobra.ExactArgs(1),
	RunE:  runTicketsShow,
}

var ticketsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a ticket (assign, status, addressed state)",
	Args:  cobra.ExactArgs(1),
	RunE:  runTicketsUpdate,
}

var ticketsCloseCmd = &cobra.Command{
	Use:   "close <id>",
	Short: "Close a ticket",
	Args:  cobra.ExactArgs(1),
	RunE:  runTicketsClose,
}

var ticketsOpenCmd = &cobra.Command{
	Use:   "open <id>",
	Short: "Reopen a closed ticket",
	Args:  cobra.ExactArgs(1),
	RunE:  runTicketsOpen,
}

var ticketsAssignCmd = &cobra.Command{
	Use:   "assign <ticket-id> <user-id-or-email>",
	Short: "Assign a ticket to a user (use 'unassign' to remove, 'autoassign' for auto)",
	Args:  cobra.ExactArgs(2),
	RunE:  runTicketsAssign,
}

func init() {
	ticketsListCmd.Flags().StringP("status", "s", "", "Filter by status: open or closed")
	ticketsListCmd.Flags().String("query", "", "Search query")
	ticketsListCmd.Flags().StringP("user", "u", "", "Filter by assigned user (email or user ID)")
	ticketsListCmd.Flags().String("order", "most_recent_activity", "Sort order: newest, oldest, most_recent_activity, oldest_activity, best_match")
	ticketsListCmd.Flags().IntP("limit", "l", 20, "Number of results")
	ticketsListCmd.Flags().String("after", "", "Cursor for next page (from previous last_key)")
	ticketsListCmd.Flags().Bool("unaddressed", false, "Only show unaddressed tickets")
	ticketsListCmd.Flags().String("channel", "", "Filter by channel: sms, whatsapp, live_chat")

	ticketsUpdateCmd.Flags().String("assign", "", "Assign to user (email, user ID, 'unassign', or 'autoassign')")
	ticketsUpdateCmd.Flags().String("status", "", "Set status: open or closed")
	ticketsUpdateCmd.Flags().String("unaddressed", "", "Mark as unaddressed: true or false")

	ticketsCmd.AddCommand(ticketsListCmd, ticketsShowCmd, ticketsUpdateCmd, ticketsCloseCmd, ticketsOpenCmd, ticketsAssignCmd)
	rootCmd.AddCommand(ticketsCmd)
}

func runTicketsList(cmd *cobra.Command, args []string) error {
	client, modePtr, err := clientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}
	mode := *modePtr

	status, _ := cmd.Flags().GetString("status")
	query, _ := cmd.Flags().GetString("query")
	user, _ := cmd.Flags().GetString("user")
	order, _ := cmd.Flags().GetString("order")
	limit, _ := cmd.Flags().GetInt("limit")
	after, _ := cmd.Flags().GetString("after")
	unaddressed, _ := cmd.Flags().GetBool("unaddressed")
	channel, _ := cmd.Flags().GetString("channel")

	unaddressedStr := ""
	if unaddressed {
		unaddressedStr = "true"
	}

	page, err := client.ListTickets(api.ListTicketsParams{
		Query:       query,
		Status:      status,
		UserID:      user,
		Order:       order,
		Limit:       limit,
		After:       after,
		Unaddressed: unaddressedStr,
		Channel:     channel,
	})
	if err != nil {
		output.PrintError(err)
		return nil
	}

	switch mode {
	case output.ModeJSON:
		return output.PrintJSON(page)
	case output.ModeQuiet:
		for _, t := range page.Tickets {
			fmt.Println(t.ID)
		}
	default:
		t := output.NewTable([]string{"ID", "Status", "Contact", "Assigned To", "Unaddressed", "Created"})
		for _, tk := range page.Tickets {
			createdAt := ""
			if tk.CreatedAt > 0 {
				createdAt = time.Unix(int64(tk.CreatedAt), 0).UTC().Format("01/02 15:04")
			}
			t.Append([]string{
				tk.ID,
				tk.Status,
				tk.Contact,
				tk.UserID,
				output.BoolStr(tk.Unaddressed),
				createdAt,
			})
		}
		t.Render()
		if page.TotalCount > 0 {
			fmt.Printf("\nShowing %d of %d total", len(page.Tickets), page.TotalCount)
		}
		if page.LastKey != "" && mode == output.ModeHuman {
			fmt.Printf("  •  next: --after %q\n", page.LastKey)
		} else {
			fmt.Println()
		}
	}
	return nil
}

func runTicketsShow(cmd *cobra.Command, args []string) error {
	client, modePtr, err := clientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}

	ticket, err := client.GetTicket(args[0])
	if err != nil {
		output.PrintError(err)
		return nil
	}

	if *modePtr == output.ModeJSON {
		return output.PrintJSON(ticket)
	}

	fmt.Printf("Ticket: %s\n\n", ticket.ID)
	t := output.NewTable([]string{"Field", "Value"})
	t.Append([]string{"ID", ticket.ID})
	t.Append([]string{"UUID", ticket.UUID})
	t.Append([]string{"Status", ticket.Status})
	t.Append([]string{"Contact", ticket.Contact})
	t.Append([]string{"Assigned To", ticket.UserID})
	t.Append([]string{"Unaddressed", output.BoolStr(ticket.Unaddressed)})
	if ticket.Channel != "" {
		t.Append([]string{"Channel", ticket.Channel})
	}
	if ticket.Origin != "" {
		t.Append([]string{"Origin", ticket.Origin})
	}
	if ticket.Summary != "" {
		t.Append([]string{"Summary", ticket.Summary})
	}
	if ticket.CreatedAt > 0 {
		t.Append([]string{"Created At", time.Unix(int64(ticket.CreatedAt), 0).UTC().Format("2006-01-02 15:04:05 UTC")})
	}
	t.Render()
	return nil
}

func runTicketsUpdate(cmd *cobra.Command, args []string) error {
	client, modePtr, err := writeClientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}

	assign, _ := cmd.Flags().GetString("assign")
	status, _ := cmd.Flags().GetString("status")
	unaddressed, _ := cmd.Flags().GetString("unaddressed")

	ticket, err := client.UpdateTicket(args[0], api.UpdateTicketParams{
		Assign:      assign,
		Status:      status,
		Unaddressed: unaddressed,
	})
	if err != nil {
		output.PrintError(err)
		return nil
	}

	if *modePtr == output.ModeJSON {
		return output.PrintJSON(ticket)
	}

	fmt.Printf("Updated ticket %s\nStatus: %s  Assigned: %s\n", ticket.ID, ticket.Status, ticket.UserID)
	return nil
}

func runTicketsClose(cmd *cobra.Command, args []string) error {
	client, modePtr, err := writeClientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}
	ticket, err := client.UpdateTicket(args[0], api.UpdateTicketParams{Status: "closed"})
	if err != nil {
		output.PrintError(err)
		return nil
	}
	if *modePtr == output.ModeJSON {
		return output.PrintJSON(ticket)
	}
	fmt.Printf("Closed ticket %s\n", ticket.ID)
	return nil
}

func runTicketsOpen(cmd *cobra.Command, args []string) error {
	client, modePtr, err := writeClientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}
	ticket, err := client.UpdateTicket(args[0], api.UpdateTicketParams{Status: "open"})
	if err != nil {
		output.PrintError(err)
		return nil
	}
	if *modePtr == output.ModeJSON {
		return output.PrintJSON(ticket)
	}
	fmt.Printf("Reopened ticket %s\n", ticket.ID)
	return nil
}

func runTicketsAssign(cmd *cobra.Command, args []string) error {
	client, modePtr, err := writeClientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}
	ticket, err := client.UpdateTicket(args[0], api.UpdateTicketParams{Assign: args[1]})
	if err != nil {
		output.PrintError(err)
		return nil
	}
	if *modePtr == output.ModeJSON {
		return output.PrintJSON(ticket)
	}
	fmt.Printf("Ticket %s assigned to %s\n", ticket.ID, ticket.UserID)
	return nil
}
