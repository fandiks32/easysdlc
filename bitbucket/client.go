package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const baseURL = "https://api.bitbucket.org/2.0"

// AuthError indicates a 401/403 authentication failure.
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}

// NotFoundError indicates a 404 response.
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}

// APIRequestError indicates a non-2xx response that is not auth or not-found.
type APIRequestError struct {
	StatusCode int
	Message    string
}

func (e *APIRequestError) Error() string {
	return fmt.Sprintf("bitbucket API error (HTTP %d): %s", e.StatusCode, e.Message)
}

// Client is an HTTP client for the Bitbucket Cloud REST API v2.0.
type Client struct {
	httpClient *http.Client
	token      string
}

// NewClient creates a new Bitbucket API client with the given bearer token.
func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		token:      token,
	}
}

// doRequest executes an HTTP request and returns the response body.
// It maps HTTP status codes to typed errors.
func (c *Client) doRequest(ctx context.Context, method, reqURL string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, &AuthError{
			Message: "Authentication failed. Check that BITBUCKET_TOKEN is valid and has the required permissions.",
		}
	case resp.StatusCode == http.StatusNotFound:
		return nil, &NotFoundError{
			Message: "Resource not found. Verify the workspace, repo_slug, and pr_id values.",
		}
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		msg := string(respBody)
		var apiErr APIError
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error.Message != "" {
			msg = apiErr.Error.Message
		}
		return nil, &APIRequestError{StatusCode: resp.StatusCode, Message: msg}
	}

	return respBody, nil
}

// doRequestRaw executes an HTTP request and returns the response body as a string.
// Unlike doRequest, it does not set Accept: application/json (used for diff endpoints).
func (c *Client) doRequestRaw(ctx context.Context, method, reqURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, method, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return "", &AuthError{
			Message: "Authentication failed. Check that BITBUCKET_TOKEN is valid and has the required permissions.",
		}
	case resp.StatusCode == http.StatusNotFound:
		return "", &NotFoundError{
			Message: "Resource not found. Verify the workspace, repo_slug, and pr_id values.",
		}
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return "", &APIRequestError{StatusCode: resp.StatusCode, Message: string(respBody)}
	}

	return string(respBody), nil
}

// ListRecentPRs fetches open pull requests created within the last `hours` hours.
func (c *Client) ListRecentPRs(ctx context.Context, workspace, repoSlug string, hours int) ([]PullRequest, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	cutoffStr := cutoff.Format("2006-01-02T15:04:05.000000+00:00")

	params := url.Values{}
	params.Set("state", "OPEN")
	params.Set("pagelen", "50")
	params.Set("q", fmt.Sprintf(`created_on > "%s"`, cutoffStr))

	reqURL := fmt.Sprintf("%s/repositories/%s/%s/pullrequests?%s",
		baseURL, url.PathEscape(workspace), url.PathEscape(repoSlug), params.Encode())

	var allPRs []PullRequest
	for i := 0; i < 100; i++ { // safety cap
		body, err := c.doRequest(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}

		var page PaginatedResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		allPRs = append(allPRs, page.Values...)

		if page.Next == "" {
			break
		}
		reqURL = page.Next
	}

	return allPRs, nil
}

// GetPRDetails fetches the metadata for a specific pull request.
func (c *Client) GetPRDetails(ctx context.Context, workspace, repoSlug string, prID int) (*PullRequest, error) {
	reqURL := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d",
		baseURL, url.PathEscape(workspace), url.PathEscape(repoSlug), prID)

	body, err := c.doRequest(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	var pr PullRequest
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse pull request: %w", err)
	}

	return &pr, nil
}

// GetPRDiff fetches the unified diff for a specific pull request.
func (c *Client) GetPRDiff(ctx context.Context, workspace, repoSlug string, prID int) (string, error) {
	reqURL := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/diff",
		baseURL, url.PathEscape(workspace), url.PathEscape(repoSlug), prID)

	return c.doRequestRaw(ctx, http.MethodGet, reqURL)
}

// BranchExists checks if a branch exists in the repository.
func (c *Client) BranchExists(ctx context.Context, workspace, repoSlug, branchName string) (bool, string, error) {
	reqURL := fmt.Sprintf("%s/repositories/%s/%s/refs/branches/%s",
		baseURL, url.PathEscape(workspace), url.PathEscape(repoSlug), url.PathEscape(branchName))

	body, err := c.doRequest(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		var notFound *NotFoundError
		if errors.As(err, &notFound) {
			return false, "", nil
		}
		return false, "", err
	}

	var branch BranchResponse
	if err := json.Unmarshal(body, &branch); err != nil {
		return false, "", fmt.Errorf("failed to parse branch response: %w", err)
	}

	return true, branch.Target.Hash, nil
}

// CreateBranch creates a new branch from a source branch's HEAD.
func (c *Client) CreateBranch(ctx context.Context, workspace, repoSlug, branchName, fromBranch string) (*BranchResponse, error) {
	// First resolve the source branch's HEAD commit hash.
	exists, hash, err := c.BranchExists(ctx, workspace, repoSlug, fromBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve source branch %q: %w", fromBranch, err)
	}
	if !exists {
		return nil, &NotFoundError{Message: fmt.Sprintf("Source branch %q does not exist.", fromBranch)}
	}

	reqURL := fmt.Sprintf("%s/repositories/%s/%s/refs/branches",
		baseURL, url.PathEscape(workspace), url.PathEscape(repoSlug))

	payload := map[string]any{
		"name":   branchName,
		"target": map[string]string{"hash": hash},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal branch payload: %w", err)
	}

	body, err := c.doRequest(ctx, http.MethodPost, reqURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var branch BranchResponse
	if err := json.Unmarshal(body, &branch); err != nil {
		return nil, fmt.Errorf("failed to parse branch response: %w", err)
	}

	return &branch, nil
}

// CreatePR creates a new pull request on Bitbucket.
func (c *Client) CreatePR(ctx context.Context, workspace, repoSlug, title, description, sourceBranch, targetBranch string) (*CreatePRResponse, error) {
	reqURL := fmt.Sprintf("%s/repositories/%s/%s/pullrequests",
		baseURL, url.PathEscape(workspace), url.PathEscape(repoSlug))

	payload := map[string]any{
		"title":               title,
		"description":         description,
		"source":              map[string]any{"branch": map[string]string{"name": sourceBranch}},
		"destination":         map[string]any{"branch": map[string]string{"name": targetBranch}},
		"close_source_branch": true,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal PR payload: %w", err)
	}

	body, err := c.doRequest(ctx, http.MethodPost, reqURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var pr CreatePRResponse
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, fmt.Errorf("failed to parse PR response: %w", err)
	}

	return &pr, nil
}

// PostComment posts a top-level comment on a pull request.
func (c *Client) PostComment(ctx context.Context, workspace, repoSlug string, prID int, comment string) (*Comment, error) {
	reqURL := fmt.Sprintf("%s/repositories/%s/%s/pullrequests/%d/comments",
		baseURL, url.PathEscape(workspace), url.PathEscape(repoSlug), prID)

	payload := map[string]any{
		"content": map[string]string{
			"raw": comment,
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal comment: %w", err)
	}

	body, err := c.doRequest(ctx, http.MethodPost, reqURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	var result Comment
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse comment response: %w", err)
	}

	return &result, nil
}
