package httpadapter

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/karo/cuttlegate/internal/domain"
)

type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// WriteError writes a JSON error response derived from err.
// domain.ErrNotFound → 404, domain.ErrConflict → 409, domain.ErrForbidden → 403,
// errBadRequest → 400, all other errors → 500.
func WriteError(w http.ResponseWriter, err error) {
	var status int
	var code, message string

	var valErr *domain.ValidationError
	switch {
	case errors.As(err, &valErr):
		status, code, message = http.StatusBadRequest, "validation_error", err.Error()
	case errors.Is(err, domain.ErrNotFound):
		status, code, message = http.StatusNotFound, "not_found", "resource not found"
	case errors.Is(err, domain.ErrImmutableVariants):
		status, code, message = http.StatusBadRequest, "immutable_variants", err.Error()
	case errors.Is(err, domain.ErrLastVariant):
		status, code, message = http.StatusBadRequest, "last_variant", err.Error()
	case errors.Is(err, domain.ErrDefaultVariant):
		status, code, message = http.StatusConflict, "default_variant", err.Error()
	case errors.Is(err, domain.ErrLastAdmin):
		status, code, message = http.StatusConflict, "last_admin", "cannot remove the last admin"
	case errors.Is(err, domain.ErrConflict):
		status, code, message = http.StatusConflict, "conflict", "resource already exists"
	case errors.Is(err, domain.ErrForbidden):
		status, code, message = http.StatusForbidden, "forbidden", "access denied"
	case isBadRequest(err):
		status, code, message = http.StatusBadRequest, "bad_request", err.Error()
	default:
		status, code, message = http.StatusInternalServerError, "internal_error", "an unexpected error occurred"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorBody{Error: code, Message: message}) //nolint:errcheck
}

// writeUnauthorized writes a JSON 401 response. Used by auth middleware before
// any domain error exists, so WriteError cannot be used here.
func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(errorBody{Error: "unauthorized", Message: "authentication required"}) //nolint:errcheck
}

// badRequestError is a sentinel type for input validation errors.
type badRequestError struct{ msg string }

func (e *badRequestError) Error() string { return e.msg }

func newBadRequest(msg string) error { return &badRequestError{msg: msg} }

func isBadRequest(err error) bool {
	var e *badRequestError
	return errors.As(err, &e)
}
