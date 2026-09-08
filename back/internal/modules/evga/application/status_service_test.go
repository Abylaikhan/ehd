package application

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"ehd-api/internal/modules/auth/contract"
	"ehd-api/internal/modules/evga/domain"
	"ehd-api/internal/modules/evga/repository"
)

// --- fakes ---

type fakeStatusRepo struct {
	lastIDs    []int64
	lastDept   *int64
	lastBy     *int64
	lastSource string
	outcomes   []repository.ChangeOutcome
	userByIIN  map[string]int64
}

func (f *fakeStatusRepo) ApplyStatusChange(_ context.Context, ids []int64, _ int64, _ domain.TransitionAttrs, deptID *int64, changedBy *int64, source string) ([]repository.ChangeOutcome, int, error) {
	f.lastIDs = ids
	f.lastDept = deptID
	f.lastBy = changedBy
	f.lastSource = source
	if f.outcomes != nil {
		return f.outcomes, 0, nil
	}
	out := make([]repository.ChangeOutcome, len(ids))
	for i, id := range ids {
		out[i] = repository.ChangeOutcome{ID: id}
	}
	return out, 0, nil
}
func (f *fakeStatusRepo) History(context.Context, int64) ([]repository.HistoryEntry, error) {
	return []repository.HistoryEntry{}, nil
}
func (f *fakeStatusRepo) UserIDByIIN(_ context.Context, iin string) (*int64, error) {
	if id, ok := f.userByIIN[iin]; ok {
		return &id, nil
	}
	return nil, nil
}

type fakeScopes struct {
	scope domain.DepartmentScope
	iin   string
}

func (f *fakeScopes) Scope(context.Context, contract.Identity) (domain.DepartmentScope, error) {
	return f.scope, nil
}
func (f *fakeScopes) UserIINOf(context.Context, contract.Identity) (string, bool, error) {
	return f.iin, f.iin != "", nil
}

func dept(id int64) domain.DepartmentScope { return domain.DepartmentScope{DepartmentID: &id} }

// --- tests ---

func TestStatusReadOnlyMode(t *testing.T) {
	svc := NewStatusService(&fakeStatusRepo{}, &fakeScopes{scope: dept(7)}, false, zap.NewNop())
	err := svc.ChangeStatus(context.Background(), auditor("u1"), 1, 2, domain.TransitionAttrs{Note: "x"})
	if !errors.Is(err, ErrReadOnlyMode) {
		t.Fatalf("want ErrReadOnlyMode, got %v", err)
	}
	if _, err := svc.BulkChangeStatus(context.Background(), auditor("u1"), []int64{1}, 2, domain.TransitionAttrs{}); !errors.Is(err, ErrReadOnlyMode) {
		t.Fatalf("bulk: want ErrReadOnlyMode, got %v", err)
	}
}

func TestStatusCuratorForbidden(t *testing.T) {
	svc := NewStatusService(&fakeStatusRepo{}, &fakeScopes{}, true, zap.NewNop())
	curator := contract.Identity{UserID: "c1", RoleCodes: []string{domain.RoleCurator}}
	if err := svc.ChangeStatus(context.Background(), curator, 1, 2, domain.TransitionAttrs{}); !errors.Is(err, ErrCuratorReadOnly) {
		t.Fatalf("want ErrCuratorReadOnly, got %v", err)
	}
}

func TestStatusAuditorScopeAndChangedBy(t *testing.T) {
	repo := &fakeStatusRepo{userByIIN: map[string]int64{"880101300123": 55}}
	svc := NewStatusService(repo, &fakeScopes{scope: dept(9), iin: "880101300123"}, true, zap.NewNop())
	if err := svc.ChangeStatus(context.Background(), auditor("u1"), 42, 2, domain.TransitionAttrs{Note: "n"}); err != nil {
		t.Fatal(err)
	}
	if repo.lastDept == nil || *repo.lastDept != 9 {
		t.Fatalf("department not enforced: %v", repo.lastDept)
	}
	if repo.lastBy == nil || *repo.lastBy != 55 {
		t.Fatalf("changed_by not resolved: %v", repo.lastBy)
	}
	if repo.lastSource != domain.ChangeSourceManual {
		t.Fatalf("source: %s", repo.lastSource)
	}
}

func TestStatusAdminNoDeptFilterNullChangedBy(t *testing.T) {
	repo := &fakeStatusRepo{userByIIN: map[string]int64{}}
	svc := NewStatusService(repo, &fakeScopes{}, true, zap.NewNop())
	admin := contract.Identity{UserID: "a1", IsAdmin: true}
	if err := svc.ChangeStatus(context.Background(), admin, 42, 2, domain.TransitionAttrs{Note: "n"}); err != nil {
		t.Fatal(err)
	}
	if repo.lastDept != nil {
		t.Fatalf("admin must not be dept-filtered: %v", repo.lastDept)
	}
	if repo.lastBy != nil {
		t.Fatalf("changed_by must be NULL for admin without obm_evga user: %v", repo.lastBy)
	}
}

func TestStatusSingleTransitionErrorPropagates(t *testing.T) {
	rejected := &domain.TransitionError{Base: domain.ErrTransitionNotAllowed, Reason: "Переход из статуса 4 в статус 1 не предусмотрен"}
	repo := &fakeStatusRepo{outcomes: []repository.ChangeOutcome{{ID: 1, Err: rejected, Reason: rejected.Reason}}}
	svc := NewStatusService(repo, &fakeScopes{scope: dept(9), iin: "1"}, true, zap.NewNop())
	err := svc.ChangeStatus(context.Background(), auditor("u1"), 1, 1, domain.TransitionAttrs{})
	if !errors.Is(err, domain.ErrTransitionNotAllowed) {
		t.Fatalf("want ErrTransitionNotAllowed, got %v", err)
	}
}

func TestBulkReportCounts(t *testing.T) {
	repo := &fakeStatusRepo{outcomes: []repository.ChangeOutcome{
		{ID: 1}, {ID: 2},
		{ID: 3, Err: domain.ErrTransitionNotAllowed, Reason: "Переход из статуса 8 в статус 2 не предусмотрен"},
	}}
	svc := NewStatusService(repo, &fakeScopes{scope: dept(9), iin: "1"}, true, zap.NewNop())
	rep, err := svc.BulkChangeStatus(context.Background(), auditor("u1"), []int64{1, 2, 3}, 2, domain.TransitionAttrs{Note: "n"})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Processed != 2 || rep.Rejected != 1 || len(rep.Rejections) != 1 || rep.Rejections[0].ID != 3 {
		t.Fatalf("report: %+v", rep)
	}
	if repo.lastSource != domain.ChangeSourceBulk {
		t.Fatalf("source: %s", repo.lastSource)
	}
}

func TestBulkLimit(t *testing.T) {
	svc := NewStatusService(&fakeStatusRepo{}, &fakeScopes{scope: dept(9)}, true, zap.NewNop())
	if _, err := svc.BulkChangeStatus(context.Background(), auditor("u1"), nil, 2, domain.TransitionAttrs{}); !errors.Is(err, domain.ErrBulkLimit) {
		t.Fatalf("empty: %v", err)
	}
	big := make([]int64, MaxBulkIDs+1)
	if _, err := svc.BulkChangeStatus(context.Background(), auditor("u1"), big, 2, domain.TransitionAttrs{}); !errors.Is(err, domain.ErrBulkLimit) {
		t.Fatalf("too big: %v", err)
	}
}
