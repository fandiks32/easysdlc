package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mekari/easysdlc/bitbucket"
	"github.com/mekari/easysdlc/shell"
)

// SetupBitbucketBranchTool returns the MCP tool definition for setup_bitbucket_branch.
func SetupBitbucketBranchTool() mcp.Tool {
	return mcp.NewTool("setup_bitbucket_branch",
		mcp.WithDescription("Set up a working branch for development. Checks if the branch exists on Bitbucket, creates it if not (from the target branch), then runs git fetch and git checkout locally to sync the dev environment."),
		mcp.WithReadOnlyHintAnnotation(false),
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
		mcp.WithString("branch_name",
			mcp.Description("Name of the branch to create or check out"),
			mcp.Required(),
		),
		mcp.WithString("from_branch",
			mcp.Description("Source branch to create from (default: main)"),
		),
		mcp.WithString("work_dir",
			mcp.Description("Local git repository working directory (default: current directory)"),
		),
	)
}

// HandleSetupBitbucketBranch returns a handler that sets up a branch both remotely and locally.
func HandleSetupBitbucketBranch(client *bitbucket.Client) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workspace, err := request.RequireString("workspace")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		repoSlug, err := request.RequireString("repo_slug")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		branchName, err := request.RequireString("branch_name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		fromBranch := request.GetString("from_branch", "main")
		workDir := request.GetString("work_dir", ".")

		var report strings.Builder

		// Step 1: Check if branch exists on Bitbucket.
		exists, _, err := client.BranchExists(ctx, workspace, repoSlug, branchName)
		if err != nil {
			return mapError(err), nil
		}

		if exists {
			report.WriteString(fmt.Sprintf("Branch `%s` already exists on Bitbucket.\n\n", branchName))
		} else {
			// Create the branch remotely.
			branch, err := client.CreateBranch(ctx, workspace, repoSlug, branchName, fromBranch)
			if err != nil {
				return mapError(err), nil
			}
			report.WriteString(fmt.Sprintf("Created branch `%s` on Bitbucket from `%s` (commit: `%.12s`).\n\n", branchName, fromBranch, branch.Target.Hash))
		}

		// Step 2: Sync locally.
		report.WriteString("### Local sync\n\n")

		commands := [][]string{
			{"git", "fetch", "origin"},
			{"git", "checkout", branchName},
		}

		results, err := shell.RunAll(ctx, workDir, commands)
		if err != nil {
			return mcp.NewToolResultError("Failed to execute git commands: " + err.Error()), nil
		}

		report.WriteString(shell.FormatResults(results))

		allPassed := true
		for _, r := range results {
			if !r.IsSuccess() {
				allPassed = false
				break
			}
		}

		if allPassed {
			report.WriteString(fmt.Sprintf("\nReady to develop on branch `%s`.", branchName))
		} else {
			report.WriteString("\nLocal sync encountered errors. Review the output above.")
		}

		return mcp.NewToolResultText(report.String()), nil
	}
}
