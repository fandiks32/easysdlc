package instructions

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// ReviewPRPrompt returns a prompt template for reviewing a pull request.
func ReviewPRPrompt() mcp.Prompt {
	return mcp.NewPrompt("review_pr",
		mcp.WithPromptDescription("Generate a thorough code review for a Bitbucket pull request. Uses read_pr_content to fetch the PR, then produces a structured review."),
		mcp.WithArgument("workspace",
			mcp.ArgumentDescription("Bitbucket workspace slug"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("repo_slug",
			mcp.ArgumentDescription("Repository slug"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("pr_id",
			mcp.ArgumentDescription("Pull request ID"),
			mcp.RequiredArgument(),
		),
	)
}

// HandleReviewPRPrompt returns a handler that produces review instructions for the LLM.
func HandleReviewPRPrompt() func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		workspace := request.Params.Arguments["workspace"]
		repoSlug := request.Params.Arguments["repo_slug"]
		prID := request.Params.Arguments["pr_id"]

		return mcp.NewGetPromptResult(
			"Code review instructions for a Bitbucket pull request",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(
					fmt.Sprintf(`Please review the pull request #%s in the repository %s/%s.

Steps:
1. Use the read_pr_content tool with workspace=%q, repo_slug=%q, pr_id=%s to fetch the PR details and diff.
2. Analyze the changes and provide a structured review covering:
   - **Summary**: A brief overview of what the PR does.
   - **Strengths**: What was done well.
   - **Issues**: Bugs, logic errors, or security concerns (with file/line references from the diff).
   - **Suggestions**: Improvements for readability, performance, or maintainability.
   - **Verdict**: APPROVE, REQUEST_CHANGES, or COMMENT.
3. Use the submit_pr_review tool to post your review as a comment on the PR.`, prID, workspace, repoSlug, workspace, repoSlug, prID),
				)),
			},
		), nil
	}
}

// SummarizeRecentPRsPrompt returns a prompt for summarizing recent pull requests.
func SummarizeRecentPRsPrompt() mcp.Prompt {
	return mcp.NewPrompt("summarize_recent_prs",
		mcp.WithPromptDescription("Fetch and summarize all recent open pull requests for a repository."),
		mcp.WithArgument("workspace",
			mcp.ArgumentDescription("Bitbucket workspace slug"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("repo_slug",
			mcp.ArgumentDescription("Repository slug"),
			mcp.RequiredArgument(),
		),
	)
}

// HandleSummarizeRecentPRsPrompt returns a handler that produces summary instructions.
func HandleSummarizeRecentPRsPrompt() func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		workspace := request.Params.Arguments["workspace"]
		repoSlug := request.Params.Arguments["repo_slug"]

		return mcp.NewGetPromptResult(
			"Summarize recent open pull requests",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(
					fmt.Sprintf(`Please summarize the recent open pull requests for the repository %s/%s.

Steps:
1. Use the get_recent_prs tool with workspace=%q, repo_slug=%q to fetch all open PRs from the last 48 hours.
2. For each PR, provide:
   - PR number, title, and author
   - A one-line summary of what it does (based on the title/description)
   - How old it is
3. At the end, provide a brief overall status: how many PRs are open, any that look stale or urgent.`, workspace, repoSlug, workspace, repoSlug),
				)),
			},
		), nil
	}
}

// BatchCodeReviewPrompt returns a prompt for reviewing all recent open PRs.
func BatchCodeReviewPrompt() mcp.Prompt {
	return mcp.NewPrompt("batch_code_review",
		mcp.WithPromptDescription("Fetch all open PRs from the last 3 days and perform a code review on each one, posting review comments."),
		mcp.WithArgument("workspace",
			mcp.ArgumentDescription("Bitbucket workspace slug"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("repo_slug",
			mcp.ArgumentDescription("Repository slug"),
			mcp.RequiredArgument(),
		),
	)
}

// HandleBatchCodeReviewPrompt returns a handler for batch code review.
func HandleBatchCodeReviewPrompt() func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		workspace := request.Params.Arguments["workspace"]
		repoSlug := request.Params.Arguments["repo_slug"]

		return mcp.NewGetPromptResult(
			"Batch code review of recent open pull requests",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(
					fmt.Sprintf(`Review all recent open pull requests for the repository %s/%s.

## Step 1: Fetch All PRs
Use the review_open_prs tool with workspace=%q, repo_slug=%q, days=3 to fetch all open PRs from the last 3 days along with their diffs.

## Step 2: Review Each PR
For each PR, provide a structured code review:
- **Summary**: What the PR does in 1-2 sentences.
- **Strengths**: What was done well.
- **Issues**: Bugs, logic errors, security concerns, or style violations (cite specific files and lines from the diff).
- **Suggestions**: Improvements for readability, performance, or maintainability.
- **Verdict**: APPROVE, REQUEST_CHANGES, or COMMENT.

## Step 3: Post Reviews
For each PR, use the submit_pr_review tool with workspace=%q, repo_slug=%q to post the review as a comment.

## Step 4: Summary
After reviewing all PRs, provide a brief summary:
- Total PRs reviewed
- How many approved vs. need changes
- Any cross-cutting concerns seen across multiple PRs`, workspace, repoSlug, workspace, repoSlug, workspace, repoSlug),
				)),
			},
		), nil
	}
}

