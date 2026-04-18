package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mekari/easysdlc/bitbucket"
)

// ReviewOpenPRsTool returns the MCP tool definition for review_open_prs.
func ReviewOpenPRsTool() mcp.Tool {
	return mcp.NewTool("review_open_prs",
		mcp.WithDescription("Fetch all open pull requests from the last N days along with their full diffs, ready for code review. Returns each PR's metadata and diff in a single response."),
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
			mcp.Description("Number of days to look back (default: 3)"),
		),
	)
}

// HandleReviewOpenPRs returns a handler that fetches recent PRs with their diffs.
func HandleReviewOpenPRs(client *bitbucket.Client) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		var report strings.Builder
		report.WriteString(fmt.Sprintf("# Open Pull Requests — Last %d Days\n\n", days))
		report.WriteString(fmt.Sprintf("Found **%d** open PR(s) in `%s/%s`.\n\n", len(prs), workspace, repoSlug))

		for i, pr := range prs {
			report.WriteString(fmt.Sprintf("---\n\n## PR #%d: %s\n\n", pr.ID, pr.Title))
			report.WriteString(fmt.Sprintf("- **Author:** %s\n", pr.Author.DisplayName))
			report.WriteString(fmt.Sprintf("- **Branch:** %s → %s\n", pr.Source.Branch.Name, pr.Destination.Branch.Name))
			report.WriteString(fmt.Sprintf("- **Created:** %s\n", pr.CreatedOn))
			report.WriteString(fmt.Sprintf("- **URL:** %s\n\n", pr.Links.HTML.Href))

			if pr.Description != "" {
				report.WriteString("### Description\n\n")
				report.WriteString(pr.Description)
				report.WriteString("\n\n")
			}

			// Fetch the diff for this PR.
			diff, err := client.GetPRDiff(ctx, workspace, repoSlug, pr.ID)
			if err != nil {
				report.WriteString(fmt.Sprintf("### Diff\n\n⚠ Failed to fetch diff: %v\n\n", err))
			} else {
				report.WriteString("### Diff\n\n```diff\n")
				report.WriteString(diff)
				report.WriteString("\n```\n\n")
			}

			if i < len(prs)-1 {
				report.WriteString("\n")
			}
		}

		return mcp.NewToolResultText(report.String()), nil
	}
}
