package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mekari/easysdlc/bitbucket"
)

// ReadPRContentTool returns the MCP tool definition for read_pr_content.
func ReadPRContentTool() mcp.Tool {
	return mcp.NewTool("read_pr_content",
		mcp.WithDescription("Read the full content of a Bitbucket pull request, including title, description, and the complete git diff."),
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
		mcp.WithNumber("pr_id",
			mcp.Description("Pull request ID"),
			mcp.Required(),
		),
	)
}

// HandleReadPRContent returns a handler that fetches PR metadata and diff.
func HandleReadPRContent(client *bitbucket.Client) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workspace, err := request.RequireString("workspace")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		repoSlug, err := request.RequireString("repo_slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		prID, err := request.RequireInt("pr_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		pr, err := client.GetPRDetails(ctx, workspace, repoSlug, prID)
		if err != nil {
			return mapError(err), nil
		}

		diff, err := client.GetPRDiff(ctx, workspace, repoSlug, prID)
		if err != nil {
			return mapError(err), nil
		}

		output := fmt.Sprintf(`# PR #%d: %s

**Author:** %s
**State:** %s
**Source:** %s → **Destination:** %s
**Created:** %s
**Updated:** %s
**URL:** %s

## Description

%s

## Diff

%s`,
			pr.ID, pr.Title,
			pr.Author.DisplayName,
			pr.State,
			pr.Source.Branch.Name, pr.Destination.Branch.Name,
			pr.CreatedOn,
			pr.UpdatedOn,
			pr.Links.HTML.Href,
			pr.Description,
			diff,
		)

		return mcp.NewToolResultText(output), nil
	}
}
