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

section "Building cmd/server"
go build ./cmd/server

section "Building cmd/memory_agent_cmd"
go build ./cmd/memory_agent_cmd

section "Building cmd/memory_chat_cmd"
go build ./cmd/memory_chat_cmd

section "Building cmd/memory_index_cmd"
go build ./cmd/memory_index_cmd

section "Building cmd/text_analyze_cmd"
go build ./cmd/text_analyze_cmd

section "Building cmd/image_analyze_cmd"
go build ./cmd/image_analyze_cmd

if [[ "${RUN_AI:-0}" == "1" ]]; then
  section "Smoke test: text_analyze_cmd"
  go run ./cmd/text_analyze_cmd "今天我把 MemoryFlow 的 cmd 调试入口补好了"

  section "Smoke test: memory_chat_cmd"
  go run ./cmd/memory_chat_cmd "我最近在做什么项目"

  section "Smoke test: memory_agent_cmd debug trace"
  go run ./cmd/memory_agent_cmd --debug "最近我记录了什么"
fi

if [[ "${RUN_INDEX:-0}" == "1" ]]; then
  section "Smoke test: memory_index_cmd"
  go run ./cmd/memory_index_cmd --batch-size=50
fi

section "Cleaning build artifacts"
rm -f server memory_agent_cmd memory_chat_cmd memory_index_cmd text_analyze_cmd image_analyze_cmd

printf '\nAll checks passed\n'
