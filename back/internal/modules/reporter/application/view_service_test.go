package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"ehd-api/internal/modules/reporter/domain"
)

// --- фейки ---

type fakeViewRepo struct {
	view        domain.DataView
	viewErr     error
	cols        []domain.ViewColumn
	roles       []string
	slugTaken   bool
	created     domain.DataView
	createdCols []domain.ViewColumn
	pubSnapshot string
	pubHash     string
	pubCalled   bool
	status      string
	statusSet   bool
}

func (r *fakeViewRepo) CreateView(_ context.Context, v domain.DataView, cols []domain.ViewColumn) (domain.DataView, error) {
	r.created = v
	r.createdCols = cols
	v.ID = "vid"
	return v, nil
}
func (r *fakeViewRepo) GetView(context.Context, string) (domain.DataView, error) {
	if r.viewErr != nil {
		return domain.DataView{}, r.viewErr
	}
	return r.view, nil
}
func (r *fakeViewRepo) GetViewBySlug(context.Context, string) (domain.DataView, error) {
	if r.viewErr != nil {
		return domain.DataView{}, r.viewErr
	}
	return r.view, nil
}
func (r *fakeViewRepo) ListViews(context.Context) ([]domain.DataView, error) { return nil, nil }
func (r *fakeViewRepo) UpdateViewMeta(_ context.Context, v domain.DataView) (domain.DataView, error) {
	r.view = v
	return v, nil
}
func (r *fakeViewRepo) SetStatus(_ context.Context, _ string, status string) error {
	r.status = status
	r.statusSet = true
	return nil
}
func (r *fakeViewRepo) DeleteView(context.Context, string) error { return nil }
func (r *fakeViewRepo) SlugExists(context.Context, string, string) (bool, error) {
	return r.slugTaken, nil
}
func (r *fakeViewRepo) GetColumns(context.Context, string) ([]domain.ViewColumn, error) {
	return r.cols, nil
}
func (r *fakeViewRepo) SaveColumns(_ context.Context, cols []domain.ViewColumn) error {
	r.cols = cols
	return nil
}
func (r *fakeViewRepo) ReplaceColumns(_ context.Context, _ string, cols []domain.ViewColumn) error {
	r.cols = cols
	return nil
}
func (r *fakeViewRepo) GetPermissions(context.Context, string) ([]string, error) { return r.roles, nil }
func (r *fakeViewRepo) SetPermissions(_ context.Context, _ string, codes []string) error {
	r.roles = codes
	return nil
}
func (r *fakeViewRepo) Publish(_ context.Context, _, snap, hash string, _ time.Time) error {
	r.pubSnapshot = snap
	r.pubHash = hash
	r.pubCalled = true
	return nil
}
func (r *fakeViewRepo) GetSnapshot(context.Context, string) (string, error) {
	return r.pubSnapshot, nil
}

type fakeInspector struct {
	source  domain.DataSource
	getErr  error
	cols    []domain.Column
	colsErr error
}

func (f fakeInspector) GetSource(context.Context, string) (domain.DataSource, error) {
	return f.source, f.getErr
}
func (f fakeInspector) Columns(context.Context, string, string, string) ([]domain.Column, error) {
	return f.cols, f.colsErr
}

var ctx = context.Background()

func activeSource() domain.DataSource {
	return domain.DataSource{ID: "s", Status: domain.SourceStatusActive}
}

// --- тесты ---

func TestCreateView_LoadsColumnsHiddenWithDisplayType(t *testing.T) {
	repo := &fakeViewRepo{}
	insp := fakeInspector{
		source: activeSource(),
		cols: []domain.Column{
			{Name: "id", Type: "UInt64", Position: 1},
			{Name: "amount", Type: "Decimal(18, 2)", Position: 2},
			{Name: "created_at", Type: "DateTime", Position: 3},
		},
	}
	svc := NewViewService(repo, insp, zap.NewNop())

	_, err := svc.CreateView(ctx, CreateViewInput{
		Name: "Демо", Slug: "demo", DataSourceID: "s", Database: "ehd_src", Table: "demo_transactions",
	})
	if err != nil {
		t.Fatalf("CreateView: %v", err)
	}
	if len(repo.createdCols) != 3 {
		t.Fatalf("ожидалось 3 колонки, получено %d", len(repo.createdCols))
	}
	for _, c := range repo.createdCols {
		if c.Visible {
			t.Errorf("колонка %q должна быть невидимой по умолчанию", c.SourceName)
		}
	}
	if repo.createdCols[0].DisplayType != domain.DisplayNumber ||
		repo.createdCols[2].DisplayType != domain.DisplayDateTime {
		t.Errorf("display_type выведен неверно: %+v", repo.createdCols)
	}
}

