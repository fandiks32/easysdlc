# sdlc-bridge

An MCP (Model Context Protocol) server in Go that provides Bitbucket Cloud integration and local Go tooling for SDLC workflows. Designed to work alongside an Atlassian MCP (for Confluence/Jira) and a Jira MCP.

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    Claude / MCP Client                    │
└──────┬──────────────────┬──────────────────┬─────────────┘
       │                  │                  │
  sdlc-bridge        Atlassian MCP       Jira MCP
  (this server)      (Confluence)        (tickets)
       │
       ├── Bitbucket Cloud API
       │     PRs, branches, diffs, comments
       │
       └── Local shell
             git, go fmt, go vet, go test
```

## Project Structure

```
easysdlc/
├── main.go              # Entry point: env vars, client init, tool/resource/prompt registration
├── bitbucket/
│   ├── types.go          # API response structs (PR, Branch, Comment, pagination)
│   └── client.go         # HTTP client: Bearer auth, PR/branch/comment/diff APIs
├── shell/
│   └── runner.go         # Command execution with timeout and output capture
├── tools/
│   ├── errors.go         # Shared error mapping (Bitbucket → MCP errors)
│   ├── get_recent_prs.go
│   ├── read_pr_content.go
│   ├── review_open_prs.go
│   ├── run_go_verification.go
│   ├── setup_bitbucket_branch.go
│   ├── submit_bitbucket_pr.go
│   └── submit_pr_review.go
├── resources/
│   └── resources.go      # MCP resource templates (PR list, PR detail)
├── instructions/
│   ├── prompts.go        # MCP prompt definitions and handlers
│   ├── sdlc_workflow.md  # SDLC workflow prompt template (embedded at build)
│   └── full_copilot.md   # Full copilot prompt template (embedded at build)
├── go.mod
└── go.sum
```

## Prerequisites

- Go 1.23+
- A Bitbucket Cloud **Repository Access Token** with `pullrequest` and `repository` scopes

## Build

```bash
go build -o easysdlc .
```

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `BITBUCKET_TOKEN` | yes | Bitbucket repository access token (Bearer) |

## Claude Desktop Integration

Add to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "sdlc-bridge": {
      "command": "/absolute/path/to/easysdlc",
      "env": {
        "BITBUCKET_TOKEN": "your-bb-token"
      }
    }
  }
}
```

Config file locations:
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

## Tools

### Bitbucket — PR Review

| Tool | Description | Key Parameters |
|---|---|---|
| `get_recent_prs` | List open PRs from last N days (default: 3) | `workspace`, `repo_slug`, `days` |
| `read_pr_content` | Fetch PR metadata + full diff | `workspace`, `repo_slug`, `pr_id` |
| `review_open_prs` | Fetch all recent open PRs with their diffs in one call, ready for code review | `workspace`, `repo_slug`, `days` |
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

## Prompts

| Prompt | Description |
|---|---|
| `review_pr` | Guided code review workflow for a single PR |
| `batch_code_review` | Fetch all open PRs from the last 3 days and code review each one |
| `summarize_recent_prs` | Summary of recent open PRs |
| `sdlc_workflow` | Full RFC→Analysis→Breakdown→Human Review→Jira Dedup→Branch→Code→Verify→PR workflow |
| `full_copilot` | Autonomous copilot: vague requirement→codebase analysis→breakdown→Jira→implement→PR |

### sdlc_workflow

Phased pipeline for implementing a Confluence RFC with full traceability:

```
Phase 1:  Fetch RFC + comments, scan repo          → Structured analysis
Phase 2:  Comprehensive task breakdown              → Atomic tasks with tests
  GATE:   Human review (mandatory stop)
Phase 2b: Jira dedup + issue creation               → Idempotent ticket creation
Phase 3:  Branch from v_next + implement per task
Phase 4:  go fmt / go vet / go test
Phase 5:  PR targeting v_next
```

Parameters: `confluence_url`, `jira_epic`, `jira_ticket`, `workspace`, `repo_slug`, `branch_name`, `project_key` (optional)

### full_copilot

Autonomous workflow starting from a task title and unclear requirement — no RFC needed:

```
Phase 1:  Analyze codebase + understand requirement → Structured analysis with assumptions
Phase 2:  Comprehensive task breakdown              → Atomic tasks with tests
  GATE:   Human review (mandatory stop)
Phase 3:  Create Jira tickets (dedup) + self-assign
Phase 4:  Branch from v_next + implement per task
Phase 5:  go fmt / go vet / go test
Phase 6:  PR titled [FULL_COPILOT]: #TICKET #Title + Jira comments
```

Parameters: `task_title`, `requirement`, `workspace`, `repo_slug`, `project_key`, `jira_epic` (optional)

## Troubleshooting

| Error | Cause |
|-------|-------|
| `Authentication failed` | Token is invalid, expired, or lacks required scopes |
| `Resource not found` | Incorrect workspace, repo_slug, or pr_id |
| `Request timed out` | API did not respond within 30 seconds |
| `Source branch does not exist` | The `from_branch` for branch creation doesn't exist |
