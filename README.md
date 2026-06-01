# MemoryFlow

MemoryFlow 是一个面向个人长期记忆管理的 AI 后端服务。它可以接收文字和图片记忆，
自动生成摘要、标签、情绪和重要度，并将记忆写入向量索引。用户可以通过自然语言回顾
近期记录、搜索相关记忆，或者总结一段时间内的主要活动。

项目使用 Go 开发，HTTP 服务基于 Gin，结构化数据存储在 SQLite 中，向量检索使用
Milvus。对话链路支持工具调用，模型会根据问题查询长期记忆、读取记忆详情并结合时间
信息组织回答。

## 核心能力

- 记录文字记忆和图片记忆。
- 统一分析 `text`、`image`、`mixed` 三种记忆输入。
- 自动生成摘要、标签、情绪和重要度。
- 使用 Embedding 和 Milvus 建立长期记忆索引。
- 通过自然语言搜索、回顾和总结个人记忆。
- 提供时间线、近期记录、重建索引等 HTTP API。
- 支持调试模式查看 chat pipeline 的工具调用 trace。

## 项目结构

```text
MemoryFlow/
├── backend/
│   ├── main.go                         # 正式 HTTP 服务入口
│   ├── cmd/
│   │   ├── chat_cmd/                   # 调试用户侧问答链路
│   │   └── knowledge_cmd/              # 调试知识入库索引链路
│   ├── configs/
│   │   └── config.example.yaml         # 配置模板
│   ├── internal/
│   │   ├── ai/agent/chat_pipeline/     # 对话、工具调用、总结、时间线回答
│   │   ├── ai/agent/knowledge_pipeline/ # loader -> transformer -> embedding -> indexer
│   │   ├── ai/workflow/memory_analyze/ # 统一记忆分析 workflow
│   │   ├── api/                        # HTTP API
│   │   ├── bootstrap/                  # 服务依赖初始化
│   │   └── task/                       # 异步任务 worker
│   └── scripts/
│       └── test_memoryflow.sh           # 测试和构建检查
└── docs/
    └── cmd_usage.md                     # 命令说明
```

`chat_pipeline` 负责用户侧问答、工具调用、总结和时间线回答。ReAct 是其内部的
tool-calling 执行策略，不单独提供命令。

`knowledge_pipeline` 负责记忆和知识的入库索引。文字、图片和图文混合记忆统一通过
`memory_analyze` workflow 分析。

## 快速开始

### 1. 环境要求

- Go `1.26` 或兼容版本
- 可访问的 Milvus 服务，默认地址为 `localhost:19530`
- 可用的对话模型 API
- 可用的 Embedding API

模型和 Embedding 接口通过 `base_url`、`api_key` 和 `model_name` 配置。

### 2. 创建本地配置

进入后端目录，并从模板创建配置文件：

```bash
cd backend
cp configs/config.example.yaml configs/config.yaml
```

编辑 `configs/config.yaml`：

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  driver: sqlite
  dsn: ../memoryflow-data/data/memoryflow.db

storage:
  upload_dir: ../memoryflow-data/uploads

model:
  base_url: "https://your-model-provider.example/v1"
  api_key: "your-model-api-key"
  model_name: "your-chat-model"

embedding:
  base_url: "https://your-embedding-provider.example/v1"
  api_key: "your-embedding-api-key"
  model_name: "your-embedding-model"
  dim: 1024

milvus:
  address: "localhost:19530"
  collection: "memoryflow_memories"
```

SQLite 数据目录会在首次启动时自动创建。Milvus 需要提前启动，并确保
`embedding.dim` 与实际 Embedding 模型输出维度一致。

### 3. 启动服务

本地启动正式 HTTP 服务：

```bash
cd backend
go run .
```

正式服务入口是 `backend/main.go`，不需要运行 `cmd/` 下的调试入口。

默认监听 `http://localhost:8080`。检查服务是否启动成功：

```bash
curl -s "http://localhost:8080/health"
```

预期返回：

```json
{"status":"ok"}
```

### 4. 写入一条文字记忆

```bash
curl -s -X POST "http://localhost:8080/api/memories/text" \
  -H "Content-Type: application/json" \
  -d '{
    "content_text": "今天整理了 MemoryFlow 的入口和 workflow。",
    "location": "上海",
    "occurred_at": "2026-06-01T10:00:00+08:00"
  }'
```

服务会创建记忆和异步分析任务。worker 完成分析后，会继续生成 Embedding 并写入
Milvus。

### 5. 用自然语言查询记忆

```bash
curl -s "http://localhost:8080/api/memories/ask?q=最近一周我记录了什么&debug=true"
```

查看 chat pipeline 已注册的工具：

```bash
curl -s "http://localhost:8080/api/agent/tools"
```

调试 trace 中可以看到 `get_current_time`、`query_long_term_memory`、
`get_memory_detail` 等工具调用。

### 6. 使用 Docker 启动

构建 backend 镜像：

```bash
cd backend
docker build -t memoryflow-backend .
```

使用 Docker 启动：

