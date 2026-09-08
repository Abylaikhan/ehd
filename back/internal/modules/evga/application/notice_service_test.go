package application

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"ehd-api/internal/modules/auth/contract"
	"ehd-api/internal/modules/evga/domain"
)

// --- fakes ---

type fakeNoticeRepo struct {
	lastDept    *int64
	lastOwner   int64
	lastBy      *int64
	lastGroups  []domain.CreateGroupInput
	created     []domain.CreatedNotice
	deleteErr   error
	previewErrs []domain.RejectedRecord
}

func (f *fakeNoticeRepo) Preview(_ context.Context, ids []int64, deptID *int64) ([]domain.NoticeGroup, []domain.RejectedRecord, error) {
	f.lastDept = deptID
	return []domain.NoticeGroup{{GU: "111", Count: len(ids)}}, f.previewErrs, nil
}
func (f *fakeNoticeRepo) SearchCli(context.Context, string) ([]domain.CliOrg, error) {
	return []domain.CliOrg{{ID: 1, Title: "Орг"}}, nil
}
func (f *fakeNoticeRepo) CreateNotices(_ context.Context, groups []domain.CreateGroupInput, deptID *int64, ownerDept int64, createdBy *int64, _ string) ([]domain.CreatedNotice, []domain.RejectedRecord, error) {
	f.lastGroups = groups
	f.lastDept = deptID
	f.lastOwner = ownerDept
	f.lastBy = createdBy
	if f.created == nil {
		f.created = []domain.CreatedNotice{{NoticeID: 100, GU: "111", Records: 2}}
	}
	return f.created, nil, nil
}
func (f *fakeNoticeRepo) ListNotices(_ context.Context, deptID, _, _ *int64, _ string, _ domain.Page) ([]domain.NoticeListItem, int64, error) {
	f.lastDept = deptID
	return []domain.NoticeListItem{}, 0, nil
}
func (f *fakeNoticeRepo) GetNotice(_ context.Context, id int64, deptID *int64) (domain.NoticeCard, error) {
	f.lastDept = deptID
	return domain.NoticeCard{NoticeListItem: domain.NoticeListItem{ID: id}}, nil
}
func (f *fakeNoticeRepo) DeleteNotice(_ context.Context, _ int64, deptID *int64) error {
	f.lastDept = deptID
	return f.deleteErr
}

func newNoticeSvc(repo *fakeNoticeRepo, scopes *fakeScopes, write bool) *NoticeService {
	users := &fakeStatusRepo{userByIIN: map[string]int64{"880101300123": 55}}
	return NewNoticeService(repo, users, scopes, write, zap.NewNop())
}

// --- tests ---

func TestNoticeCreateRBAC(t *testing.T) {
	ctx := context.Background()
	groups := []domain.CreateGroupInput{{RecordIDs: []int64{1, 2}, CliID: 9}}

	// read-only режим
	svc := newNoticeSvc(&fakeNoticeRepo{}, &fakeScopes{scope: dept(7)}, false)
	if _, err := svc.Create(ctx, auditor("u1"), groups); !errors.Is(err, ErrReadOnlyMode) {
		t.Fatalf("read-only: %v", err)
	}

	// куратор — запрещено
	svc = newNoticeSvc(&fakeNoticeRepo{}, &fakeScopes{}, true)
	curator := contract.Identity{UserID: "c1", RoleCodes: []string{domain.RoleCurator}}
	if _, err := svc.Create(ctx, curator, groups); !errors.Is(err, ErrCuratorReadOnly) {
		t.Fatalf("curator: %v", err)
	}

	// админ без департамента-владельца — запрещено (ErrNoOwnerDepartment)
	svc = newNoticeSvc(&fakeNoticeRepo{}, &fakeScopes{scope: domain.DepartmentScope{AllDepartments: true}}, true)
	admin := contract.Identity{UserID: "a1", IsAdmin: true}
	if _, err := svc.Create(ctx, admin, groups); !errors.Is(err, domain.ErrNoOwnerDepartment) {
		t.Fatalf("admin: %v", err)
	}

	// аудитор с департаментом — ок, владелец и created_by проставлены
	repo := &fakeNoticeRepo{}
	svc = newNoticeSvc(repo, &fakeScopes{scope: dept(7), iin: "880101300123"}, true)
	res, err := svc.Create(ctx, auditor("u1"), groups)
	if err != nil || len(res.Created) != 1 {
		t.Fatalf("auditor create: %+v (%v)", res, err)
	}
	if repo.lastOwner != 7 || repo.lastDept == nil || *repo.lastDept != 7 {
		t.Fatalf("owner/dept: %d %v", repo.lastOwner, repo.lastDept)
	}
	if repo.lastBy == nil || *repo.lastBy != 55 {
		t.Fatalf("created_by: %v", repo.lastBy)
	}
}

