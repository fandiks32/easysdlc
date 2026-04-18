package confluence

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Client is an HTTP client for the Confluence Cloud REST API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	authHeader string
}

// NewClient creates a new Confluence API client.
// baseURL should be like "https://mycompany.atlassian.net/wiki".
// Uses Basic auth with email:apiToken.
func NewClient(baseURL, email, token string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	creds := base64.StdEncoding.EncodeToString([]byte(email + ":" + token))
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		authHeader: "Basic " + creds,
	}
}

// doRequest executes an HTTP request and returns the response body.
func (c *Client) doRequest(ctx context.Context, reqURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, &AuthError{
			Message: "Confluence authentication failed. Check CONFLUENCE_EMAIL and CONFLUENCE_TOKEN.",
		}
	case resp.StatusCode == http.StatusNotFound:
		return nil, &NotFoundError{
			Message: "Confluence page not found. Verify the page ID or URL.",
		}
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return nil, &APIRequestError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("Confluence API error (HTTP %d): %s", resp.StatusCode, string(body)),
		}
	}

	return body, nil
}

// pageIDPattern matches a numeric page ID embedded in a Confluence URL path.
var pageIDPattern = regexp.MustCompile(`/pages/(\d+)`)

// ResolvePageID extracts a numeric page ID from either a raw ID string or a full Confluence URL.
// Supports formats:
//   - "12345" (raw numeric ID)
//   - "https://mycompany.atlassian.net/wiki/spaces/SPACE/pages/12345/Page+Title"
//   - "https://mycompany.atlassian.net/wiki/spaces/SPACE/pages/12345"
func ResolvePageID(input string) (string, error) {
	input = strings.TrimSpace(input)

	// If purely numeric, use directly.
	if isNumeric(input) {
		return input, nil
	}

	// Try to parse as URL and extract page ID from path.
	u, err := url.Parse(input)
	if err == nil && u.Scheme != "" {
		if m := pageIDPattern.FindStringSubmatch(u.Path); len(m) == 2 {
			return m[1], nil
		}
	}

	return "", fmt.Errorf("cannot extract page ID from %q: expected a numeric ID or a Confluence page URL containing /pages/{id}", input)
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// GetPage fetches a Confluence page by ID with its storage-format body.
func (c *Client) GetPage(ctx context.Context, pageID string) (*PageResponse, error) {
	reqURL := fmt.Sprintf("%s/rest/api/content/%s?expand=body.storage,version,space",
		c.baseURL, url.PathEscape(pageID))

	body, err := c.doRequest(ctx, reqURL)
	if err != nil {
		return nil, err
	}

	var page PageResponse
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("failed to parse page response: %w", err)
	}

	return &page, nil
}

// FetchRFC fetches a Confluence RFC page and returns it as structured data
// with the XHTML body converted to Markdown.
func (c *Client) FetchRFC(ctx context.Context, pageIDOrURL string) (*RFCContent, error) {
	pageID, err := ResolvePageID(pageIDOrURL)
	if err != nil {
		return nil, err
	}

	page, err := c.GetPage(ctx, pageID)
	if err != nil {
		return nil, err
	}

	markdown := ConvertStorageToMarkdown(page.Body.Storage.Value)

	webURL := page.Links.Base + page.Links.WebUI

	return &RFCContent{
		PageID:   page.ID,
		Title:    page.Title,
		SpaceKey: page.Space.Key,
		Version:  page.Version.Number,
		Author:   page.Version.By.DisplayName,
		URL:      webURL,
		Body:     markdown,
	}, nil
}

// RFCContent holds the processed RFC data ready for LLM consumption.
type RFCContent struct {
	PageID   string `json:"page_id"`
	Title    string `json:"title"`
	SpaceKey string `json:"space_key"`
	Version  int    `json:"version"`
	Author   string `json:"author"`
	URL      string `json:"url"`
	Body     string `json:"body"`
}
