package bacon

import "fmt"

// Error represents a structured error response from the Feature Bacon API,
// following RFC 7807 problem details.
type Error struct {
	StatusCode int    `json:"-"`
	Type       string `json:"type"`
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	Instance   string `json:"instance"`
}

func (e *Error) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("bacon: %s (%d): %s", e.Title, e.StatusCode, e.Detail)
	}
	return fmt.Sprintf("bacon: %s (%d)", e.Title, e.StatusCode)
}
