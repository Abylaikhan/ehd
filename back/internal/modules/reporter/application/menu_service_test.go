package application

import (
	"context"
	"testing"

	"ehd-api/internal/modules/reporter/domain"
)

type fakeMenuRepo struct {
	items []domain.MenuItem
	slugs map[string]string
}

func (r *fakeMenuRepo) List(context.Context) ([]domain.MenuItem, error) { return r.items, nil }
func (r *fakeMenuRepo) Get(_ context.Context, id string) (domain.MenuItem, error) {
	for _, it := range r.items {
		if it.ID == id {
			return it, nil
		}
	}
	return domain.MenuItem{}, domain.ErrMenuNotFound
}
func (r *fakeMenuRepo) Create(_ context.Context, it domain.MenuItem) (domain.MenuItem, error) {
	it.ID = "new"
	return it, nil
}
func (r *fakeMenuRepo) Update(_ context.Context, it domain.MenuItem) (domain.MenuItem, error) {
	return it, nil
}
func (r *fakeMenuRepo) Delete(context.Context, string) error { return nil }
func (r *fakeMenuRepo) ViewSlugByID(context.Context) (map[string]string, error) {
	return r.slugs, nil
}

func TestValidateParent_CycleAndDepth(t *testing.T) {
	// a → b → c (глубина 3)
	items := []domain.MenuItem{
		{ID: "a"},
		{ID: "b", ParentID: "a"},
		{ID: "c", ParentID: "b"},
	}
	// цикл: a под c (c — потомок a)
	if err := validateParent(items, "a", "c"); err != domain.ErrMenuCycle {
		t.Errorf("цикл → ожидалась ErrMenuCycle, получено %v", err)
	}
	// самородитель
	if err := validateParent(items, "b", "b"); err != domain.ErrMenuCycle {
		t.Errorf("self-parent → ожидалась ErrMenuCycle, получено %v", err)
	}
	// глубина: новый пункт под c (глубина c=3) → 4 > 3
	if err := validateParent(items, "", "c"); err != domain.ErrMenuDepth {
		t.Errorf("глубина → ожидалась ErrMenuDepth, получено %v", err)
	}
	// ок: новый под a (глубина a=1) → 2
	if err := validateParent(items, "", "a"); err != nil {
		t.Errorf("валидный родитель → ошибок быть не должно, получено %v", err)
	}
	// несуществующий родитель
	if err := validateParent(items, "", "zzz"); err != domain.ErrMenuValidation {
		t.Errorf("неизвестный родитель → ожидалась ErrMenuValidation, получено %v", err)
	}
}

func TestNavigation_FiltersAndPrunes(t *testing.T) {
	repo := &fakeMenuRepo{
		items: []domain.MenuItem{
			{ID: "sec", NameRu: "Раздел", Position: 1},
			{ID: "pub", ParentID: "sec", NameRu: "Публичный", DataViewID: "v1", PublicAccess: true, Position: 1},
			{ID: "role", ParentID: "sec", NameRu: "Аналитика", DataViewID: "v2", RoleCodes: []string{"analyst"}, Position: 2},
			{ID: "empty", NameRu: "Пустой раздел", Position: 2},
		},
		slugs: map[string]string{"v1": "view-one", "v2": "view-two"},
	}
	svc := NewMenuService(repo)

	// пользователь без ролей: видит только публичный пункт; пустой раздел скрыт
	nodes, err := svc.Navigation(ctx, Requester{})
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 || nodes[0].ID != "sec" {
		t.Fatalf("ожидался один раздел sec, получено %+v", nodes)
	}
	if len(nodes[0].Children) != 1 || nodes[0].Children[0].To != "/reporter/view-one" {
		t.Errorf("без ролей должен быть виден только публичный пункт с маршрутом: %+v", nodes[0].Children)
	}

	// аналитик: видит и публичный, и ролевой пункт
	na, _ := svc.Navigation(ctx, Requester{RoleCodes: []string{"analyst"}})
	if len(na) != 1 || len(na[0].Children) != 2 {
		t.Errorf("аналитик должен видеть оба пункта: %+v", na)
	}

	// админ: видит всё, но пустой раздел без маршрута/детей всё равно отсечён
	nad, _ := svc.Navigation(ctx, Requester{IsAdmin: true})
	for _, n := range nad {
		if n.ID == "empty" {
			t.Errorf("пустой раздел не должен попадать в навигацию даже админу")
		}
	}
}
