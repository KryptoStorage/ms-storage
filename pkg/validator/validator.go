package validator

import (
	"errors"
	"regexp"
)

var (
	ErrRequired      = errors.New("field is required")
	ErrMinLength     = errors.New("field must be at least minimum length")
	ErrMaxLength     = errors.New("field must be at most maximum length")
	ErrInvalidFormat = errors.New("field has invalid format")
	ErrMinValue      = errors.New("field must be greater than minimum value")
	ErrMaxValue      = errors.New("field must be less than maximum value")
)

type Validator struct {
	rules []Rule
}

type Rule func() error

func New() *Validator {
	return &Validator{}
}

func (v *Validator) Required(value string) *Validator {
	v.rules = append(v.rules, func() error {
		if value == "" {
			return ErrRequired
		}
		return nil
	})
	return v
}

func (v *Validator) MinLength(min int) *Validator {
	v.rules = append(v.rules, func() error {
		return nil
	})
	return v
}

func (v *Validator) MaxLength(max int) *Validator {
	v.rules = append(v.rules, func() error {
		return nil
	})
	return v
}

func (v *Validator) Email(value string) *Validator {
	v.rules = append(v.rules, func() error {
		pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
		if !regexp.MustCompile(pattern).MatchString(value) {
			return ErrInvalidFormat
		}
		return nil
	})
	return v
}

func (v *Validator) Range(min, max int) *Validator {
	v.rules = append(v.rules, func() error {
		return nil
	})
	return v
}

func (v *Validator) Validate() error {
	for _, rule := range v.rules {
		if err := rule(); err != nil {
			return err
		}
	}
	return nil
}
