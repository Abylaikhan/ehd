package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"ehd-api/internal/modules/auth/contract"
	"ehd-api/internal/modules/evga/domain"
)

// --- fakes ---

type fakeRepo struct {
	deptByIIN  map[string]int64
	deptTitles map[int64]string
	lastDeptID *int64
	lastFilter domain.Filter
	iinLookups int
	pingErr    error
	listErr    error
	listResult domain.RecordPage
}

func (f *fakeRepo) List(_ context.Context, fl domain.Filter, deptID *int64, _ domain.Page) (domain.RecordPage, error) {
	f.lastFilter = fl
	f.lastDeptID = deptID
	if f.listErr != nil {
		return domain.RecordPage{}, f.listErr
	}
	return f.listResult, nil
}
func (f *fakeRepo) ListAll(_ context.Context, fl domain.Filter, deptID *int64, _, _ string, _ int) ([]domain.RiskRecord, error) {
	f.lastFilter = fl
	f.lastDeptID = deptID
	return f.listResult.Items, nil
}
func (f *fakeRepo) Get(_ context.Context, id int64, deptID *int64) (domain.RiskRecord, error) {
	f.lastDeptID = deptID
	if id == 404 {
		return domain.RiskRecord{}, domain.ErrNotFound
	}
	return domain.RiskRecord{ID: id}, nil
}
func (f *fakeRepo) Statuses(context.Context) ([]domain.Reference, error) {
	return []domain.Reference{{ID: 1, Title: "В работе у ДВГА"}}, nil
}
func (f *fakeRepo) Profiles(context.Context) ([]domain.Reference, error) {
	return []domain.Reference{{ID: 1, Title: "Профиль"}}, nil
}
func (f *fakeRepo) Departments(context.Context) ([]domain.Reference, error) {
	return []domain.Reference{{ID: 5, Title: "ДВГА по X"}}, nil
}
func (f *fakeRepo) Activities(context.Context) ([]domain.Reference, error) {
	return []domain.Reference{{ID: 3, Title: "Аудит 2026"}}, nil
}
func (f *fakeRepo) DepartmentByUserIIN(_ context.Context, iin string) (*int64, string, error) {
	f.iinLookups++
	if id, ok := f.deptByIIN[iin]; ok {
		return &id, f.deptTitles[id], nil
	}
	return nil, "", nil
}
func (f *fakeRepo) Ping(context.Context) error { return f.pingErr }

type fakeIIN struct {
	iin      string
	verified bool
	err      error
	calls    int
}

func (f *fakeIIN) UserIIN(context.Context, string) (string, bool, error) {
	f.calls++
	return f.iin, f.verified, f.err
}

func newTestSvc(repo *fakeRepo, iin *fakeIIN) *Service {
	return NewService(repo, iin, zap.NewNop())
}

func auditor(userID string) contract.Identity {
	return contract.Identity{UserID: userID, RoleCodes: []string{domain.RoleAuditor}}
}

// --- tests ---

func TestHasModuleAccess(t *testing.T) {
	if !HasModuleAccess(contract.Identity{IsAdmin: true}) {
		t.Fatal("admin must have access")
	}
	if !HasModuleAccess(contract.Identity{RoleCodes: []string{domain.RoleCurator}}) {
		t.Fatal("curator must have access")
	}
	if !HasModuleAccess(auditor("u1")) {
		t.Fatal("auditor must have access")
	}
	if HasModuleAccess(contract.Identity{RoleCodes: []string{"reporter_user"}}) {
		t.Fatal("stranger must NOT have access")
	}
}

func TestScopeByRoles(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{deptByIIN: map[string]int64{"880101300123": 7}, deptTitles: map[int64]string{7: "ДВГА по Y"}}

	// куратор и админ — все департаменты, ИИН не запрашивается
	iin := &fakeIIN{}
	svc := newTestSvc(repo, iin)
	for _, id := range []contract.Identity{
		{UserID: "c1", RoleCodes: []string{domain.RoleCurator}},
		{UserID: "a1", IsAdmin: true},
	} {
		scope, err := svc.Scope(ctx, id)
		if err != nil || !scope.AllDepartments || scope.Unmapped {
			t.Fatalf("%s: want AllDepartments, got %+v (%v)", id.UserID, scope, err)
		}
	}
	if iin.calls != 0 {
		t.Fatalf("curator/admin must not resolve IIN, calls=%d", iin.calls)
	}

	// аудитор с подтверждённым ИИН, найден в obm_evga
	iin = &fakeIIN{iin: "880101300123", verified: true}
	svc = newTestSvc(repo, iin)
	scope, err := svc.Scope(ctx, auditor("u1"))
	if err != nil || scope.DepartmentID == nil || *scope.DepartmentID != 7 || scope.DepartmentTitle != "ДВГА по Y" {
		t.Fatalf("mapped auditor: got %+v (%v)", scope, err)
	}

	// кэш: повторный вызов не дёргает провайдера и репозиторий
	if _, err := svc.Scope(ctx, auditor("u1")); err != nil {
		t.Fatal(err)
	}
	if iin.calls != 1 || repo.iinLookups != 1 {
		t.Fatalf("scope must be cached: iin.calls=%d repo.lookups=%d", iin.calls, repo.iinLookups)
	}

	// ИИН не подтверждён → Unmapped (в obm_evga не ходим)
	iin = &fakeIIN{iin: "880101300123", verified: false}
	svc = newTestSvc(repo, iin)
	before := repo.iinLookups
	scope, err = svc.Scope(ctx, auditor("u2"))
	if err != nil || !scope.Unmapped {
		t.Fatalf("unverified: want Unmapped, got %+v (%v)", scope, err)
	}
	if repo.iinLookups != before {
		t.Fatal("unverified IIN must not be looked up in obm_evga")
	}

	// ИИН не найден в obm_evga → Unmapped
	iin = &fakeIIN{iin: "000000000000", verified: true}
	svc = newTestSvc(repo, iin)
	scope, err = svc.Scope(ctx, auditor("u3"))
	if err != nil || !scope.Unmapped {
		t.Fatalf("not found: want Unmapped, got %+v (%v)", scope, err)
	}
}

