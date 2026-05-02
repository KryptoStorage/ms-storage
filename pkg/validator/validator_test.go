package validator

import (
	"errors"
	"testing"
)

func TestRequired(t *testing.T) {
	v := New().Required("name", "  ")
	if err := v.Validate(); err == nil || !errors.Is(err.(Errors)[0].Err, ErrRequired) {
		t.Fatalf("expected ErrRequired, got %v", err)
	}

	if err := New().Required("name", "alice").Validate(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestLengthBounds(t *testing.T) {
	tests := []struct {
		name    string
		run     func() *Validator
		wantErr error
	}{
		{"too short", func() *Validator { return New().MinLength("x", "ab", 3) }, ErrMinLength},
		{"min ok", func() *Validator { return New().MinLength("x", "abc", 3) }, nil},
		{"too long", func() *Validator { return New().MaxLength("x", "abcd", 3) }, ErrMaxLength},
		{"max ok", func() *Validator { return New().MaxLength("x", "abc", 3) }, nil},
		{"unicode counted by rune", func() *Validator { return New().MinLength("x", "héllo", 5) }, nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run().Validate()
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected nil, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err.(Errors)[0].Err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestEmailAndRange(t *testing.T) {
	v := New().Email("e", "not-an-email").Range("n", 5, 10, 20)
	err := v.Validate()
	if err == nil {
		t.Fatal("expected errors")
	}
	if len(err.(Errors)) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(err.(Errors)))
	}

	if err := New().Email("e", "a@b.co").Range("n", 15, 10, 20).Validate(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
