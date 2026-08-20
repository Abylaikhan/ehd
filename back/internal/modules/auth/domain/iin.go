package domain

// ValidateIIN — ИИН РК: ровно 12 цифр.
func ValidateIIN(iin string) error {
	if len(iin) != 12 {
		return ErrInvalidIIN
	}
	for _, r := range iin {
		if r < '0' || r > '9' {
			return ErrInvalidIIN
		}
	}
	return nil
}
