package errors

import "errors"

type Kind int

const (
	KindUnknown Kind = iota
	KindValidation
	KindNotFound
	KindConflict
	KindUnauthorized
	KindForbidden
	KindUnavailable
	KindInternal
)

type Error struct {
	Kind    Kind
	Code    string
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Cause }

func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

func Wrap(kind Kind, code, message string, cause error) *Error {
	return &Error{Kind: kind, Code: code, Message: message, Cause: cause}
}

func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return KindUnknown
}

// Is allows errors.Is comparison by Kind via sentinels below.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Kind == t.Kind && (t.Code == "" || t.Code == e.Code)
}

var (
	ErrValidation   = &Error{Kind: KindValidation, Message: "validation error"}
	ErrNotFound     = &Error{Kind: KindNotFound, Message: "not found"}
	ErrConflict     = &Error{Kind: KindConflict, Message: "conflict"}
	ErrUnauthorized = &Error{Kind: KindUnauthorized, Message: "unauthorized"}
	ErrForbidden    = &Error{Kind: KindForbidden, Message: "forbidden"}
	ErrUnavailable  = &Error{Kind: KindUnavailable, Message: "service unavailable"}
	ErrInternal     = &Error{Kind: KindInternal, Message: "internal error"}
)
