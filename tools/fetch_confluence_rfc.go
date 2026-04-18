package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mekari/easysdlc/confluence"
)

// FetchConfluenceRFCTool returns the MCP tool definition for fetch_confluence_rfc.
func FetchConfluenceRFCTool() mcp.Tool {
	return mcp.NewTool("fetch_confluence_rfc",
		mcp.WithDescription("Fetch an RFC or design document from Confluence. Accepts a page ID or full Confluence URL. Returns the page content converted from XHTML to clean Markdown for easy consumption."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("page_id",
			mcp.Description("Confluence page ID (numeric) or full page URL"),
			mcp.Required(),
		),
	)
}

// HandleFetchConfluenceRFC returns a handler that fetches and converts an RFC from Confluence.
func HandleFetchConfluenceRFC(client *confluence.Client) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pageIDOrURL, err := request.RequireString("page_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		rfc, err := client.FetchRFC(ctx, pageIDOrURL)
		if err != nil {
			return mapConfluenceError(err), nil
		}

		output := fmt.Sprintf(`# %s

**Page ID:** %s
**Space:** %s
**Version:** %d
**Author:** %s
**URL:** %s

---

%s`,
			rfc.Title,
			rfc.PageID,
			rfc.SpaceKey,
			rfc.Version,
			rfc.Author,
			rfc.URL,
			rfc.Body,
		)

		return mcp.NewToolResultText(output), nil
	}
}
