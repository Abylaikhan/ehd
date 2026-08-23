package domain

// Канонические операторы фильтрации (ТЗ, «Соответствие типов и фильтров»).
const (
	OpEq         = "eq"
	OpNeq        = "neq"
	OpContains   = "contains"
	OpStartsWith = "starts_with"
	OpIn         = "in"
	OpGt         = "gt"
	OpGte        = "gte"
	OpLt         = "lt"
	OpLte        = "lte"
	OpBetween    = "between"
	OpBefore     = "before"
	OpAfter      = "after"
	OpIsNull     = "is_null"
	OpIsNotNull  = "is_not_null"
)

// AllowedOperators — серверная таблица совместимости display_type → операторы (Принцип 3).
func AllowedOperators(displayType string) []string {
	switch displayType {
	case DisplayText:
		return []string{OpEq, OpNeq, OpContains, OpStartsWith, OpIn, OpIsNull, OpIsNotNull}
	case DisplayNumber, DisplayMoney, DisplayPercent:
		return []string{OpEq, OpNeq, OpGt, OpGte, OpLt, OpLte, OpBetween, OpIn, OpIsNull, OpIsNotNull}
	case DisplayDate, DisplayDateTime:
		return []string{OpEq, OpBefore, OpAfter, OpBetween, OpIsNull, OpIsNotNull}
	case DisplayBoolean:
		return []string{OpEq, OpIsNull, OpIsNotNull}
	case DisplayEnum:
		return []string{OpEq, OpIn, OpIsNull, OpIsNotNull}
	case DisplayUUID:
		return []string{OpEq, OpIn, OpIsNull, OpIsNotNull}
	default: // json и прочее — фильтрация не поддерживается
		return nil
	}
}

// OperatorAllowed сообщает, допустим ли оператор для display_type.
func OperatorAllowed(displayType, op string) bool {
	for _, o := range AllowedOperators(displayType) {
		if o == op {
			return true
		}
	}
	return false
}

// OperatorNeedsValue — требует ли оператор одиночное значение.
func OperatorNeedsValue(op string) bool {
	switch op {
	case OpIsNull, OpIsNotNull, OpIn, OpBetween:
		return false
	default:
		return true
	}
}

// Filter — один фильтр QuerySpec.
type Filter struct {
	Column   string
	Operator string
	Value    any
	Values   []any
}

// QuerySpec — единственный контракт, принимаемый /query (SQL-текст запрещён).
type QuerySpec struct {
	Filters  []Filter
	Search   string
	SortDir  string
	PageSize int
	Cursor   string
}

// ResultColumn — описание колонки в ответе данных.
type ResultColumn struct {
	SourceName  string
	Label       string
	DisplayType string
}

// QueryResult — страница данных.
type QueryResult struct {
	Columns    []ResultColumn
	Rows       []map[string]any
	NextCursor string
	PageSize   int
}
