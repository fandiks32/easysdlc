package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mekari/easysdlc/bitbucket"
)

// PRListResource returns a resource template for listing open PRs in a repository.
func PRListResource() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		"bitbucket://{workspace}/{repo_slug}/pull-requests",
		"Open Pull Requests",
		mcp.WithTemplateDescription("List of currently open pull requests for a Bitbucket repository."),
		mcp.WithTemplateMIMEType("application/json"),
	)
}

// HandlePRListResource returns a handler that fetches open PRs for the resource template.
func HandlePRListResource(client *bitbucket.Client) func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		workspace := request.Params.URI
		// Parse workspace and repo_slug from the URI: bitbucket://{workspace}/{repo_slug}/pull-requests
		var ws, repo string
		_, err := fmt.Sscanf(request.Params.URI, "bitbucket://%[^/]/%[^/]/pull-requests", &ws, &repo)
		if err != nil {
			return nil, fmt.Errorf("invalid resource URI: %s", workspace)
		}

		prs, err := client.ListRecentPRs(ctx, ws, repo, 48)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch pull requests: %w", err)
		}

		data, err := json.MarshalIndent(prs, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal pull requests: %w", err)
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "application/json",
				Text:     string(data),
			},
		}, nil
	}
}

// PRDetailResource returns a resource template for a specific PR's details.
func PRDetailResource() mcp.ResourceTemplate {
	return mcp.NewResourceTemplate(
		"bitbucket://{workspace}/{repo_slug}/pull-requests/{pr_id}",
		"Pull Request Detail",
		mcp.WithTemplateDescription("Detailed information about a specific Bitbucket pull request, including metadata and diff."),
		mcp.WithTemplateMIMEType("text/markdown"),
	)
}

// HandlePRDetailResource returns a handler that fetches a specific PR's details and diff.
func HandlePRDetailResource(client *bitbucket.Client) func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	return func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		var ws, repo string
		var prID int
		_, err := fmt.Sscanf(request.Params.URI, "bitbucket://%[^/]/%[^/]/pull-requests/%d", &ws, &repo, &prID)
		if err != nil {
			return nil, fmt.Errorf("invalid resource URI: %s", request.Params.URI)
		}

		pr, err := client.GetPRDetails(ctx, ws, repo, prID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch PR details: %w", err)
		}

		diff, err := client.GetPRDiff(ctx, ws, repo, prID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch PR diff: %w", err)
		}

		output := fmt.Sprintf(`# PR #%d: %s

**Author:** %s
**State:** %s
**Source:** %s → **Destination:** %s
**Created:** %s
**Updated:** %s
**URL:** %s

## Description

%s

## Diff

%s`,
			pr.ID, pr.Title,
			pr.Author.DisplayName,
			pr.State,
			pr.Source.Branch.Name, pr.Destination.Branch.Name,
			pr.CreatedOn,
			pr.UpdatedOn,
			pr.Links.HTML.Href,
			pr.Description,
			diff,
		)

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      request.Params.URI,
				MIMEType: "text/markdown",
				Text:     output,
			},
		}, nil
	}
}

