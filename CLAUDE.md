# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**sdlc-bridge** — an MCP server in Go that provides Bitbucket Cloud integration and local Go tooling for SDLC workflows. Uses the `mark3labs/mcp-go` SDK over stdio JSON-RPC transport. Designed to work alongside an Atlassian MCP (for Confluence) and a Jira MCP.

## Build & Run

```bash
go build -o easysdlc .
BITBUCKET_TOKEN=xxx ./easysdlc
```

## Architecture

```
main.go              → Entry point: env var validation, client construction, capability registration
bitbucket/
  types.go           → Bitbucket API response structs (PR, Branch, Comment, pagination)
  client.go          → HTTP client: Bearer auth, typed errors, PR/branch/comment/diff APIs
shell/
  runner.go          → Local command execution with timeout, stdout/stderr capture, sequential runner
tools/
  errors.go          → Shared error mapping: Bitbucket errors → MCP error results
  get_recent_prs.go  → Lists open PRs from last N days (default 3)
  read_pr_content.go → Fetches PR metadata + full diff
  review_open_prs.go → Fetches all recent PRs with diffs in one call for batch review
  submit_pr_review.go→ Posts a review comment on a PR
  setup_bitbucket_branch.go → Creates branch on BB + local git fetch/checkout via shell
  run_go_verification.go    → Runs go fmt, go vet, go test ./... sequentially
  submit_bitbucket_pr.go    → Git push + creates PR via Bitbucket API
resources/
  resources.go       → MCP resource templates (PR list, PR detail)
instructions/
  prompts.go         → MCP prompt templates (review_pr, batch_code_review, summarize_recent_prs, sdlc_workflow)
```

**Key design**: `bitbucket/` is a pure HTTP client with no MCP awareness. `shell/` is a generic command runner. `tools/` bridges MCP to these packages. `main.go` is wiring only. Confluence is handled by a separate Atlassian MCP, not this server.

## MCP SDK Patterns

- **Tools**: `mcp.NewTool()` with `mcp.WithString()` / `mcp.WithNumber()` and `mcp.Required()`; handlers return `(*mcp.CallToolResult, error)` — tool errors go in `mcp.NewToolResultError()`, not as Go errors
- **Resources**: `mcp.NewResourceTemplate()` with URI templates; handlers return `([]mcp.ResourceContents, error)`
- **Prompts**: `mcp.NewPrompt()` with `mcp.WithArgument()`; handlers return `(*mcp.GetPromptResult, error)`
- stdout is reserved for JSON-RPC, all logging must go to stderr

## Shell Execution

Tools that run local commands (`setup_bitbucket_branch`, `run_go_verification`, `submit_bitbucket_pr`) use the `shell` package with a 5-minute timeout. Commands run in the `work_dir` specified by the caller (defaults to `.`).