// SDLCWorkflowPrompt returns a prompt template for the full RFC-to-PR workflow.
func SDLCWorkflowPrompt() mcp.Prompt {
	return mcp.NewPrompt("sdlc_workflow",
		mcp.WithPromptDescription("Full SDLC workflow: read an RFC via Atlassian MCP, break down tasks in Jira, set up a branch, implement, verify, and submit a PR on Bitbucket."),
		mcp.WithArgument("confluence_url",
			mcp.ArgumentDescription("Confluence page URL for the RFC (fetched via Atlassian MCP)"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("jira_ticket",
			mcp.ArgumentDescription("Jira ticket key to post the task breakdown to (e.g. PROJ-123)"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("workspace",
			mcp.ArgumentDescription("Bitbucket workspace slug"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("repo_slug",
			mcp.ArgumentDescription("Repository slug"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("branch_name",
			mcp.ArgumentDescription("Feature branch name to create"),
			mcp.RequiredArgument(),
		),
	)
}

// HandleSDLCWorkflowPrompt returns a handler for the SDLC workflow prompt.
func HandleSDLCWorkflowPrompt() func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		confluenceURL := request.Params.Arguments["confluence_url"]
		jiraTicket := request.Params.Arguments["jira_ticket"]
		workspace := request.Params.Arguments["workspace"]
		repoSlug := request.Params.Arguments["repo_slug"]
		branchName := request.Params.Arguments["branch_name"]

		return mcp.NewGetPromptResult(
			"Full SDLC workflow from RFC to Pull Request",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(
					fmt.Sprintf(`Execute the full SDLC workflow for implementing an RFC.

IMPORTANT: Execute each step independently with fresh context. Do not carry assumptions between steps — re-read the relevant data at the start of each step.

## Step 1: Understand the Requirements
- Confluence RFC URL: %s
- Use the Atlassian MCP to fetch the RFC page at the URL above.
- Read and summarize the key requirements, acceptance criteria, and technical constraints.

## Step 2: Post Task Breakdown to Jira
- Jira ticket: %s
- Re-read the RFC summary from Step 1.
- Break down the work into concrete subtasks.
- Use the Jira MCP to create subtasks under ticket %s with issue type "Task". Each subtask should be clear, actionable, and estimable.

## Step 3: Set Up the Branch
- Workspace: %q, Repo: %q, Branch: %q
- Use setup_bitbucket_branch with the parameters above to create and check out the feature branch.
- Verify you are on the correct branch before proceeding.

## Step 4: Implement
- Re-read the RFC requirements and the Jira subtasks from Step 2.
- Write the code to fulfill the requirements. Follow the existing code patterns and conventions in the repository.
- Add comments to Jira ticket %s via the Jira MCP to log progress as you work through each subtask.

## Step 5: Verify
- Run run_go_verification to execute go fmt, go vet, and go test.
- If any checks fail, read the error output, fix the issues, and re-run until all pass.
- Do not proceed to Step 6 until all checks are green.

## Step 6: Submit
- Workspace: %q, Repo: %q, Source branch: %q
- Use submit_bitbucket_pr with the parameters above to push and create the pull request.
- Write a clear PR title and description that references the RFC (%s) and Jira ticket %s.`,
						confluenceURL,
						jiraTicket, jiraTicket,
						workspace, repoSlug, branchName,
						jiraTicket,
						workspace, repoSlug, branchName,
						confluenceURL, jiraTicket),
				)),
			},
		), nil
	}
}