func TestListVisibility(t *testing.T) {
	ctx := context.Background()
	dept9 := int64(9)
	repo := &fakeRepo{deptByIIN: map[string]int64{"880101300123": dept9}, deptTitles: map[int64]string{}}

	// аудитор: департамент навязан, клиентский department_id игнорируется (FR-4/6)
	svc := newTestSvc(repo, &fakeIIN{iin: "880101300123", verified: true})
	otherDept := int64(1)
	_, scope, err := svc.List(ctx, auditor("u1"), domain.Filter{DepartmentID: &otherDept}, domain.Page{})
	if err != nil {
		t.Fatal(err)
	}
	if scope.Unmapped || repo.lastDeptID == nil || *repo.lastDeptID != dept9 {
		t.Fatalf("auditor scope not enforced: lastDeptID=%v scope=%+v", repo.lastDeptID, scope)
	}
	if repo.lastFilter.DepartmentID != nil {
		t.Fatal("client department_id must be dropped for auditor")
	}

	// куратор: департамент не навязан, фильтр по департаменту работает
	svc = newTestSvc(repo, &fakeIIN{})
	_, _, err = svc.List(ctx, contract.Identity{UserID: "c1", RoleCodes: []string{domain.RoleCurator}}, domain.Filter{DepartmentID: &otherDept}, domain.Page{})
	if err != nil {
		t.Fatal(err)
	}
	if repo.lastDeptID != nil || repo.lastFilter.DepartmentID == nil || *repo.lastFilter.DepartmentID != otherDept {
		t.Fatalf("curator filter broken: lastDeptID=%v filter=%v", repo.lastDeptID, repo.lastFilter.DepartmentID)
	}

	// unmapped аудитор: пустой результат без похода в репозиторий
	repo2 := &fakeRepo{deptByIIN: map[string]int64{}}
	svc = newTestSvc(repo2, &fakeIIN{iin: "111", verified: true})
	res, scope, err := svc.List(ctx, auditor("u2"), domain.Filter{}, domain.Page{})
	if err != nil || !scope.Unmapped || len(res.Items) != 0 || res.Total != 0 {
		t.Fatalf("unmapped: want empty, got %+v scope=%+v (%v)", res, scope, err)
	}
}

func TestCardNotFoundOutOfScope(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{deptByIIN: map[string]int64{"880101300123": 9}, deptTitles: map[int64]string{}}
	svc := newTestSvc(repo, &fakeIIN{iin: "880101300123", verified: true})

	if _, err := svc.Card(ctx, auditor("u1"), 404); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	rec, err := svc.Card(ctx, auditor("u1"), 42)
	if err != nil || rec.ID != 42 {
		t.Fatalf("card: got %+v (%v)", rec, err)
	}
	// unmapped → NotFound
	svc = newTestSvc(&fakeRepo{deptByIIN: map[string]int64{}}, &fakeIIN{iin: "1", verified: true})
	if _, err := svc.Card(ctx, auditor("u9"), 42); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unmapped card: want ErrNotFound, got %v", err)
	}
}

func TestSourceUnavailableWrap(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{
		deptByIIN: map[string]int64{"880101300123": 9},
		listErr:   errors.New("dial tcp: connection refused"),
		pingErr:   errors.New("down"),
	}
	svc := newTestSvc(repo, &fakeIIN{iin: "880101300123", verified: true})
	_, _, err := svc.List(ctx, auditor("u1"), domain.Filter{}, domain.Page{})
	if !errors.Is(err, domain.ErrSourceUnavailable) {
		t.Fatalf("want ErrSourceUnavailable, got %v", err)
	}
}

func TestScopeCacheExpires(t *testing.T) {
	ctx := context.Background()
	repo := &fakeRepo{deptByIIN: map[string]int64{"880101300123": 7}, deptTitles: map[int64]string{}}
	iin := &fakeIIN{iin: "880101300123", verified: true}
	svc := newTestSvc(repo, iin)

	base := time.Unix(1_700_000_000, 0)
	svc.now = func() time.Time { return base }
	if _, err := svc.Scope(ctx, auditor("u1")); err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return base.Add(scopeCacheTTL + time.Second) }
	if _, err := svc.Scope(ctx, auditor("u1")); err != nil {
		t.Fatal(err)
	}
	if iin.calls != 2 {
		t.Fatalf("cache must expire after TTL, calls=%d", iin.calls)
	}
}
