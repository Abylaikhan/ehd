package querybuilder

import (
	"strings"
	"testing"

	"ehd-api/internal/modules/reporter/domain"
)

func basePlan() Plan {
	return Plan{
		Database:   "ehd_src",
		Table:      "demo",
		SelectCols: []string{"id", "full_name"},
		Keyset:     Keyset{Column: "id", Dir: "asc"},
		Limit:      51,
	}
}

func TestBuild_NoSelectStarBacktickIdents(t *testing.T) {
	sql, _, err := basePlan().Build()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sql, "*") {
		t.Errorf("SELECT * запрещён: %s", sql)
	}
	if !strings.Contains(sql, "SELECT `id`, `full_name` FROM `ehd_src`.`demo`") {
		t.Errorf("неожиданный SQL: %s", sql)
	}
	if !strings.Contains(sql, "ORDER BY `id` ASC") || !strings.Contains(sql, "LIMIT 51") {
		t.Errorf("нет order/limit: %s", sql)
	}
}

func TestBuild_FilterValueIsParameterizedNotConcatenated(t *testing.T) {
	p := basePlan()
	inj := "'; DROP TABLE x; --"
	p.Filters = []Filter{{Column: "full_name", DisplayType: domain.DisplayText, Operator: domain.OpEq, Value: inj}}
	sql, args, err := p.Build()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sql, "DROP") {
		t.Errorf("значение попало в текст SQL (инъекция): %s", sql)
	}
	if !strings.Contains(sql, "`full_name` = ?") {
		t.Errorf("нет плейсхолдера параметра: %s", sql)
	}
	if len(args) != 1 || args[0] != inj {
		t.Errorf("значение не передано параметром: %v", args)
	}
}

func TestBuild_UnsafeIdentifierRejected(t *testing.T) {
	p := basePlan()
	p.SelectCols = []string{"id", "a; DROP"}
	if _, _, err := p.Build(); err != ErrUnsafeIdentifier {
		t.Fatalf("ожидалась ErrUnsafeIdentifier, получено %v", err)
	}
}

func TestBuild_KeysetPredicateAscDesc(t *testing.T) {
	p := basePlan()
	p.Keyset.Cursor = uint64(100)
	sql, _, _ := p.Build()
	if !strings.Contains(sql, "`id` > ?") {
		t.Errorf("нет keyset-предиката ASC: %s", sql)
	}
	p.Keyset.Dir = "desc"
	sql, _, _ = p.Build()
	if !strings.Contains(sql, "`id` < ?") || !strings.Contains(sql, "ORDER BY `id` DESC") {
		t.Errorf("keyset DESC неверен: %s", sql)
	}
}

func TestBuild_RowScopeVariants(t *testing.T) {
	// оба измерения
	p := basePlan()
	p.RowScope = RowScope{RegionColumn: "region_code", Regions: []string{"01"}, DepartmentColumn: "department_code", Departments: []string{"D1"}}
	sql, args, _ := p.Build()
	if !strings.Contains(sql, "`region_code` IN ? AND `region_code` IS NOT NULL") {
		t.Errorf("нет RLS по региону: %s", sql)
	}
	if !strings.Contains(sql, "`department_code` IN ? AND `department_code` IS NOT NULL") {
		t.Errorf("нет RLS по подразделению: %s", sql)
	}
	if len(args) != 2 {
		t.Errorf("ожидалось 2 массива-параметра, получено %v", args)
	}
	// только регион
	p2 := basePlan()
	p2.RowScope = RowScope{RegionColumn: "region_code", Regions: []string{"01"}}
	sql2, _, _ := p2.Build()
	if strings.Contains(sql2, "department") {
		t.Errorf("не должно быть предиката по подразделению: %s", sql2)
	}
	// пустой профиль → нет предикатов (fail-open)
	sql3, _, _ := basePlan().Build()
	if strings.Contains(sql3, "IS NOT NULL") {
		t.Errorf("при пустом профиле RLS-предикатов быть не должно: %s", sql3)
	}
}

func TestBuildCount_NoOrderNoLimit(t *testing.T) {
	p := basePlan()
	p.Filters = []Filter{{Column: "full_name", DisplayType: domain.DisplayText, Operator: domain.OpContains, Value: "иван"}}
	sql, args, err := p.BuildCount()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(sql, "SELECT count() FROM `ehd_src`.`demo`") {
		t.Errorf("неверный COUNT SQL: %s", sql)
	}
	if strings.Contains(sql, "ORDER BY") || strings.Contains(sql, "LIMIT") {
		t.Errorf("COUNT не должен содержать ORDER/LIMIT: %s", sql)
	}
	if !strings.Contains(sql, "`full_name` ILIKE ?") || len(args) != 1 || args[0] != "%иван%" {
		t.Errorf("фильтр в COUNT неверен: sql=%s args=%v", sql, args)
	}
}

func TestSafeIdent(t *testing.T) {
	for _, ok := range []string{"id", "full_name", "_x", "a1"} {
		if !SafeIdent(ok) {
			t.Errorf("%q должен быть безопасным", ok)
		}
	}
	for _, bad := range []string{"", "a b", "a;b", "a-b", "a.b", "1a", "région"} {
		if SafeIdent(bad) {
			t.Errorf("%q должен быть отклонён", bad)
		}
	}
}
