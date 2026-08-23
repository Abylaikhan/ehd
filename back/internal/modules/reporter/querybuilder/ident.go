// Package querybuilder — безопасная сборка параметризованного read-only SELECT
// для ClickHouse по опубликованному snapshot (Принцип 3 конституции).
// Идентификаторы берутся только из whitelist, проверяются регуляркой и экранируются
// backtick; значения передаются исключительно параметрами (?), без конкатенации.
package querybuilder

import "regexp"

var identRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// SafeIdent проверяет, что идентификатор допустим (латиница/цифры/подчёркивание).
func SafeIdent(s string) bool { return identRe.MatchString(s) }

// quoteIdent экранирует уже проверенный идентификатор backtick'ами.
func quoteIdent(s string) string { return "`" + s + "`" }
