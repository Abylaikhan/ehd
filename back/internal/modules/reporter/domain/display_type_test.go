package domain

import "testing"

func TestDisplayTypeFor(t *testing.T) {
	cases := map[string]string{
		"String":                           DisplayText,
		"FixedString(16)":                  DisplayText,
		"LowCardinality(String)":           DisplayText,
		"UInt64":                           DisplayNumber,
		"Int32":                            DisplayNumber,
		"Float64":                          DisplayNumber,
		"Decimal(18, 2)":                   DisplayNumber,
		"Date":                             DisplayDate,
		"Date32":                           DisplayDate,
		"DateTime":                         DisplayDateTime,
		"DateTime64(3)":                    DisplayDateTime,
		"Bool":                             DisplayBoolean,
		"Enum8('a' = 1)":                   DisplayEnum,
		"UUID":                             DisplayUUID,
		"Array(String)":                    DisplayJSON,
		"Map(String, UInt64)":              DisplayJSON,
		"Tuple(UInt8, String)":             DisplayJSON,
		"JSON":                             DisplayJSON,
		"Nullable(Int64)":                  DisplayNumber,
		"Nullable(String)":                 DisplayText,
		"LowCardinality(Nullable(String))": DisplayText,
	}
	for in, want := range cases {
		if got := DisplayTypeFor(in); got != want {
			t.Errorf("DisplayTypeFor(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidSlug(t *testing.T) {
	valid := []string{"demo", "demo-2", "a1-b2-c3", "reports"}
	invalid := []string{"", "Demo", "demo_2", "демо", "with space", "UPPER", "a/b"}
	for _, s := range valid {
		if !ValidSlug(s) {
			t.Errorf("ValidSlug(%q) = false, want true", s)
		}
	}
	for _, s := range invalid {
		if ValidSlug(s) {
			t.Errorf("ValidSlug(%q) = true, want false", s)
		}
	}
}
