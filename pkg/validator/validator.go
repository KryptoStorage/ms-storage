package validator

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	ErrRequired      = errors.New("field is required")
	ErrMinLength     = errors.New("field is shorter than the minimum length")
	ErrMaxLength     = errors.New("field is longer than the maximum length")
	ErrInvalidFormat = errors.New("field has an invalid format")
	ErrOutOfRange    = errors.New("field is out of the allowed range")
)

var emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type FieldError struct {
	Field string
	Err   error
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Err.Error())
}

func (e *FieldError) Unwrap() error { return e.Err }

type Errors []*FieldError

func (es Errors) Error() string {
	parts := make([]string, 0, len(es))
	for _, e := range es {
		parts = append(parts, e.Error())
	}
	return strings.Join(parts, "; ")
}

type Validator struct {
	errs Errors
}

func New() *Validator { return &Validator{} }

func (v *Validator) add(field string, err error) {
	v.errs = append(v.errs, &FieldError{Field: field, Err: err})
}

func (v *Validator) Required(field, value string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.add(field, ErrRequired)
	}
	return v
}

func (v *Validator) MinLength(field, value string, min int) *Validator {
	if utf8.RuneCountInString(value) < min {
		v.add(field, fmt.Errorf("%w (min %d)", ErrMinLength, min))
	}
	return v
}

func (v *Validator) MaxLength(field, value string, max int) *Validator {
	if utf8.RuneCountInString(value) > max {
		v.add(field, fmt.Errorf("%w (max %d)", ErrMaxLength, max))
	}
	return v
}

func (v *Validator) Email(field, value string) *Validator {
	if !emailRegexp.MatchString(value) {
		v.add(field, ErrInvalidFormat)
	}
	return v
}

func (v *Validator) Range(field string, value, min, max int) *Validator {
	if value < min || value > max {
		v.add(field, fmt.Errorf("%w [%d, %d]", ErrOutOfRange, min, max))
	}
	return v
}

func (v *Validator) Valid() bool { return len(v.errs) == 0 }

func (v *Validator) Errors() Errors { return v.errs }

func (v *Validator) Validate() error {
	if v.Valid() {
		return nil
	}
	return v.errs
}
