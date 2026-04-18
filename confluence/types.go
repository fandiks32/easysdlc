package confluence

// PageResponse represents a Confluence REST API content response.
type PageResponse struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Title   string    `json:"title"`
	Body    PageBody  `json:"body"`
	Version Version   `json:"version"`
	Links   PageLinks `json:"_links"`
	Space   Space     `json:"space"`
}

// PageBody contains the page content in various representations.
type PageBody struct {
	Storage StorageBody `json:"storage"`
}

// StorageBody holds the XHTML storage format content.
type StorageBody struct {
	Value          string `json:"value"`
	Representation string `json:"representation"`
}

// Version holds page version metadata.
type Version struct {
	Number int  `json:"number"`
	By     User `json:"by"`
}

// User represents a Confluence user.
type User struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
}

// PageLinks holds hypermedia links for a page.
type PageLinks struct {
	WebUI string `json:"webui"`
	Base  string `json:"base"`
}

// Space represents a Confluence space.
type Space struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// AuthError indicates authentication failure against Confluence.
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}

// NotFoundError indicates a 404 from Confluence.
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}

// APIRequestError indicates a non-2xx response from Confluence.
type APIRequestError struct {
	StatusCode int
	Message    string
}

func (e *APIRequestError) Error() string {
	return e.Message
}
