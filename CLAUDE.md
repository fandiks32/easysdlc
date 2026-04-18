# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**sdlc-bridge** — an MCP server in Go that bridges Confluence (RFCs) and Bitbucket (Code/PRs) into a unified SDLC workflow. Uses the `mark3labs/mcp-go` SDK over stdio JSON-RPC transport. Designed to work alongside an existing Jira MCP.

## Build & Run

```bash
go build -o easysdlc .
BITBUCKET_TOKEN=xxx ./easysdlc                               # Bitbucket only
BITBUCKET_TOKEN=xxx CONFLUENCE_BASE_URL=... CONFLUENCE_EMAIL=... CONFLUENCE_TOKEN=... ./easysdlc  # full mode
```

## Architecture

```
main.go              → Entry point: env var validation, client construction, capability registration
bitbucket/
  types.go           → Bitbucket API response structs (PR, Branch, Comment, pagination)
  client.go          → HTTP client: Bearer auth, typed errors, PR/branch/comment APIs
confluence/
  types.go           → Confluence API response structs (Page, Space, Version)
  client.go          → HTTP client: Basic auth (email:token), page fetching, URL/ID resolution
  convert.go         → XHTML storage format → Markdown converter (recursive HTML tree walker)
shell/
  runner.go          → Local command execution with timeout, stdout/stderr capture, sequential runner
tools/
  errors.go          → Shared error mapping for both Bitbucket and Confluence → MCP error results
  get_recent_prs.go  → Lists open PRs from last 48h
  read_pr_content.go → Fetches PR metadata + full diff
  submit_pr_review.go→ Posts a review comment on a PR
  fetch_confluence_rfc.go   → Fetches RFC from Confluence, converts XHTML→Markdown
  setup_bitbucket_branch.go → Creates branch on BB + local git fetch/checkout via shell
  run_go_verification.go    → Runs go fmt, go vet, go test ./... sequentially
  submit_bitbucket_pr.go    → Git push + creates PR via Bitbucket API
resources/
  resources.go       → MCP resource templates (PR list, PR detail, Confluence RFC)
instructions/
  prompts.go         → MCP prompt templates (review_pr, summarize_recent_prs, sdlc_workflow)
```

**Key design**: `bitbucket/` and `confluence/` are pure HTTP clients with no MCP awareness. `shell/` is a generic command runner. `tools/` bridges MCP to these packages. `main.go` is wiring only. Confluence integration is optional — tools/resources are only registered when env vars are set.

## MCP SDK Patterns

- **Tools**: `mcp.NewTool()` with `mcp.WithString()` / `mcp.WithNumber()` and `mcp.Required()`; handlers return `(*mcp.CallToolResult, error)` — tool errors go in `mcp.NewToolResultError()`, not as Go errors
- **Resources**: `mcp.NewResourceTemplate()` with URI templates; handlers return `([]mcp.ResourceContents, error)`
- **Prompts**: `mcp.NewPrompt()` with `mcp.WithArgument()`; handlers return `(*mcp.GetPromptResult, error)`
- stdout is reserved for JSON-RPC, all logging must go to stderr

## API Auth Details

- **Bitbucket**: Bearer token via `Authorization: Bearer <BITBUCKET_TOKEN>` header
- **Confluence**: Basic auth via `Authorization: Basic base64(CONFLUENCE_EMAIL:CONFLUENCE_TOKEN)` header
- Base URLs: Bitbucket=`https://api.bitbucket.org/2.0`, Confluence=`CONFLUENCE_BASE_URL/rest/api/content`

## Shell Execution

Tools that run local commands (`setup_bitbucket_branch`, `run_go_verification`, `submit_bitbucket_pr`) use the `shell` package with a 5-minute timeout. Commands run in the `work_dir` specified by the caller (defaults to `.`).
