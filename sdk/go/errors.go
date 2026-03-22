package cuttlegate

import "fmt"

// AuthError is returned when the server responds with a 401 or 403 status.
type AuthError struct {
	StatusCode int
	Message    string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("cuttlegate: auth error %d: %s", e.StatusCode, e.Message)
}

// NotFoundError is returned when the requested resource does not exist.
// Resource is one of "flag" or "project".
type NotFoundError struct {
	Resource string
	Key      string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("cuttlegate: %s %q not found", e.Resource, e.Key)
}

// ServerError is returned when the server responds with an unexpected 5xx status.
type ServerError struct {
	StatusCode int
	Message    string
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("cuttlegate: server error %d: %s", e.StatusCode, e.Message)
}
