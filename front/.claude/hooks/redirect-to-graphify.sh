#!/usr/bin/env bash
# PreToolUse hook: жёстко запрещает Grep/Glob для анализа структуры кода
# и перенаправляет агента на Graphify Knowledge Graph.
cat >/dev/null   # поглотить вход инструмента со stdin

cat <<'JSON'
{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"Grep/Glob запрещены для анализа структуры фронтенда. Используй Graphify MCP (mcp__graphify): graphify_locate — найти компонент/composable/страницу; graphify_query/graphify_path/graphify_explain — связи; graphify_overview/graphify_subgraph — структура layers; graphify_impact — что затронет изменение. Если граф пуст/устарел (graphify_freshness) — сначала graphify_build (update=true). Для чтения конкретного известного файла используй Read."}}
JSON
