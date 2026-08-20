package domain

import (
	"errors"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	cases := []struct {
		name string
		pw   string
		ok   bool
	}{
		{"valid", "Passw0rd", true},
		{"too short", "Pa0ss", false},
		{"no upper", "passw0rd", false},
		{"no lower", "PASSW0RD", false},
		{"no digit", "Password", false},
		{"exactly 8 valid", "Aa000000", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePassword(tc.pw)
			if tc.ok && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
			if !tc.ok && !errors.Is(err, ErrWeakPassword) {
				t.Fatalf("expected ErrWeakPassword, got %v", err)
			}
		})
	}
}

func TestValidateIIN(t *testing.T) {
	if err := ValidateIIN("990101300123"); err != nil {
		t.Fatalf("valid IIN rejected: %v", err)
	}
	for _, bad := range []string{"", "12345", "99010130012a", "9901013001234"} {
		if err := ValidateIIN(bad); !errors.Is(err, ErrInvalidIIN) {
			t.Fatalf("IIN %q: expected ErrInvalidIIN, got %v", bad, err)
		}
	}
}
