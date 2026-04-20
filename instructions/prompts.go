package instructions

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

//go:embed sdlc_workflow.md
var sdlcWorkflowTemplate string

//go:embed full_copilot.md
var fullCopilotTemplate string

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
		mcp.WithPromptDescription("Full SDLC workflow: analyze RFC, scan repo, plan tasks with human review gate, deduplicate and publish to Jira, set up branch from v_next, implement, verify, and submit PR."),
		mcp.WithArgument("confluence_url",
			mcp.ArgumentDescription("Confluence page URL for the RFC (fetched via Atlassian MCP)"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("jira_epic",
			mcp.ArgumentDescription("Jira epic or parent ticket key to create subtasks under (e.g. PROJ-100)"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("jira_ticket",
			mcp.ArgumentDescription("Jira ticket key for this specific work item (e.g. PROJ-123)"),
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
			mcp.ArgumentDescription("Feature branch name to create (will be branched from v_next)"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("project_key",
			mcp.ArgumentDescription("Jira project key for dedup queries (e.g. PROJ). If omitted, derived from jira_epic prefix."),
		),
	)
}

// HandleSDLCWorkflowPrompt returns a handler for the SDLC workflow prompt.
func HandleSDLCWorkflowPrompt() func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := request.Params.Arguments

		projectKey := args["project_key"]
		if projectKey == "" {
			if idx := strings.Index(args["jira_epic"], "-"); idx > 0 {
				projectKey = args["jira_epic"][:idx]
			}
		}

		replacements := map[string]string{
			"{{confluence_url}}": args["confluence_url"],
			"{{jira_epic}}":     args["jira_epic"],
			"{{jira_ticket}}":   args["jira_ticket"],
			"{{workspace}}":     args["workspace"],
			"{{repo_slug}}":     args["repo_slug"],
			"{{branch_name}}":   args["branch_name"],
			"{{project_key}}":   projectKey,
		}

		prompt := sdlcWorkflowTemplate
		for placeholder, value := range replacements {
			prompt = strings.ReplaceAll(prompt, placeholder, value)
		}

		return mcp.NewGetPromptResult(
			"Full SDLC workflow from RFC to Pull Request",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(prompt)),
			},
		), nil
	}
}

// FullCopilotPrompt returns a prompt template for the autonomous full-copilot workflow.
func FullCopilotPrompt() mcp.Prompt {
	return mcp.NewPrompt("full_copilot",
		mcp.WithPromptDescription("Autonomous copilot: takes a task title and unclear requirement, analyzes codebase, plans tasks with human review gate, creates Jira tickets, implements, and delivers a PR titled [FULL_COPILOT]."),
		mcp.WithArgument("task_title",
			mcp.ArgumentDescription("Short title for the task (e.g. 'Add user validation endpoint')"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("requirement",
			mcp.ArgumentDescription("The requirement description — can be vague or incomplete"),
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
		mcp.WithArgument("project_key",
			mcp.ArgumentDescription("Jira project key for ticket creation and dedup (e.g. PROJ)"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("jira_epic",
			mcp.ArgumentDescription("Jira epic or parent ticket key to create subtasks under (e.g. PROJ-100). Optional — omit for standalone tickets."),
		),
		mcp.WithArgument("jira_ticket",
			mcp.ArgumentDescription("Existing Jira ticket key (e.g. PROJ-456). If provided, skips analysis and Jira creation — reads ticket and goes straight to implement."),
		),
	)
}

// HandleFullCopilotPrompt returns a handler for the full-copilot workflow prompt.
func HandleFullCopilotPrompt() func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := request.Params.Arguments

		replacements := map[string]string{
			"{{task_title}}":   args["task_title"],
			"{{requirement}}":  args["requirement"],
			"{{workspace}}":    args["workspace"],
			"{{repo_slug}}":    args["repo_slug"],
			"{{project_key}}":  args["project_key"],
			"{{jira_epic}}":    args["jira_epic"],
			"{{jira_ticket}}":  args["jira_ticket"],
		}

		prompt := fullCopilotTemplate
		for placeholder, value := range replacements {
			prompt = strings.ReplaceAll(prompt, placeholder, value)
		}

		return mcp.NewGetPromptResult(
			"Full Copilot: autonomous task-to-PR workflow",
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(prompt)),
			},
		), nil
	}
}
