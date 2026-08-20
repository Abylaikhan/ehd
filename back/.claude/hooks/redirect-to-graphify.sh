#!/usr/bin/env bash
# PreToolUse hook: жёстко запрещает Grep/Glob для анализа структуры кода
# и перенаправляет агента на Graphify Knowledge Graph.
cat >/dev/null   # поглотить вход инструмента со stdin

cat <<'JSON'
{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"Grep/Glob запрещены для анализа структуры кода в этом проекте. Используй Graphify MCP (mcp__graphify): graphify_locate — найти символ/файл; graphify_query/graphify_path/graphify_explain — связи и зависимости; graphify_overview/graphify_subgraph/graphify_skeleton — структура; graphify_impact — что затронет изменение. Если граф пуст или устарел (graphify_freshness) — сначала graphify_build (update=true). Для чтения конкретного известного файла используй Read. Для навигации по Go-символам — GoLand MCP (mcp__jetbrains)."}}
JSON
