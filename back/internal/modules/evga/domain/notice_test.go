package domain

import (
	"errors"
	"testing"
)

// Спека 024 FR-2: список адресатов при замене на доработке.
func TestValidateRecipientIDs(t *testing.T) {
	cases := []struct {
		name string
		ids  []int64
		ok   bool
	}{
		{"один адресат", []int64{7}, true},
		{"несколько в своём порядке", []int64{33, 7, 12}, true},
		{"пустой список — уведомление без адресата недопустимо", nil, false},
		{"пустой срез", []int64{}, false},
		{"дубликат адресата", []int64{7, 12, 7}, false},
		{"нулевой id", []int64{7, 0}, false},
		{"отрицательный id", []int64{-1}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateRecipientIDs(c.ids)
			if c.ok && err != nil {
				t.Fatalf("ожидался успех, получено: %v", err)
			}
			if !c.ok {
				if err == nil {
					t.Fatal("ожидалась ошибка")
				}
				if !errors.Is(err, ErrInvalidRecipients) {
					t.Fatalf("ожидалась ErrInvalidRecipients, получено: %v", err)
				}
			}
		})
	}
}
