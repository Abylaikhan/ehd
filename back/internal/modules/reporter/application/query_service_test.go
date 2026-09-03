package application

import (
	"testing"

	"ehd-api/internal/modules/reporter/domain"
)

func snapWith(cols []domain.SnapshotColumn) domain.PublishedSnapshot {
	return domain.PublishedSnapshot{
		DatabaseName: "ehd_src", TableName: "demo",
		PageSizeDefault: 50, PageSizeMin: 20, PageSizeMax: 200,
		RowScopeMode: domain.RowScopeByProfile,
		KeysetColumn: "id", KeysetType: "UInt64", KeysetDir: "asc",
		RowScopeRegionColumn: "region_code", RowScopeDepartmentColumn: "department_code",
		Columns: cols,
	}
}

func textCol(name string, filterable bool) domain.SnapshotColumn {
	return domain.SnapshotColumn{SourceName: name, SourceType: "String", DisplayType: domain.DisplayText, Filterable: filterable}
}

func TestBuildPlan_RejectsNonWhitelistColumn(t *testing.T) {
	snap := snapWith([]domain.SnapshotColumn{textCol("full_name", true)})
	spec := domain.QuerySpec{Filters: []domain.Filter{{Column: "secret", Operator: domain.OpEq, Value: "x"}}}
	if _, _, err := buildPlan(snap, spec, Requester{IsAdmin: true}, false); err != domain.ErrQueryValidation {
		t.Fatalf("колонка вне whitelist → ожидалась ErrQueryValidation, получено %v", err)
	}
}

func TestBuildPlan_RejectsNonFilterableColumn(t *testing.T) {
	snap := snapWith([]domain.SnapshotColumn{textCol("full_name", false)})
	spec := domain.QuerySpec{Filters: []domain.Filter{{Column: "full_name", Operator: domain.OpEq, Value: "x"}}}
	if _, _, err := buildPlan(snap, spec, Requester{IsAdmin: true}, false); err != domain.ErrQueryValidation {
		t.Fatalf("не-filterable колонка → ожидалась ErrQueryValidation, получено %v", err)
	}
}

func TestBuildPlan_RejectsIncompatibleOperator(t *testing.T) {
	snap := snapWith([]domain.SnapshotColumn{textCol("full_name", true)})
	// gt недопустим для текста
	spec := domain.QuerySpec{Filters: []domain.Filter{{Column: "full_name", Operator: domain.OpGt, Value: "x"}}}
	if _, _, err := buildPlan(snap, spec, Requester{IsAdmin: true}, false); err != domain.ErrQueryValidation {
		t.Fatalf("несовместимый оператор → ожидалась ErrQueryValidation, получено %v", err)
	}
}

