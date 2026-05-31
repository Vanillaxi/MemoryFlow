#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${BACKEND_DIR}"

section() {
  printf '\n==> %s\n' "$1"
}

section "Running Go tests"
go test ./...

section "Building service"
go build .

section "Building cmd/chat_cmd"
go build ./cmd/chat_cmd

section "Building cmd/knowledge_cmd"
go build ./cmd/knowledge_cmd

if [[ "${RUN_AI:-0}" == "1" ]]; then
  section "Smoke test: chat_cmd recent week debug trace"
  go run ./cmd/chat_cmd --debug "最近一周我记录了什么"

  section "Smoke test: chat_cmd aggregate debug trace"
  go run ./cmd/chat_cmd --debug "总结一下五月份我主要做了什么"
fi

if [[ "${RUN_INDEX:-0}" == "1" ]]; then
  section "Smoke test: knowledge_cmd"
  go run ./cmd/knowledge_cmd --batch-size=50
fi

section "Cleaning build artifacts"
rm -f memoryflow chat_cmd knowledge_cmd

printf '\nAll checks passed\n'
