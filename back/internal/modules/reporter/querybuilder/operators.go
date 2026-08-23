package querybuilder

import (
	"fmt"

	"ehd-api/internal/modules/reporter/domain"
)

// Filter — уже проверенный на уровне сервиса фильтр (колонка из whitelist, оператор допустим).
type Filter struct {
	Column      string
	DisplayType string
	Operator    string
	Value       any
	Values      []any
}

// fragment строит SQL-условие с плейсхолдерами ? и возвращает соответствующие аргументы.
// Значения НИКОГДА не конкатенируются в текст — только параметры.
func (f Filter) fragment() (string, []any, error) {
	if !SafeIdent(f.Column) {
		return "", nil, ErrUnsafeIdentifier
	}
	col := quoteIdent(f.Column)

	switch f.Operator {
	case domain.OpEq:
		return col + " = ?", []any{coerce(f.Value, f.DisplayType)}, nil
	case domain.OpNeq:
		return col + " != ?", []any{coerce(f.Value, f.DisplayType)}, nil
	case domain.OpGt, domain.OpAfter:
		return col + " > ?", []any{coerce(f.Value, f.DisplayType)}, nil
	case domain.OpGte:
		return col + " >= ?", []any{coerce(f.Value, f.DisplayType)}, nil
	case domain.OpLt, domain.OpBefore:
		return col + " < ?", []any{coerce(f.Value, f.DisplayType)}, nil
	case domain.OpLte:
		return col + " <= ?", []any{coerce(f.Value, f.DisplayType)}, nil
	case domain.OpContains:
		return col + " ILIKE ?", []any{"%" + toString(f.Value) + "%"}, nil
	case domain.OpStartsWith:
		return "startsWith(" + col + ", ?)", []any{toString(f.Value)}, nil
	case domain.OpIn:
		if len(f.Values) == 0 {
			return "", nil, ErrBadFilter
		}
		return col + " IN ?", []any{coerceSlice(f.Values, f.DisplayType)}, nil
	case domain.OpBetween:
		if len(f.Values) != 2 {
			return "", nil, ErrBadFilter
		}
		return col + " BETWEEN ? AND ?", []any{coerce(f.Values[0], f.DisplayType), coerce(f.Values[1], f.DisplayType)}, nil
	case domain.OpIsNull:
		return col + " IS NULL", nil, nil
	case domain.OpIsNotNull:
		return col + " IS NOT NULL", nil, nil
	default:
		return "", nil, ErrBadFilter
	}
}

// coerce приводит JSON-значение к Go-типу под тип отображения (для корректного биндинга драйвером).
func coerce(v any, displayType string) any {
	switch displayType {
	case domain.DisplayNumber, domain.DisplayMoney, domain.DisplayPercent:
		return toFloat(v)
	case domain.DisplayBoolean:
		if b, ok := v.(bool); ok {
			return b
		}
		return v
	default: // text, enum, uuid, date, datetime — строковые литералы/параметры
		return toString(v)
	}
}

func coerceSlice(vals []any, displayType string) any {
	if displayType == domain.DisplayNumber || displayType == domain.DisplayMoney || displayType == domain.DisplayPercent {
		out := make([]float64, len(vals))
		for i, v := range vals {
			out[i] = toFloat(v)
		}
		return out
	}
	out := make([]string, len(vals))
	for i, v := range vals {
		out[i] = toString(v)
	}
	return out
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}
