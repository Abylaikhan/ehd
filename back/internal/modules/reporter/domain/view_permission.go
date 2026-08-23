package domain

// ViewPermission — доступ роли к представлению (ТЗ, Права).
// Администратор имеет доступ всегда, независимо от этих записей.
type ViewPermission struct {
	ID       string
	ViewID   string
	RoleCode string
}
