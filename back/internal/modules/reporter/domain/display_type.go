package domain

import "strings"

// Типы отображения колонки (ТЗ, «Соответствие типов и фильтров»).
const (
	DisplayText     = "text"
	DisplayNumber   = "number"
	DisplayMoney    = "money"
	DisplayPercent  = "percent"
	DisplayDate     = "date"
	DisplayDateTime = "datetime"
	DisplayBoolean  = "boolean"
	DisplayEnum     = "enum"
	DisplayJSON     = "json"
	DisplayUUID     = "uuid"
)

// DisplayTypeFor выводит display_type из физического типа ClickHouse.
// Nullable(T) и LowCardinality(T) разворачиваются во внутренний тип.
func DisplayTypeFor(chType string) string {
	t := strings.TrimSpace(chType)
	if inner, ok := unwrap(t, "Nullable"); ok {
		return DisplayTypeFor(inner)
	}
	if inner, ok := unwrap(t, "LowCardinality"); ok {
		return DisplayTypeFor(inner)
	}

	base := baseType(t)
	switch {
	case base == "String" || base == "FixedString":
		return DisplayText
	case base == "UUID":
		return DisplayUUID
	case base == "Bool":
		return DisplayBoolean
	case strings.HasPrefix(base, "Enum"):
		return DisplayEnum
	case strings.HasPrefix(base, "Int") || strings.HasPrefix(base, "UInt") ||
		strings.HasPrefix(base, "Float") || base == "Decimal":
		return DisplayNumber
	case base == "Date" || base == "Date32":
		return DisplayDate
	case base == "DateTime" || base == "DateTime64":
		return DisplayDateTime
	case base == "Array" || base == "Map" || base == "Tuple" || base == "JSON" || base == "Object":
		return DisplayJSON
	default:
		return DisplayText
	}
}

// unwrap возвращает внутренний тип обёртки Wrapper(T), если тип ею обёрнут.
func unwrap(t, wrapper string) (string, bool) {
	p := wrapper + "("
	if strings.HasPrefix(t, p) && strings.HasSuffix(t, ")") {
		return t[len(p) : len(t)-1], true
	}
	return "", false
}

// baseType — имя типа до первой круглой скобки (Decimal(18,2) -> Decimal).
func baseType(t string) string {
	if i := strings.IndexByte(t, '('); i >= 0 {
		return t[:i]
	}
	return t
}
