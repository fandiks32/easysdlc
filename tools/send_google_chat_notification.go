package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mekari/easysdlc/webhook"
)

// SendGoogleChatNotificationTool returns the MCP tool definition for send_google_chat_notification.
func SendGoogleChatNotificationTool() mcp.Tool {
	return mcp.NewTool("send_google_chat_notification",
		mcp.WithDescription("Send a code review request notification to Google Chat. Posts a formatted message with PR link, Jira tickets, and overview to the configured webhook."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("pr_url",
			mcp.Description("Pull request URL"),
			mcp.Required(),
		),
		mcp.WithString("jira_tickets",
			mcp.Description("Comma-separated Jira ticket keys (e.g. PROJ-456, PROJ-457)"),
			mcp.Required(),
		),
		mcp.WithString("title",
			mcp.Description("PR or task title"),
			mcp.Required(),
		),
		mcp.WithString("overview",
			mcp.Description("High-level summary of the changes for reviewers"),
			mcp.Required(),
		),
	)
}

// HandleSendGoogleChatNotification returns a handler that posts a review request to Google Chat.
func HandleSendGoogleChatNotification(client *webhook.Client) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		prURL, err := request.RequireString("pr_url")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		jiraTickets, err := request.RequireString("jira_tickets")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		title, err := request.RequireString("title")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		overview, err := request.RequireString("overview")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		message := fmt.Sprintf(`🔍 *Code Review Request*

*Title:* %s
*PR:* %s
*Jira:* %s

*Overview:*
%s

Please review when available.`, title, prURL, jiraTickets, overview)

		if err := client.Send(ctx, message); err != nil {
			return mcp.NewToolResultError("Failed to send Google Chat notification: " + err.Error()), nil
		}

		return mcp.NewToolResultText("Google Chat notification sent successfully."), nil
	}
}
