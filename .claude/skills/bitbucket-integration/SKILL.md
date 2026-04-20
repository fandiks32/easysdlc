---
name: bitbucket-integration
description: Guide for extending the Bitbucket HTTP client in bitbucket/client.go with new API endpoints. Use this skill when the user wants to add webhook support, pipeline triggers, repository settings, branch permissions, deployment environments, or any other Bitbucket Cloud API v2.0 endpoint. Also trigger for "add bitbucket API for X", "extend bitbucket client", "I need to call bitbucket endpoint Y".
---

# Extending the Bitbucket Client

This skill covers how to add new Bitbucket Cloud API v2.0 endpoints to `bitbucket/client.go`. The client is a pure HTTP layer — no MCP awareness, no business logic beyond request/response handling.

## Client Architecture

```
bitbucket/
  client.go   → HTTP methods, error handling, request execution
  types.go    → Response/request structs with JSON tags
```

The `Client` struct provides:
- **`doRequest(ctx, method, url, body)`** → `([]byte, error)` — for JSON responses
- **`doRequestRaw(ctx, method, url)`** → `(string, error)` — for plain text (diffs, raw content)

Both methods handle:
- Bearer token auth via `Authorization` header
- Content-Type for POST/PUT/PATCH
- Typed error mapping: 401/403 → `AuthError`, 404 → `NotFoundError`, other non-2xx → `APIRequestError`

## Adding a GET Endpoint

For endpoints that return JSON:

```go
// GetThing fetches a specific thing.
func (c *Client) GetThing(ctx context.Context, workspace, repoSlug string, thingID int) (*Thing, error) {
    reqURL := fmt.Sprintf("%s/repositories/%s/%s/things/%d",
        baseURL, url.PathEscape(workspace), url.PathEscape(repoSlug), thingID)

    body, err := c.doRequest(ctx, http.MethodGet, reqURL, nil)
    if err != nil {
        return nil, err
    }

    var thing Thing
    if err := json.Unmarshal(body, &thing); err != nil {
        return nil, fmt.Errorf("failed to parse thing response: %w", err)
    }
    return &thing, nil
}
```

## Adding a POST Endpoint

For endpoints that create resources:

```go
// CreateThing creates a new thing.
func (c *Client) CreateThing(ctx context.Context, workspace, repoSlug, name string) (*Thing, error) {
    reqURL := fmt.Sprintf("%s/repositories/%s/%s/things",
        baseURL, url.PathEscape(workspace), url.PathEscape(repoSlug))

    payload := map[string]any{
        "name": name,
    }
    data, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal thing payload: %w", err)
    }

    body, err := c.doRequest(ctx, http.MethodPost, reqURL, bytes.NewReader(data))
    if err != nil {
        return nil, err
    }

    var thing Thing
    if err := json.Unmarshal(body, &thing); err != nil {
        return nil, fmt.Errorf("failed to parse thing response: %w", err)
    }
    return &thing, nil
}
```

## Adding a Paginated List Endpoint

Follow `ListRecentPRs` pattern exactly:

```go
func (c *Client) ListThings(ctx context.Context, workspace, repoSlug string) ([]Thing, error) {
    params := url.Values{}
    params.Set("pagelen", "50")
    // Add filters as needed:
    // params.Set("q", `state = "active"`)

    reqURL := fmt.Sprintf("%s/repositories/%s/%s/things?%s",
        baseURL, url.PathEscape(workspace), url.PathEscape(repoSlug), params.Encode())

    var all []Thing
    for i := 0; i < 100; i++ { // safety cap prevents infinite loops
        body, err := c.doRequest(ctx, http.MethodGet, reqURL, nil)
        if err != nil {
            return nil, err
        }

        // Use a local paginated struct if response shape differs from PaginatedResponse:
        var page struct {
            Values []Thing `json:"values"`
            Next   string  `json:"next"`
        }
        if err := json.Unmarshal(body, &page); err != nil {
            return nil, fmt.Errorf("failed to parse response: %w", err)
        }

        all = append(all, page.Values...)
        if page.Next == "" {
            break
        }
        reqURL = page.Next
    }
    return all, nil
}
```

## Adding Response Types

Add to `bitbucket/types.go`. Match existing conventions:

```go
// Thing represents a Bitbucket thing.
type Thing struct {
    ID        int    `json:"id"`
    Name      string `json:"name"`
    State     string `json:"state"`
    CreatedOn string `json:"created_on"`
    // Nest objects as needed:
    Links     Links  `json:"links"`
}
```

Rules:
- Exported fields only
- Always include `json` tags matching Bitbucket API response keys
- Reuse existing types (`Links`, `User`, `Branch`) where they match
- Add doc comments on types explaining what they represent
- Check Bitbucket API docs for actual field names — don't guess

## Handling Exists/Check Pattern

When you need to check if something exists before creating it (like `BranchExists`):

```go
func (c *Client) ThingExists(ctx context.Context, workspace, repoSlug, name string) (bool, error) {
    reqURL := fmt.Sprintf("%s/repositories/%s/%s/things/%s",
        baseURL, url.PathEscape(workspace), url.PathEscape(repoSlug), url.PathEscape(name))

    _, err := c.doRequest(ctx, http.MethodGet, reqURL, nil)
    if err != nil {
        var notFound *NotFoundError
        if errors.As(err, &notFound) {
            return false, nil  // doesn't exist, not an error
        }
        return false, err  // actual error
    }
    return true, nil
}
```

## Error Handling Rules

The client returns typed errors. Never wrap them with additional context that hides the type:

```go
// WRONG — wrapping hides the type from errors.As():
return nil, fmt.Errorf("branch check failed: %w", err)

// RIGHT — let doRequest errors pass through:
return nil, err

// RIGHT — wrap only for context on your own errors:
return nil, fmt.Errorf("failed to resolve source branch %q: %w", fromBranch, err)
```

The tool layer (`tools/errors.go`) uses `errors.As()` to map these typed errors to MCP error results. If you wrap them incorrectly, `errors.As()` still works (because of `%w`), but adding unnecessary wrapping just adds noise.

## Query Parameters

Bitbucket v2.0 supports filtering via the `q` parameter with a custom query language:

```go
params.Set("q", `state = "OPEN"`)
params.Set("q", fmt.Sprintf(`created_on > "%s"`, cutoffStr))
params.Set("sort", "-created_on")  // descending
```

Date format: `2006-01-02T15:04:05.000000+00:00`

## Common Bitbucket API Endpoints

Reference for what's available to integrate:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/repositories/{ws}/{repo}/pipelines/` | GET/POST | Pipeline runs |
| `/repositories/{ws}/{repo}/deploy-keys/` | GET/POST | Deploy keys |
| `/repositories/{ws}/{repo}/webhooks/` | GET/POST | Webhooks |
| `/repositories/{ws}/{repo}/branch-restrictions/` | GET/POST | Branch permissions |
| `/repositories/{ws}/{repo}/environments/` | GET | Deployment environments |
| `/repositories/{ws}/{repo}/downloads/` | GET/POST | Repository downloads |
| `/repositories/{ws}/{repo}/commit/{sha}/statuses/` | GET/POST | Build statuses |

Always verify exact endpoint paths against Bitbucket Cloud API docs — endpoints evolve.

## Testing New Client Methods

Write table-driven tests following Go conventions:

```go
func TestClient_GetThing(t *testing.T) {
    // Use httptest.NewServer to mock Bitbucket responses
    // Test: success case, 404, 401, timeout
}
```

Focus on:
- Correct URL construction (path escaping)
- Correct HTTP method
- Request body shape (for POST/PUT)
- Error type mapping
- Pagination termination
