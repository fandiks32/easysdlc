---
name: go-mcp-tool
description: Guide for adding new MCP tools, resources, or prompts to the sdlc-bridge server. Use this skill whenever the user wants to add a new tool, create a new MCP endpoint, extend the server with new functionality, add a Bitbucket API integration as a tool, add a new prompt template, or add a new resource. Trigger even for vague requests like "add pipeline support" or "I need a tool that does X" — if it involves extending this MCP server, use this skill.
---

# Adding MCP Capabilities to sdlc-bridge

This skill encodes the exact architecture patterns used in this codebase so new tools, resources, and prompts are consistent with existing ones.

## Architecture Rules

This server has strict package boundaries. Understanding them prevents the most common mistake — putting MCP-aware code in the wrong package.

```
main.go              → Wiring only: env vars, client construction, registration
bitbucket/
  client.go          → Pure HTTP client. NO mcp imports. Bearer auth, typed errors.
  types.go           → API response structs only.
shell/
  runner.go          → Generic command execution. NO mcp imports.
tools/
  errors.go          → Shared error mapping: bitbucket errors → MCP error results
  <tool_name>.go     → One file per tool. Bridges MCP ↔ bitbucket/shell packages.
resources/
  resources.go       → MCP resource templates + handlers
instructions/
  prompts.go         → MCP prompt templates + handlers
```

**Key rule**: `bitbucket/` and `shell/` must never import `mcp-go`. They are pure library code. Only `tools/`, `resources/`, and `instructions/` import MCP.

## Adding a New Tool

### Step 1: Determine Where Logic Lives

- **Calls Bitbucket API?** → Add method to `bitbucket/client.go`, response struct to `bitbucket/types.go`
- **Runs local commands?** → Use `shell.Run()` or `shell.RunAll()`
- **Both?** → Add API method to client, use shell for local parts, combine in tool handler

### Step 2: Add API Method (if needed)

Follow the existing pattern in `bitbucket/client.go`:

```go
// DescribeAction fetches/creates/updates the thing.
func (c *Client) DescribeAction(ctx context.Context, workspace, repoSlug string, /* specific params */) (*ResponseType, error) {
    reqURL := fmt.Sprintf("%s/repositories/%s/%s/endpoint",
        baseURL, url.PathEscape(workspace), url.PathEscape(repoSlug))

    // For GET: use c.doRequest(ctx, http.MethodGet, reqURL, nil)
    // For POST: marshal payload, use c.doRequest(ctx, http.MethodPost, reqURL, bytes.NewReader(data))
    // For raw text responses (diffs): use c.doRequestRaw(ctx, http.MethodGet, reqURL)

    body, err := c.doRequest(ctx, http.MethodGet, reqURL, nil)
    if err != nil {
        return nil, err  // typed errors (AuthError, NotFoundError) handled by doRequest
    }

    var result ResponseType
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }
    return &result, nil
}
```

Add response structs to `bitbucket/types.go`. Match existing style — exported fields with `json` tags.

### Step 3: Create Tool File

Create `tools/<tool_name>.go`. Every tool file has exactly two exported functions:

**1. Tool definition function** — returns `mcp.Tool`:

```go
func MyNewToolTool() mcp.Tool {
    return mcp.NewTool("my_new_tool",
        mcp.WithDescription("Clear description of what this tool does. Written for an LLM caller."),
        // Hint annotations — set these correctly:
        mcp.WithReadOnlyHintAnnotation(true),       // true if no side effects
        mcp.WithDestructiveHintAnnotation(false),    // true if deletes/overwrites
        mcp.WithIdempotentHintAnnotation(true),      // omit if not applicable
        mcp.WithOpenWorldHintAnnotation(true),       // true if calls external APIs
        // Parameters:
        mcp.WithString("workspace",
            mcp.Description("Bitbucket workspace slug"),
            mcp.Required(),
        ),
        mcp.WithString("repo_slug",
            mcp.Description("Repository slug"),
            mcp.Required(),
        ),
        // Add tool-specific params...
        mcp.WithNumber("count",
            mcp.Description("Number of items (default: 10)"),
        ),
    )
}
```

**2. Handler function** — returns the handler closure:

```go
// If tool needs Bitbucket client:
func HandleMyNewTool(client *bitbucket.Client) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Extract required params — error goes to MCP, not Go:
        workspace, err := request.RequireString("workspace")
        if err != nil {
            return mcp.NewToolResultError(err.Error()), nil
        }

        // Extract optional params with defaults:
        count := request.GetInt("count", 10)

        // Call business logic:
        result, err := client.SomeMethod(ctx, workspace, ...)
        if err != nil {
            return mapError(err), nil  // uses shared error mapper
        }

        // Format output (markdown or JSON):
        return mcp.NewToolResultText(formattedOutput), nil
    }
}

// If tool only needs shell (no client):
func HandleMyNewTool() func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    // Same pattern, no client parameter
}
```

