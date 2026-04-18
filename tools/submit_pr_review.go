package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mekari/easysdlc/bitbucket"
)

// SubmitPRReviewTool returns the MCP tool definition for submit_pr_review.
func SubmitPRReviewTool() mcp.Tool {
	return mcp.NewTool("submit_pr_review",
		mcp.WithDescription("Post a review comment on a Bitbucket pull request."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
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
		mcp.WithString("review_text",
			mcp.Description("The review comment text to post on the pull request"),
			mcp.Required(),
		),
	)
}

// HandleSubmitPRReview returns a handler that posts a review comment on a PR.
func HandleSubmitPRReview(client *bitbucket.Client) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		reviewText, err := request.RequireString("review_text")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		comment, err := client.PostComment(ctx, workspace, repoSlug, prID, reviewText)
		if err != nil {
			return mapError(err), nil
		}

		return mcp.NewToolResultText(
			fmt.Sprintf("Review comment posted successfully on PR #%d (comment ID: %d).", prID, comment.ID),
		), nil
	}
}
