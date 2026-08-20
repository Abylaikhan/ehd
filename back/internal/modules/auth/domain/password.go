package domain

import "unicode"

// MinPasswordLen — минимальная длина пароля по ТЗ.
const MinPasswordLen = 8

// ValidatePassword: ≥8 символов, ≥1 заглавная, ≥1 строчная, ≥1 цифра (ТЗ, «Регистрация…»).
func ValidatePassword(pw string) error {
	if len(pw) < MinPasswordLen {
		return ErrWeakPassword
	}
	var upper, lower, digit bool
	for _, r := range pw {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		}
	}
	if !upper || !lower || !digit {
		return ErrWeakPassword
	}
	return nil
}