func TestNoticeCreateValidation(t *testing.T) {
	ctx := context.Background()
	svc := newNoticeSvc(&fakeNoticeRepo{}, &fakeScopes{scope: dept(7), iin: "880101300123"}, true)

	// без адресата — 400 ADDRESSEE_REQUIRED (EVGA-FR-034)
	if _, err := svc.Create(ctx, auditor("u1"), []domain.CreateGroupInput{{RecordIDs: []int64{1}}}); !errors.Is(err, domain.ErrAddresseeRequired) {
		t.Fatalf("no cli: %v", err)
	}
	// пусто
	if _, err := svc.Create(ctx, auditor("u1"), nil); !errors.Is(err, domain.ErrBulkLimit) {
		t.Fatalf("empty: %v", err)
	}
}

func TestNoticePreviewScope(t *testing.T) {
	ctx := context.Background()
	repo := &fakeNoticeRepo{}
	svc := newNoticeSvc(repo, &fakeScopes{scope: dept(9), iin: "880101300123"}, false) // preview работает и в read-only
	groups, _, err := svc.Preview(ctx, auditor("u1"), []int64{1, 2, 3})
	if err != nil || len(groups) != 1 {
		t.Fatalf("preview: %v %v", groups, err)
	}
	if repo.lastDept == nil || *repo.lastDept != 9 {
		t.Fatalf("dept filter: %v", repo.lastDept)
	}
	// куратору preview не нужен и запрещён
	curator := contract.Identity{UserID: "c1", RoleCodes: []string{domain.RoleCurator}}
	if _, _, err := svc.Preview(ctx, curator, []int64{1}); !errors.Is(err, ErrCuratorReadOnly) {
		t.Fatalf("curator preview: %v", err)
	}
}

func TestNoticeListScope(t *testing.T) {
	ctx := context.Background()
	repo := &fakeNoticeRepo{}
	svc := newNoticeSvc(repo, &fakeScopes{scope: dept(9), iin: "880101300123"}, true)
	_, _, scope, err := svc.List(ctx, auditor("u1"), nil, nil, "", domain.Page{})
	if err != nil || scope.Unmapped {
		t.Fatalf("list: %v %+v", err, scope)
	}
	if repo.lastDept == nil || *repo.lastDept != 9 {
		t.Fatalf("dept: %v", repo.lastDept)
	}
	// unmapped аудитор — пусто без похода в репо
	svc = newNoticeSvc(&fakeNoticeRepo{}, &fakeScopes{scope: domain.DepartmentScope{Unmapped: true}}, true)
	items, total, scope, err := svc.List(ctx, auditor("u2"), nil, nil, "", domain.Page{})
	if err != nil || !scope.Unmapped || len(items) != 0 || total != 0 {
		t.Fatalf("unmapped list: %v %v %+v", items, total, scope)
	}
}

func TestNoticeSearchCliMinLen(t *testing.T) {
	svc := newNoticeSvc(&fakeNoticeRepo{}, &fakeScopes{}, true)
	items, err := svc.SearchCli(context.Background(), " а ")
	if err != nil || len(items) != 0 {
		t.Fatalf("short q: %v %v", items, err)
	}
}
