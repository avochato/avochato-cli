package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/avochato/avochato-cli/internal/api"
	"github.com/avochato/avochato-cli/internal/output"
	"github.com/spf13/cobra"
)

var messagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "Send and list messages",
}

var messagesSendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send an outbound message to a phone number",
	RunE:  runMessagesSend,
}

var messagesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent messages",
	RunE:  runMessagesList,
}

func init() {
	messagesSendCmd.Flags().StringP("to", "t", "", "Recipient phone number in E.164 format (e.g. +16505551234)")
	messagesSendCmd.Flags().StringP("text", "m", "", "Message text to send")
	messagesSendCmd.Flags().String("from", "", "Sender phone number (must exist in your account)")
	messagesSendCmd.Flags().String("media-url", "", "MMS media attachment URL")
	messagesSendCmd.Flags().String("as", "", "Send on behalf of a user (email or user ID)")
	messagesSendCmd.Flags().Int("delay", 0, "Send after N seconds")
	_ = messagesSendCmd.MarkFlagRequired("to")

	messagesListCmd.Flags().IntP("limit", "l", 20, "Number of results")
	messagesListCmd.Flags().IntP("page", "p", 1, "Page number")
	messagesListCmd.Flags().Bool("all", false, "Fetch all pages")

	messagesCmd.AddCommand(messagesSendCmd, messagesListCmd)
	rootCmd.AddCommand(messagesCmd)
}

func runMessagesSend(cmd *cobra.Command, args []string) error {
	client, modePtr, err := writeClientFromCmd(cmd)
	if err != nil {
		output.PrintError(err)
		return nil
	}

	cfg := client.Config()
	fmt.Fprintf(os.Stderr, "Sending from inbox: %s\n", cfg.Account)

	to, _ := cmd.Flags().GetString("to")
	text, _ := cmd.Flags().GetString("text")
	from, _ := cmd.Flags().GetString("from")
	mediaURL, _ := cmd.Flags().GetString("media-url")
	as, _ := cmd.Flags().GetString("as")
	delay, _ := cmd.Flags().GetInt("delay")
	if text == "" && mediaURL == "" {
		output.PrintError(fmt.Errorf("pass --text, --media-url, or both"))
		return nil
	}

	params := api.SendMessageParams{
		To:           to,
		Text:         text,
		From:         from,
		MediaURL:     mediaURL,
		DelaySeconds: delay,
	}
	// --as accepts email or user ID
	if as != "" {
		if strings.Contains(as, "@") {
			params.SendAsEmail = as
		} else {
			params.SendAsUserID = as
		}
	}

	msg, err := client.SendMessage(params)
	if err != nil {
		output.PrintError(err)
		return nil
	}

	if *modePtr == output.ModeJSON {
		return output.PrintJSON(msg)
	}

	if delay > 0 {
		fmt.Printf("✓ Message scheduled (in %ds)\n", delay)
	} else {
		fmt.Printf("✓ Message sent\n")
	}
	fmt.Printf("  ID:        %s\n", msg.ID)
	fmt.Printf("  Inbox:     %s\n", cfg.Account)
	fmt.Printf("  To:        %s\n", msg.To)
	fmt.Printf("  From:      %s\n", msg.From)
	fmt.Printf("  Status:    %s\n", msg.Status)
	if msg.SentAt > 0 {
		fmt.Printf("  Sent at:   %s\n", time.Unix(int64(msg.SentAt), 0).UTC().Format("2006-01-02 15:04:05 UTC"))
	}
	return nil
}

func runMessagesList(cmd *cobra.Command, args []string) error {
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
	var messages []api.Message
	for {
		pageMsgs, err := client.ListMessages(page, limit)
		if err != nil {
			output.PrintError(err)
			return nil
		}
		messages = append(messages, pageMsgs...)
		if len(pageMsgs) == 0 || (!all && len(messages) >= limit) {
			break
		}
		page++
	}
	if !all && len(messages) > limit {
		messages = messages[:limit]
	}

	switch mode {
	case output.ModeJSON:
		return output.PrintJSON(messages)
	case output.ModeQuiet:
		for _, m := range messages {
			fmt.Println(m.ID)
		}
	default:
		t := output.NewTable([]string{"ID", "Direction", "From", "To", "Status", "Sent At"})
		for _, m := range messages {
			sentAt := ""
			if m.SentAt > 0 {
				sentAt = time.Unix(int64(m.SentAt), 0).UTC().Format("01/02 15:04")
			}
			t.Append([]string{m.ID, m.Direction, m.From, m.To, m.Status, sentAt})
		}
		t.Render()
	}
	return nil
}
