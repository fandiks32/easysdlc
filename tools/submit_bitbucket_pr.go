package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mekari/easysdlc/bitbucket"
	"github.com/mekari/easysdlc/shell"
)

// SubmitBitbucketPRTool returns the MCP tool definition for submit_bitbucket_pr.
func SubmitBitbucketPRTool() mcp.Tool {
	return mcp.NewTool("submit_bitbucket_pr",
		mcp.WithDescription("Push local commits to Bitbucket and open a pull request. Runs git push, then creates the PR via Bitbucket API. Returns the PR URL."),
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
		mcp.WithString("title",
			mcp.Description("Pull request title"),
			mcp.Required(),
		),
		mcp.WithString("description",
			mcp.Description("Pull request description (Markdown)"),
			mcp.Required(),
		),
		mcp.WithString("source_branch",
			mcp.Description("Source branch name"),
			mcp.Required(),
		),
		mcp.WithString("target_branch",
			mcp.Description("Target/destination branch name (default: main)"),
		),
		mcp.WithString("work_dir",
			mcp.Description("Local git repository working directory (default: current directory)"),
		),
	)
}

// HandleSubmitBitbucketPR returns a handler that pushes and creates a PR.
func HandleSubmitBitbucketPR(client *bitbucket.Client) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workspace, err := request.RequireString("workspace")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		repoSlug, err := request.RequireString("repo_slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		title, err := request.RequireString("title")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		description, err := request.RequireString("description")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		sourceBranch, err := request.RequireString("source_branch")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		targetBranch := request.GetString("target_branch", "main")
		workDir := request.GetString("work_dir", ".")

		var report strings.Builder

		// Step 1: Push local commits.
		report.WriteString("### Git Push\n\n")
		pushResult, err := shell.Run(ctx, workDir, "git", "push", "origin", sourceBranch)
		if err != nil {
			return mcp.NewToolResultError("Failed to execute git push: " + err.Error()), nil
		}
		report.WriteString(shell.FormatResults([]*shell.Result{pushResult}))

		if !pushResult.IsSuccess() {
			report.WriteString("\nGit push failed. Fix the issue and retry.")
			return mcp.NewToolResultText(report.String()), nil
		}

		// Step 2: Create PR via Bitbucket API.
		report.WriteString("### Pull Request\n\n")
		pr, err := client.CreatePR(ctx, workspace, repoSlug, title, description, sourceBranch, targetBranch)
		if err != nil {
			return mapError(err), nil
		}

		report.WriteString(fmt.Sprintf("PR #%d created successfully.\n\n", pr.ID))
		report.WriteString(fmt.Sprintf("**Title:** %s\n", pr.Title))
		report.WriteString(fmt.Sprintf("**URL:** %s\n", pr.Links.HTML.Href))
		report.WriteString(fmt.Sprintf("**Source:** %s → **Target:** %s\n", sourceBranch, targetBranch))

		return mcp.NewToolResultText(report.String()), nil
	}
}
