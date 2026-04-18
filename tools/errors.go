package tools

import (
	"context"
	"errors"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mekari/easysdlc/bitbucket"
	"github.com/mekari/easysdlc/confluence"
)

// mapError converts a Bitbucket client error into an MCP tool error result.
func mapError(err error) *mcp.CallToolResult {
	var bbAuth *bitbucket.AuthError
	if errors.As(err, &bbAuth) {
		return mcp.NewToolResultError("Authentication failed: " + bbAuth.Message)
	}

	var bbNotFound *bitbucket.NotFoundError
	if errors.As(err, &bbNotFound) {
		return mcp.NewToolResultError("Not found: " + bbNotFound.Message)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return mcp.NewToolResultError("Request timed out. The API did not respond in time.")
	}

	return mcp.NewToolResultErrorFromErr("Bitbucket API error", err)
}

// mapConfluenceError converts a Confluence client error into an MCP tool error result.
func mapConfluenceError(err error) *mcp.CallToolResult {
	var authErr *confluence.AuthError
	if errors.As(err, &authErr) {
		return mcp.NewToolResultError("Confluence auth failed: " + authErr.Message)
	}

	var notFoundErr *confluence.NotFoundError
	if errors.As(err, &notFoundErr) {
		return mcp.NewToolResultError("Confluence page not found: " + notFoundErr.Message)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return mcp.NewToolResultError("Request timed out. Confluence did not respond in time.")
	}

	return mcp.NewToolResultErrorFromErr("Confluence API error", err)
}