func TestBuildPlan_RLSAppliedForUserNotAdmin(t *testing.T) {
	snap := snapWith([]domain.SnapshotColumn{textCol("full_name", true)})
	user := Requester{RoleCodes: []string{"analyst"}, RegionCodes: []string{"01", "02"}, DepartmentCodes: []string{"D1"}}
	plan, _, err := buildPlan(snap, domain.QuerySpec{}, user, false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.RowScope.RegionColumn != "region_code" || len(plan.RowScope.Regions) != 2 {
		t.Errorf("RLS по региону не применён: %+v", plan.RowScope)
	}
	if plan.RowScope.DepartmentColumn != "department_code" || len(plan.RowScope.Departments) != 1 {
		t.Errorf("RLS по подразделению не применён: %+v", plan.RowScope)
	}
	// админ → без RLS
	planAdmin, _, _ := buildPlan(snap, domain.QuerySpec{}, Requester{IsAdmin: true}, false)
	if planAdmin.RowScope.RegionColumn != "" || len(planAdmin.RowScope.Regions) != 0 {
		t.Errorf("админ не должен получать RLS: %+v", planAdmin.RowScope)
	}
}

func TestBuildPlan_KeysetHiddenColumnAddedToSelect(t *testing.T) {
	// snapshot видимых колонок НЕ содержит id (keyset), должен быть добавлен в SELECT
	snap := snapWith([]domain.SnapshotColumn{textCol("full_name", true)})
	plan, pm, err := buildPlan(snap, domain.QuerySpec{}, Requester{IsAdmin: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range plan.SelectCols {
		if c == "id" {
			found = true
		}
	}
	if !found {
		t.Errorf("keyset-колонка id должна быть в SELECT: %v", plan.SelectCols)
	}
	// но в видимом порядке её нет (не попадёт в ответ)
	for _, name := range pm.visibleOrder {
		if name == "id" {
			t.Errorf("keyset id не должен быть в visibleOrder: %v", pm.visibleOrder)
		}
	}
}

func TestBuildPlan_SortByColumn(t *testing.T) {
	snap := snapWith([]domain.SnapshotColumn{
		{SourceName: "id", SourceType: "UInt64", DisplayType: domain.DisplayNumber},
		{SourceName: "name", SourceType: "String", DisplayType: domain.DisplayText, Sortable: true},
	})
	// сортировка по sortable-колонке → она первой, keyset id — тай-брейкером
	plan, _, err := buildPlan(snap, domain.QuerySpec{SortColumn: "name", SortDir: "desc"}, Requester{IsAdmin: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Keyset.Columns) < 2 || plan.Keyset.Columns[0] != "name" || plan.Keyset.Columns[1] != "id" {
		t.Errorf("порядок должен быть [name, id]: %v", plan.Keyset.Columns)
	}
	if plan.Keyset.Dir != "desc" {
		t.Errorf("направление сортировки: %q", plan.Keyset.Dir)
	}
	// сортировка по не-sortable / неизвестной колонке → ошибка (fail-closed)
	if _, _, err := buildPlan(snap, domain.QuerySpec{SortColumn: "id"}, Requester{IsAdmin: true}, false); err != domain.ErrQueryValidation {
		t.Fatalf("не-sortable → ожидалась ErrQueryValidation, получено %v", err)
	}
	if _, _, err := buildPlan(snap, domain.QuerySpec{SortColumn: "nope"}, Requester{IsAdmin: true}, false); err != domain.ErrQueryValidation {
		t.Fatalf("неизвестная колонка → ожидалась ErrQueryValidation, получено %v", err)
	}
}

func TestClampPageSize(t *testing.T) {
	cases := []struct{ in, want int }{{0, 50}, {5, 20}, {50, 50}, {500, 200}, {150, 150}}
	for _, c := range cases {
		if got := clampPageSize(c.in, 20, 50, 200); got != c.want {
			t.Errorf("clampPageSize(%d)=%d, want %d", c.in, got, c.want)
		}
	}
}

func TestBuildResult_StripsHiddenAndComputesNextCursor(t *testing.T) {
	snap := snapWith([]domain.SnapshotColumn{{SourceName: "full_name", Label: "ФИО", DisplayType: domain.DisplayText}})
	rows := []map[string]any{
		{"full_name": "A", "id": uint64(1), "secret": "s1"},
		{"full_name": "B", "id": uint64(2), "secret": "s2"},
		{"full_name": "C", "id": uint64(3), "secret": "s3"}, // лишняя (pageSize+1)
	}
	pm := planMeta{keysetColumns: []string{"id"}, pageSize: 2, visibleOrder: []string{"full_name"}}
	res := buildResult(snap, rows, pm)

	if len(res.Rows) != 2 {
		t.Fatalf("ожидалось 2 строки (page_size), получено %d", len(res.Rows))
	}
	if res.NextCursor != encodeCursor([]string{"2"}) {
		t.Errorf("next_cursor должен кодировать [2], получено %q", res.NextCursor)
	}
	for _, r := range res.Rows {
		if _, ok := r["secret"]; ok {
			t.Error("скрытая колонка secret попала в ответ")
		}
		if _, ok := r["id"]; ok {
			t.Error("keyset id (не видим) попал в ответ")
		}
		if _, ok := r["full_name"]; !ok {
			t.Error("видимая колонка full_name отсутствует")
		}
	}
}
