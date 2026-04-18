package bitbucket

// PaginatedResponse represents a paginated Bitbucket API response.
type PaginatedResponse struct {
	Values []PullRequest `json:"values"`
	Next   string        `json:"next"`
	Page   int           `json:"page"`
	Size   int           `json:"size"`
}

// PullRequest represents a Bitbucket pull request.
type PullRequest struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	State       string    `json:"state"`
	CreatedOn   string    `json:"created_on"`
	UpdatedOn   string    `json:"updated_on"`
	Author      User      `json:"author"`
	Source      BranchRef `json:"source"`
	Destination BranchRef `json:"destination"`
	Links       Links     `json:"links"`
}

// User represents a Bitbucket user.
type User struct {
	DisplayName string `json:"display_name"`
	UUID        string `json:"uuid"`
	Nickname    string `json:"nickname"`
}

// BranchRef represents a branch reference in a pull request.
type BranchRef struct {
	Branch     Branch  `json:"branch"`
	Repository RepoRef `json:"repository"`
}

// Branch represents a git branch.
type Branch struct {
	Name string `json:"name"`
}

// RepoRef represents a repository reference.
type RepoRef struct {
	FullName string `json:"full_name"`
}

// Links contains hypermedia links for a resource.
type Links struct {
	HTML Link `json:"html"`
}

// Link represents a single hypermedia link.
type Link struct {
	Href string `json:"href"`
}

// Comment represents a pull request comment.
type Comment struct {
	ID      int            `json:"id"`
	Content CommentContent `json:"content"`
}

// CommentContent holds the raw content of a comment.
type CommentContent struct {
	Raw string `json:"raw"`
}

// BranchResponse represents a Bitbucket branch ref.
type BranchResponse struct {
	Name   string       `json:"name"`
	Target BranchTarget `json:"target"`
}

// BranchTarget holds the commit hash a branch points to.
type BranchTarget struct {
	Hash string `json:"hash"`
}

// CreatePRResponse is the response after creating a pull request.
type CreatePRResponse struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	State string `json:"state"`
	Links Links  `json:"links"`
}

// APIError represents an error response from the Bitbucket API.
type APIError struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains the error message and detail.
type ErrorDetail struct {
	Message string `json:"message"`
	Detail  string `json:"detail"`
}
