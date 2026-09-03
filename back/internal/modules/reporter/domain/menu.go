package domain

import (
	"errors"
	"time"
)

// MaxMenuDepth — предельная глубина дерева меню (ТЗ, стр. 929).
const MaxMenuDepth = 3

// MenuItem — пункт навигации Reporter (ТЗ, menu_items).
type MenuItem struct {
	ID           string
	ParentID     string // "" — корневой
	DataViewID   string // "" — раздел без привязки
	NameRu       string
	NameKk       string
	IconKey      string
	Position     int
	IsDisabled   bool
	PublicAccess bool
	RoleCodes    []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// MenuNode — узел разрешённого дерева навигации для пользователя.
type MenuNode struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	Icon     string     `json:"icon"`
	To       string     `json:"to"` // /reporter/{slug} или "" для раздела
	Children []MenuNode `json:"children"`
}

var (
	// ErrMenuNotFound — пункт меню не найден.
	ErrMenuNotFound = errors.New("пункт меню не найден")
	// ErrMenuValidation — некорректные данные пункта меню.
	ErrMenuValidation = errors.New("ошибка валидации пункта меню")
	// ErrMenuCycle — родитель создаёт цикл (родитель = сам пункт или его потомок).
	ErrMenuCycle = errors.New("недопустимая вложенность меню (цикл)")
	// ErrMenuDepth — превышена максимальная глубина дерева.
	ErrMenuDepth = errors.New("превышена максимальная глубина меню")
	// ErrMenuHasChildren — удаление пункта с вложенными запрещено.
	ErrMenuHasChildren = errors.New("сначала удалите вложенные пункты")
)