### Step 4: Register in main.go

Add one line to the appropriate section:

```go
// Under "Tools: Bitbucket PR review" or "Tools: SDLC workflow":
s.AddTool(tools.MyNewToolTool(), tools.HandleMyNewTool(bbClient))
// or without client:
s.AddTool(tools.MyNewToolTool(), tools.HandleMyNewTool())
```

### Step 5: Verify

Run `go build -o easysdlc .` to confirm compilation.

## Critical Patterns to Follow

### Error Handling
Tool errors are **never** returned as Go errors. They go through MCP:
- Validation errors: `return mcp.NewToolResultError(err.Error()), nil`
- Bitbucket errors: `return mapError(err), nil` (uses `tools/errors.go`)
- Shell errors: `return mcp.NewToolResultError("Failed to execute: " + err.Error()), nil`

The only time to return a Go error (second return value) is for truly unrecoverable internal failures. In practice, this almost never happens.

### Output Formatting
- Use markdown for human-readable output (PR details, reports, status)
- Use JSON (`json.MarshalIndent`) for structured data (PR lists, metadata)
- Build markdown with `strings.Builder` for multi-section reports
- Use `shell.FormatResults()` when including command output

### Parameter Types
- `mcp.WithString()` + `request.RequireString()` / `request.GetString("key", "default")`
- `mcp.WithNumber()` + `request.RequireInt()` / `request.GetInt("key", default)`
- Always add `mcp.Required()` for mandatory params
- Always add `mcp.Description()` for every param — the LLM reads these

### Logging
stdout is JSON-RPC transport. All logging goes to stderr:
```go
fmt.Fprintln(os.Stderr, "debug message")
```

## Adding a New Resource

Resources go in `resources/resources.go`. Pattern:

```go
func MyResource() mcp.ResourceTemplate {
    return mcp.NewResourceTemplate(
        "bitbucket://{workspace}/{repo_slug}/my-resource",
        "My Resource Name",
        mcp.WithTemplateDescription("What this resource provides."),
        mcp.WithTemplateMIMEType("application/json"),  // or "text/markdown"
    )
}

func HandleMyResource(client *bitbucket.Client) func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
    return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
        // Parse URI variables with fmt.Sscanf:
        var ws, repo string
        _, err := fmt.Sscanf(request.Params.URI, "bitbucket://%[^/]/%[^/]/my-resource", &ws, &repo)
        if err != nil {
            return nil, fmt.Errorf("invalid resource URI: %s", request.Params.URI)
        }

        // Fetch data, format, return:
        return []mcp.ResourceContents{
            mcp.TextResourceContents{
                URI:      request.Params.URI,
                MIMEType: "application/json",
                Text:     string(data),
            },
        }, nil
    }
}
```

Register in main.go:
```go
s.AddResourceTemplate(resources.MyResource(), resources.HandleMyResource(bbClient))
```

## Adding a New Prompt

Prompts go in `instructions/prompts.go`. Pattern:

```go
func MyPrompt() mcp.Prompt {
    return mcp.NewPrompt("my_prompt",
        mcp.WithPromptDescription("What workflow this prompt guides."),
        mcp.WithArgument("arg_name",
            mcp.ArgumentDescription("What this argument is"),
            mcp.RequiredArgument(),
        ),
    )
}

func HandleMyPrompt() func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
    return func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
        argName := request.Params.Arguments["arg_name"]

        return mcp.NewGetPromptResult(
            "Description of what the prompt produces",
            []mcp.PromptMessage{
                mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(
                    fmt.Sprintf(`Instructions for the LLM...
Arg: %s`, argName),
                )),
            },
        ), nil
    }
}
```

Register in main.go:
```go
s.AddPrompt(instructions.MyPrompt(), instructions.HandleMyPrompt())
```

## Checklist

Before considering the tool complete:

- [ ] API method in `bitbucket/client.go` (if applicable) — no MCP imports
- [ ] Response structs in `bitbucket/types.go` (if applicable)
- [ ] Tool file at `tools/<name>.go` with definition + handler
- [ ] Hint annotations set correctly (read-only, destructive, idempotent, open-world)
- [ ] All params have descriptions
- [ ] Errors use `mapError()` or `mcp.NewToolResultError()`, never Go error returns
- [ ] Registered in `main.go` in correct section
- [ ] `go build -o easysdlc .` succeeds
- [ ] `go vet ./...` passes
