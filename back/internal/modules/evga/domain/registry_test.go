package domain

import (
	"errors"
	"testing"
)

func TestSortExprWhitelist(t *testing.T) {
	cases := []struct {
		sort, order string
		want        string
		wantErr     bool
	}{
		{"", "", "t.paymentdate desc, t.id desc", false},
		{"paymentdate", "asc", "t.paymentdate asc, t.id asc", false},
		{"amount_part", "desc", "t.amount_part desc, t.id desc", false},
		{"god_mes", "desc", "t.god desc, t.mes desc, t.id desc", false},
		{"id", "", "t.id desc, t.id desc", false},
		// негативные (spec FR-7, INVALID_FILTER): поле вне whitelist / инъекция / направление
		{"unknown_field", "", "", true},
		{"paymentdate; drop table users", "desc", "", true},
		{"paymentdate", "desc; --", "", true},
	}
	for _, c := range cases {
		got, err := SortExpr(c.sort, c.order)
		if c.wantErr {
			if !errors.Is(err, ErrInvalidSort) {
				t.Errorf("SortExpr(%q,%q): want ErrInvalidSort, got %v", c.sort, c.order, err)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("SortExpr(%q,%q) = (%q, %v), want %q", c.sort, c.order, got, err, c.want)
		}
	}
}

func TestPageNormalize(t *testing.T) {
	p := Page{}.Normalize()
	if p.Page != 1 || p.Size != DefaultPageSize {
		t.Fatalf("empty page: got %+v", p)
	}
	p = Page{Page: -5, Size: 100500}.Normalize()
	if p.Page != 1 || p.Size != MaxPageSize {
		t.Fatalf("out of bounds: got %+v", p)
	}
	p = Page{Page: 3, Size: 50}.Normalize()
	if p.Page != 3 || p.Size != 50 {
		t.Fatalf("valid page changed: got %+v", p)
	}
}
