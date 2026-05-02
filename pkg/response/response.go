package response

import (
	"encoding/json"
	"errors"
	"net/http"

	domainerrors "github.com/KryptoStorage/ms-storage/internal/domain/errors"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type errorEnvelope struct {
	Error ErrorBody `json:"error"`
}

// JSON writes the payload as JSON. The encode error is intentionally swallowed
// because the response has already been committed; if the connection drops
// mid-write, the access log will record the truncated byte count.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// Error writes a normalised error payload. If err is a domain *Error, its
// Kind drives the HTTP status; otherwise we default to 500.
func Error(w http.ResponseWriter, err error) {
	body, status := translate(err)
	JSON(w, status, errorEnvelope{Error: body})
}

// ErrorWith forces a specific HTTP status (use for hand-crafted responses).
func ErrorWith(w http.ResponseWriter, status int, code, message string, details any) {
	JSON(w, status, errorEnvelope{Error: ErrorBody{Code: code, Message: message, Details: details}})
}

func translate(err error) (ErrorBody, int) {
	var de *domainerrors.Error
	if !errors.As(err, &de) {
		return ErrorBody{Code: "internal_error", Message: "internal error"}, http.StatusInternalServerError
	}

	body := ErrorBody{Code: de.Code, Message: de.Message}
	if body.Code == "" {
		body.Code = defaultCode(de.Kind)
	}

	switch de.Kind {
	case domainerrors.KindValidation:
		return body, http.StatusBadRequest
	case domainerrors.KindNotFound:
		return body, http.StatusNotFound
	case domainerrors.KindConflict:
		return body, http.StatusConflict
	case domainerrors.KindUnauthorized:
		return body, http.StatusUnauthorized
	case domainerrors.KindForbidden:
		return body, http.StatusForbidden
	case domainerrors.KindUnavailable:
		return body, http.StatusServiceUnavailable
	default:
		return body, http.StatusInternalServerError
	}
}

func defaultCode(k domainerrors.Kind) string {
	switch k {
	case domainerrors.KindValidation:
		return "validation_error"
	case domainerrors.KindNotFound:
		return "not_found"
	case domainerrors.KindConflict:
		return "conflict"
	case domainerrors.KindUnauthorized:
		return "unauthorized"
	case domainerrors.KindForbidden:
		return "forbidden"
	case domainerrors.KindUnavailable:
		return "unavailable"
	default:
		return "internal_error"
	}
}
