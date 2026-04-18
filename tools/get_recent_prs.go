package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mekari/easysdlc/bitbucket"
)

// GetRecentPRsTool returns the MCP tool definition for get_recent_prs.
func GetRecentPRsTool() mcp.Tool {
	return mcp.NewTool("get_recent_prs",
		mcp.WithDescription("List recent open pull requests from a Bitbucket Cloud repository. Returns open PRs created within the specified time window."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("workspace",
			mcp.Description("Bitbucket workspace slug"),
			mcp.Required(),
		),
		mcp.WithString("repo_slug",
			mcp.Description("Repository slug"),
			mcp.Required(),
		),
		mcp.WithNumber("days",
			mcp.Description("Number of days to look back for recent PRs (default: 3)"),
		),
	)
}

// HandleGetRecentPRs returns a handler that lists recent open PRs.
func HandleGetRecentPRs(client *bitbucket.Client) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workspace, err := request.RequireString("workspace")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		repoSlug, err := request.RequireString("repo_slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		days := request.GetInt("days", 3)
		hours := days * 24

		prs, err := client.ListRecentPRs(ctx, workspace, repoSlug, hours)
		if err != nil {
			return mapError(err), nil
		}

		if len(prs) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("No open pull requests found in the last %d days.", days)), nil
		}

		data, err := json.MarshalIndent(prs, "", "  ")
		if err != nil {
			return mcp.NewToolResultError("Failed to format results: " + err.Error()), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}
