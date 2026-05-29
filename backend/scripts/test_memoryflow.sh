#!/usr/bin/env bash

set -e

BASE_URL="${BASE_URL:-http://localhost:8080}"
MEMORY_ID="${MEMORY_ID:-20}"

echo "========================================"
echo "MemoryFlow API Test"
echo "BASE_URL=${BASE_URL}"
echo "MEMORY_ID=${MEMORY_ID}"
echo "========================================"
echo

check_jq() {
  if ! command -v jq >/dev/null 2>&1; then
    echo "❌ jq not found. Please install jq first."
    echo "macOS: brew install jq"
    exit 1
  fi
}

request_get() {
  local name="$1"
  local url="$2"

  echo "----------------------------------------"
  echo "TEST: ${name}"
  echo "GET ${url}"
  echo "----------------------------------------"

  curl -s "${url}" | jq
  echo
}

request_post() {
  local name="$1"
  local url="$2"

  echo "----------------------------------------"
  echo "TEST: ${name}"
  echo "POST ${url}"
  echo "----------------------------------------"

  curl -s -X POST "${url}" | jq
  echo
}

check_jq

echo "✅ jq exists"
echo

# 1. Eino tools list
request_get \
  "Eino tool list" \
  "${BASE_URL}/api/agent/tools"

# 2. Main ask endpoint: basic QA
request_get \
  "Ask memory basic" \
  "${BASE_URL}/api/memories/ask?q=我什么时候修好了embedding&debug=true"

# 3. Ask with type filter
request_get \
  "Ask memory with type=image" \
  "${BASE_URL}/api/memories/ask?q=图片记忆&type=image&debug=true"

# 4. Ask with time filter
request_get \
  "Ask memory with date range" \
  "${BASE_URL}/api/memories/ask?q=MemoryFlow&start=2026-05-01&end=2026-06-01&debug=true"

# 5. Semantic search
request_get \
  "Semantic search" \
  "${BASE_URL}/api/memories/search?q=MemoryFlow&top_k=5"

# 6. Semantic search with type filter
request_get \
  "Semantic search with type=text" \
  "${BASE_URL}/api/memories/search?q=MemoryFlow&type=text&top_k=5"

# 7. Semantic search with date range
request_get \
  "Semantic search with date range" \
  "${BASE_URL}/api/memories/search?q=MemoryFlow&start=2026-05-01&end=2026-06-01&top_k=5"

# 8. Recent memories
request_get \
  "Recent memories" \
  "${BASE_URL}/api/memories/recent?limit=5"

# 9. Reindex
request_post \
  "Reindex memories" \
  "${BASE_URL}/api/memories/reindex?batch_size=50"

# 10. Search again after reindex
request_get \
  "Semantic search after reindex" \
  "${BASE_URL}/api/memories/search?q=MemoryFlow&top_k=5"

# 11. Reanalyze one memory
request_post \
  "Reanalyze memory ${MEMORY_ID}" \
  "${BASE_URL}/api/memories/${MEMORY_ID}/reanalyze"

echo "========================================"
echo "✅ All requests sent."
echo
echo "Manual checks:"
echo "1. /api/agent/tools should contain ask_memory/search_memory/list_recent/get_timeline"
echo "2. /api/memories/ask should return intent=ask_memory"
echo "3. answer should be natural Chinese"
echo "4. answer should NOT contain Go struct fields like Memory:{}, ContentText, OccurredAt, DeletedAt"
echo "5. /api/agent/tool should be removed or return 404"
echo "========================================"
