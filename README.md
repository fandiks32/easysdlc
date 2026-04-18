# sdlc-bridge

An MCP (Model Context Protocol) server in Go that bridges Confluence (RFCs) and Bitbucket (Code/PRs) into a unified SDLC workflow. Designed to work alongside an existing Jira MCP.

## Prerequisites

- Go 1.23+
- A Bitbucket Cloud **Repository Access Token** with `pullrequest` and `repository` scopes
- (Optional) Confluence Cloud **API Token** with page read access

## Build

```bash
go build -o easysdlc .
```

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `BITBUCKET_TOKEN` | yes | Bitbucket repository access token (Bearer) |
| `CONFLUENCE_BASE_URL` | no | Confluence base URL (e.g. `https://mycompany.atlassian.net/wiki`) |
| `CONFLUENCE_EMAIL` | no | Email for Confluence Basic auth |
| `CONFLUENCE_TOKEN` | no | Confluence API token |

Confluence tools are only registered when all three `CONFLUENCE_*` variables are set.

## Claude Desktop Integration

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "sdlc-bridge": {
      "command": "/absolute/path/to/easysdlc",
      "env": {
        "BITBUCKET_TOKEN": "your-bb-token",
        "CONFLUENCE_BASE_URL": "https://mycompany.atlassian.net/wiki",
        "CONFLUENCE_EMAIL": "you@company.com",
        "CONFLUENCE_TOKEN": "your-confluence-token"
      }
    }
  }
}
```

Config file locations:
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

## Tools

### Confluence

| Tool | Description | Key Parameters |
|---|---|---|
| `fetch_confluence_rfc` | Fetch an RFC page, convert XHTML to Markdown | `page_id` (numeric ID or full URL) |

### Bitbucket — PR Review

| Tool | Description | Key Parameters |
|---|---|---|
| `get_recent_prs` | List open PRs from last 48h | `workspace`, `repo_slug` |
| `read_pr_content` | Fetch PR metadata + full diff | `workspace`, `repo_slug`, `pr_id` |
| `submit_pr_review` | Post a review comment on a PR | `workspace`, `repo_slug`, `pr_id`, `review_text` |

### SDLC Workflow

| Tool | Description | Key Parameters |
|---|---|---|
| `setup_bitbucket_branch` | Create branch on Bitbucket + local git checkout | `workspace`, `repo_slug`, `branch_name` |
| `run_go_verification` | Run `go fmt`, `go vet`, `go test ./...` | `work_dir`, `test_args` |
| `submit_bitbucket_pr` | Git push + create PR via API | `workspace`, `repo_slug`, `title`, `description`, `source_branch` |

## Resources

| URI Template | Description |
|---|---|
| `bitbucket://{workspace}/{repo_slug}/pull-requests` | Open PRs (JSON) |
| `bitbucket://{workspace}/{repo_slug}/pull-requests/{pr_id}` | PR detail + diff (Markdown) |
| `confluence://pages/{page_id}` | Confluence RFC (Markdown) |

## Prompts

| Prompt | Description |
|---|---|
| `review_pr` | Guided code review workflow for a PR |
| `summarize_recent_prs` | Summary of recent open PRs |
| `sdlc_workflow` | Full RFC→Branch→Code→Verify→PR workflow |

## Intended Workflow

```
1. fetch_confluence_rfc  →  Understand the RFC requirements
2. (Jira MCP)            →  Create/update tickets
3. setup_bitbucket_branch →  Create branch & check out locally
4. (Code locally)        →  Implement the feature
5. run_go_verification   →  Verify quality (fix & re-run until green)
6. submit_bitbucket_pr   →  Push & open the PR
```

## Troubleshooting

| Error | Cause |
|-------|-------|
| `Authentication failed` (Bitbucket) | Token is invalid, expired, or lacks required scopes |
| `Confluence auth failed` | Check CONFLUENCE_EMAIL and CONFLUENCE_TOKEN |
| `Resource not found` | Incorrect workspace, repo_slug, pr_id, or page_id |
| `Request timed out` | API did not respond within 30 seconds |
| `Source branch does not exist` | The `from_branch` for branch creation doesn't exist |