```bash
docker run --rm -p 8080:8080 \
  --add-host host.docker.internal:host-gateway \
  -v "$(pwd)/memoryflow-data:/app/memoryflow-data" \
  -v "$(pwd)/configs/config.yaml:/app/configs/config.yaml:ro" \
  memoryflow-backend
```

当前 Docker 配置只启动 backend。Milvus 仍然是外部依赖；如果 Milvus 运行在宿主机，
请在本地 `configs/config.yaml` 中将地址设置为 `host.docker.internal:19530`。

使用 Docker Compose 启动：

```bash
cd backend
docker compose up --build -d
curl -s http://localhost:18080/api/agent/tools | jq
```

Docker Compose 将宿主机的 `18080` 端口映射到容器内部的 `8080` 端口。这样可以避免
与本地开发时运行的 `go run .` 冲突。

停止 Docker Compose 服务：

```bash
docker compose down
```

本地开发服务仍然使用 `http://localhost:8080`，Docker Compose 服务使用
`http://localhost:18080`，容器内部服务端口仍然是 `8080`。

Compose 支持通过本地 `.env` 注入容器环境变量。当前后端的模型 API Key 仍然从本地
`configs/config.yaml` 读取。`.env` 和真实配置都不要提交到仓库。
`memoryflow-data` 是持久化数据目录，不会打入镜像。

### Docker 排查

Docker 容器内的服务应监听 `0.0.0.0:8080`，而不是 `127.0.0.1:8080`。默认配置为：

```yaml
server:
  host: "0.0.0.0"
  port: 8080
```

如果容器已启动但接口无法访问，还需要检查 Milvus 地址。容器内的 `localhost`
指向容器自身；当 Milvus 运行在宿主机时，应在本地 `configs/config.yaml` 中配置：

```yaml
milvus:
  address: "host.docker.internal:19530"
```

Docker Compose 默认通过 `MILVUS_ADDRESS=host.docker.internal:19530` 覆盖该地址。
如果 Milvus 运行在其他位置，可以在本地 `.env` 中设置 `MILVUS_ADDRESS`。

Docker Compose 默认不注入代理环境变量，容器会直接访问外网。如果容器内请求
DashScope 或其他 OpenAI-compatible API 时出现 `EOF`、timeout、`connection reset`，
可以按需在 `backend/.env` 中配置宿主机代理。
Clash Verge 端口以本地实际设置为准，当前示例使用 `7897`：

```dotenv
HTTP_PROXY=http://host.docker.internal:7897
HTTPS_PROXY=http://host.docker.internal:7897
NO_PROXY=localhost,127.0.0.1,host.docker.internal
MILVUS_ADDRESS=host.docker.internal:19530
```

Docker 容器访问宿主机代理应使用 `host.docker.internal:7897`，不要在容器内使用
`127.0.0.1:7897`。`.env` 包含本地运行配置，不要提交到仓库。

如果进一步出现 `proxyconnect tcp ... connect: connection refused`，说明容器已经
尝试连接宿主机代理，但代理端口没有对 Docker VM 开放。请在 Clash Verge 中启用
`Allow LAN` 或“允许局域网连接”等价选项，并确认代理监听地址不是仅限 `127.0.0.1`。

## 调试命令

`backend/cmd/` 只保留两个调试入口。

单独测试用户侧对话链路：

```bash
go run ./cmd/chat_cmd "最近一周我记录了什么"
go run ./cmd/chat_cmd --debug "和 Eino 有关的记忆有哪些"
go run ./cmd/chat_cmd --debug "总结一下五月份我主要做了什么"
```

单独测试知识入库索引链路：

```bash
go run ./cmd/knowledge_cmd --batch-size=50
```

## 测试与构建

在 `backend/` 目录运行完整检查：

```bash
./scripts/test_memoryflow.sh
```

脚本会依次执行：

```bash
go test ./...
go build .
go build ./cmd/chat_cmd
go build ./cmd/knowledge_cmd
```

需要额外测试真实 AI 链路时：

```bash
RUN_AI=1 ./scripts/test_memoryflow.sh
```

需要额外测试真实索引链路时：

```bash
RUN_INDEX=1 ./scripts/test_memoryflow.sh
```

需要额外检查 Docker 镜像构建时：

```bash
RUN_DOCKER=1 ./scripts/test_memoryflow.sh
```

## 常用 API

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/health` | 服务健康检查 |
| `POST` | `/api/memories/text` | 创建文字记忆 |
| `POST` | `/api/memories/image` | 创建图片记忆 |
| `GET` | `/api/memories/recent` | 查看近期记忆 |
| `GET` | `/api/memories/search` | 搜索相关记忆 |
| `GET` | `/api/memories/summary` | 总结指定时间范围内的记忆 |
| `GET` | `/api/timeline` | 查看时间线 |
| `GET` | `/api/memories/ask` | 使用自然语言查询记忆 |
| `POST` | `/api/memories/:id/reanalyze` | 重新分析指定记忆 |
| `POST` | `/api/memories/reindex` | 重建记忆索引 |
| `GET` | `/api/agent/tools` | 查看 chat pipeline 工具列表 |

更多命令说明见 [docs/cmd_usage.md](docs/cmd_usage.md)。