func TestCreateView_Rejects(t *testing.T) {
	// неактивный источник
	repo := &fakeViewRepo{}
	svc := NewViewService(repo, fakeInspector{source: domain.DataSource{Status: domain.SourceStatusInactive}}, zap.NewNop())
	if _, err := svc.CreateView(ctx, CreateViewInput{Name: "n", Slug: "demo", DataSourceID: "s", Database: "d", Table: "t"}); !errors.Is(err, domain.ErrSourceInactive) {
		t.Fatalf("неактивный источник → ожидалась ErrSourceInactive, получено %v", err)
	}
	// плохой slug
	svc = NewViewService(&fakeViewRepo{}, fakeInspector{source: activeSource()}, zap.NewNop())
	if _, err := svc.CreateView(ctx, CreateViewInput{Name: "n", Slug: "Bad_Slug", DataSourceID: "s", Database: "d", Table: "t"}); !errors.Is(err, domain.ErrViewValidation) {
		t.Fatalf("плохой slug → ожидалась ErrViewValidation, получено %v", err)
	}
	// таблица не найдена (пустая интроспекция)
	svc = NewViewService(&fakeViewRepo{}, fakeInspector{source: activeSource(), cols: nil}, zap.NewNop())
	if _, err := svc.CreateView(ctx, CreateViewInput{Name: "n", Slug: "demo", DataSourceID: "s", Database: "d", Table: "missing"}); !errors.Is(err, domain.ErrTableNotFound) {
		t.Fatalf("нет колонок → ожидалась ErrTableNotFound, получено %v", err)
	}
}

func TestPublish_Gates(t *testing.T) {
	newRepo := func() *fakeViewRepo {
		return &fakeViewRepo{view: domain.DataView{ID: "v", Slug: "demo", DataSourceID: "s", Status: domain.ViewStatusDraft}}
	}
	// нет видимых колонок
	repo := newRepo()
	repo.cols = []domain.ViewColumn{{SourceName: "a", Visible: false}}
	repo.roles = []string{"analyst"}
	svc := NewViewService(repo, fakeInspector{source: activeSource()}, zap.NewNop())
	if _, err := svc.Publish(ctx, "v"); !errors.Is(err, domain.ErrPublishValidation) {
		t.Fatalf("нет видимых колонок → ожидалась ErrPublishValidation, получено %v", err)
	}
	// нет ролей
	repo = newRepo()
	repo.cols = []domain.ViewColumn{{SourceName: "a", Visible: true}}
	repo.roles = nil
	svc = NewViewService(repo, fakeInspector{source: activeSource()}, zap.NewNop())
	if _, err := svc.Publish(ctx, "v"); !errors.Is(err, domain.ErrPublishValidation) {
		t.Fatalf("нет ролей → ожидалась ErrPublishValidation, получено %v", err)
	}
	// неактивный источник
	repo = newRepo()
	repo.cols = []domain.ViewColumn{{SourceName: "a", Visible: true}}
	repo.roles = []string{"analyst"}
	svc = NewViewService(repo, fakeInspector{source: domain.DataSource{Status: domain.SourceStatusInactive}}, zap.NewNop())
	if _, err := svc.Publish(ctx, "v"); !errors.Is(err, domain.ErrPublishValidation) {
		t.Fatalf("неактивный источник → ожидалась ErrPublishValidation, получено %v", err)
	}
}

func TestPublish_SnapshotOnlyVisibleColumns(t *testing.T) {
	repo := &fakeViewRepo{
		view: domain.DataView{ID: "v", Slug: "demo", DataSourceID: "s", Status: domain.ViewStatusDraft, RowScopeMode: domain.RowScopeByProfile},
		cols: []domain.ViewColumn{
			{SourceName: "id", SourceType: "UInt64", Visible: true, Position: 1},
			{SourceName: "secret", SourceType: "String", Visible: false, Position: 2},
			{SourceName: "name", SourceType: "String", Visible: true, Position: 3},
		},
		roles: []string{"analyst"},
	}
	svc := NewViewService(repo, fakeInspector{source: activeSource()}, zap.NewNop())

	if _, err := svc.Publish(ctx, "v"); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if !repo.pubCalled {
		t.Fatal("Publish не был вызван у репозитория")
	}
	if repo.pubHash == "" {
		t.Error("schema_hash пустой")
	}
	var snap domain.PublishedSnapshot
	if err := json.Unmarshal([]byte(repo.pubSnapshot), &snap); err != nil {
		t.Fatalf("snapshot JSON: %v", err)
	}
	if len(snap.Columns) != 2 {
		t.Fatalf("в snapshot должно быть 2 видимые колонки, получено %d", len(snap.Columns))
	}
	for _, c := range snap.Columns {
		if c.SourceName == "secret" {
			t.Error("скрытая колонка попала в snapshot (нарушение REP-BR-007)")
		}
	}
}

func TestUpdateColumns_DemotesPublishedToDraft(t *testing.T) {
	repo := &fakeViewRepo{
		view: domain.DataView{ID: "v", Status: domain.ViewStatusPublished},
		cols: []domain.ViewColumn{{ID: "c1", SourceName: "id", SourceType: "UInt64"}},
	}
	svc := NewViewService(repo, fakeInspector{source: activeSource()}, zap.NewNop())

	if err := svc.UpdateColumns(ctx, "v", []ColumnConfigInput{{SourceName: "id", Visible: true}}); err != nil {
		t.Fatalf("UpdateColumns: %v", err)
	}
	if !repo.statusSet || repo.status != domain.ViewStatusDraft {
		t.Errorf("редактирование опубликованного должно переводить в draft; statusSet=%v status=%q", repo.statusSet, repo.status)
	}
}
