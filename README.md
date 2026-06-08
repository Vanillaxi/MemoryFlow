# MemoryFlow

MemoryFlow 是一个本地优先的个人长期记忆 Agent。它可以接收文字和图片记忆，
通过 LLM 自动生成摘要、标签、情绪和重要度，将分析结果写入 SQLite，并将长期记忆
写入向量索引。用户可以通过自然语言回顾近期记录、搜索相关记忆，或者总结一段时间内
的主要活动。

项目使用 Go 开发，HTTP 服务基于 Gin，结构化数据存储在 SQLite 中，向量检索使用
Milvus。Milvus 不可用时，服务会降级到 DisabledStore，HTTP 服务仍可启动。

MemoryFlow 现在也支持 Project Agent 能力，可以结合项目上下文、GitHub commits、
issues 和 PR，回答项目进展、未处理 issue、最近 PR、下一步建议等问题。
同时新增了只读 Web Tool MVP，用于在外部公开网页中补充资料和文档信息。
Project Agent 还支持 Project Handoff Summary 输出能力，适合开启新聊天、交接给
Codex / ChatGPT，或快速恢复项目上下文。

## 核心能力

- 记录文字记忆和图片记忆，通过自然语言搜索、回顾和总结个人记忆。统一分析 `text`、`image`、`mixed` 三种记忆输入， 并由LLM 自动生成摘要、标签、情绪和重要度。
- GitHub 只读工具：recent commits、issues、pull requests，Eino ReAct Project Agent，支持项目上下文问答。
- Web / External Knowledge 只读工具：`web_search` 和 `web_fetch`，用于搜索公开资料和读取公开网页。当前不会登录、点击、提交表单或写入网页，也不会向工具注入 GitHub token、LLM API key 等敏感信息。
- Project Handoff Summary：这不是一个新 Tool，而是 `project_pipeline` 的结构化输出能力；它基于 GitHub Tool、Memory Tool、Web Tool 等只读证据源生成项目交接摘要。


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
│   │   ├── ai/agent/project_pipeline/  # Eino ReAct 项目上下文 Agent
│   │   ├── ai/tools/github/            # GitHub 只读工具
│   │   ├── ai/tools/web/               # Web 只读搜索和网页读取工具
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
`memory_analyze` workflow 分析。用户询问官方文档、最新资料、API、version、
release 或“查一下/搜索/怎么用”等外部信息时，也会通过只读 Web Tool MVP 补充公开
网页信息。

`project_pipeline` 负责项目上下文问答。它会先解析当前项目，再通过只读 GitHub 工具
查询 commits、issues 和 pull requests，并在响应中返回工具调用证据。

Project Handoff Summary 是 `project_pipeline` 的一种输出模式，不会新增
`HandoffTool`，也不会写 GitHub、创建 issue、创建 TodoTool 或自动保存到 memory。
当用户询问“总结 MemoryFlow 当前进度，方便开启新聊天”“生成项目交接摘要”“给
Codex / ChatGPT 无缝衔接”或“生成项目上下文包”时，Project Agent 会基于当前项目
信息、recent commits、open issues、recent PR、长期记忆和当前时间生成结构化
Markdown 摘要。它适合新对话启动、项目交接和快速恢复上下文。

Web Tool / External Knowledge 目前只包含只读能力：

- `web_search`：搜索公开资料，第一版采用 provider 接口占位；未配置真实搜索 provider 时会返回 `ErrWebSearchProviderNotConfigured`。
- `web_fetch`：读取用户提供的 `http` / `https` 公网 URL，拒绝 `file://`、localhost、回环地址和内网 IP，限制超时、响应正文大小和最终 content 长度。
- 不登录、不点击、不提交表单、不写网页，只用于补充外部公开信息。
- 网页内容会被视为不可信外部数据；如果页面要求 Agent 忽略已有指令、泄露密钥或输出系统提示，会被当作恶意或无关内容处理。
- Web Tool evidence 会返回网页来源信息，包括 title、url、source/domain、fetched_at 和 content_preview，便于前端和调试确认回答依据。

## 快速开始

### 1. 环境要求

- Go `1.26` 或兼容版本
- 可访问的 Milvus 服务，默认地址为 `localhost:19530`
- 可用的对话模型 API
- 可用的 Embedding API
- 可选的 GitHub fine-grained token，用于 Project Agent 查询 commits、issues 和 PR

模型、Embedding 和 GitHub token 都从本地 `configs/config.yaml` 读取。

### 2. 克隆项目

```bash
git clone https://github.com/Vanillaxi/MemoryFlow.git
cd MemoryFlow
```

### 3. 创建本地配置

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

github:
  token: "your github token"
  default_limit: 10
  default_days: 7
```

SQLite 数据目录会在首次启动时自动创建。Milvus 需要提前启动，并确保
`embedding.dim` 与实际 Embedding 模型输出维度一致。`github.token` 可以为空；为空时，
Project Agent 的 GitHub 查询会因为缺少 token 返回清晰错误，但不会把 token 暴露给模型。

### 4. 启动服务

本地启动正式 HTTP 服务：

```bash
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

### 5. 创建项目

Project Agent 需要先通过 Project 表建立项目与 GitHub 仓库的映射：

```bash
curl -X POST http://localhost:8080/projects \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MemoryFlow",
    "description": "本地优先的个人长期记忆 Agent",
    "repo_owner": "Vanillaxi",
    "repo_name": "MemoryFlow",
    "repo_url": "https://github.com/Vanillaxi/MemoryFlow",
    "tech_stack": "Go, Gin, SQLite, Milvus, Eino",
    "status": "active"
  }'
```

### 6. 调用 Project Agent

查询项目进展：

```bash
curl -X POST http://localhost:8080/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"我的 MemoryFlow 最近做到哪了？","days":7,"limit":5}' | jq
```

查询未处理 issue：

```bash
curl -X POST http://localhost:8080/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"MemoryFlow 还有哪些 issue 没处理？","days":30,"limit":10}' | jq
```

查询最近 PR：

```bash
curl -X POST http://localhost:8080/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"MemoryFlow 最近有哪些 PR？","days":30,"limit":10}' | jq
```

生成项目交接摘要：

```bash
curl -X POST http://localhost:8080/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"帮我总结 MemoryFlow 当前进度，方便开启新聊天","days":30,"limit":10}' | jq
```

Project Agent 响应会包含 `pipeline`、`intent`、`used_tools`、`evidence` 和
`raw_tool_calls`，便于确认问题是否路由到正确工具。

查询官方文档或外部资料：

```bash
curl -X POST http://localhost:8080/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"帮我查一下 Gin 官方文档怎么用 middleware"}' | jq
```

如果问题中包含公开 URL，会优先使用 `web_fetch`：

```bash
curl -X POST http://localhost:8080/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"读取一下 https://example.com 这页资料"}' | jq
```

也可以显式强制走 Project Agent：

```bash
curl -X POST http://localhost:8080/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"message":"MemoryFlow 最近做到哪了？","pipeline":"project"}' | jq
```

### 7. 写入一条文字记忆

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

### 8. 用自然语言查询记忆

```bash
curl -s "http://localhost:8080/api/memories/ask?q=最近一周我记录了什么&debug=true"
```

查看 chat pipeline 已注册的工具：

```bash
curl -s "http://localhost:8080/api/agent/tools"
```

调试 trace 中可以看到 `get_current_time`、`query_long_term_memory`、
`get_memory_detail` 等工具调用。

### 9. 使用 Docker 启动

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

## 配置说明

配置文件默认读取 `backend/configs/config.yaml`。仓库只提交
`backend/configs/config.example.yaml` 作为模板，不要提交真实 token、API key 或本地
`config.yaml`。

常用配置项：

| 配置 | 用途 |
| --- | --- |
| `server.port` | HTTP 服务端口，默认 `8080` |
| `database.dsn` | SQLite 数据库路径，默认 `../memoryflow-data/data/memoryflow.db` |
| `storage.upload_dir` | 图片和上传文件目录，默认 `../memoryflow-data/uploads` |
| `model.base_url` | OpenAI-compatible 对话模型接口地址 |
| `model.api_key` | 对话模型 API key，本地配置，不要提交 |
| `model.model_name` | 对话模型名称 |
| `embedding.base_url` | Embedding 模型接口地址 |
| `embedding.api_key` | Embedding API key，本地配置，不要提交 |
| `embedding.model_name` | Embedding 模型名称 |
| `embedding.dim` | Embedding 维度，必须和模型输出一致 |
| `milvus.address` | Milvus 地址，本地默认 `localhost:19530` |
| `milvus.collection` | Milvus collection 名称 |
| `github.token` | GitHub fine-grained token，只用于只读查询 |
| `github.default_limit` | GitHub 工具默认返回数量 |
| `github.default_days` | GitHub commits/issues 默认查询天数 |

GitHub token 建议使用 fine-grained token，并只授予 read-only 权限：

- Contents: Read-only
- Issues: Read-only
- Pull requests: Read-only
- Metadata: Read-only

## 安全设计

- GitHub tools 全部只读，只查询 commits、issues 和 pull requests。
- 不支持 merge PR、关闭 issue、评论 issue、创建 release 等写操作。
- GitHub 原始数据不落 SQLite，只作为实时证据源参与当次回答。
- GitHub token 从本地配置读取，不进入 LLM prompt，不打印到日志。
- Project Agent prompt 明确禁止修改 GitHub 仓库或泄露 token、API key 等敏感信息。

## 降级设计

- Milvus 可用时，启动 `MilvusStore` 并用于向量检索和向量写入。
- Milvus 不可用时，启动 `DisabledStore`。
- `DisabledStore` 会让向量检索和插入返回清晰错误，但不会阻塞 HTTP 服务启动。
- GitHub Project Agent 不依赖 Milvus。Milvus 挂了，仍然可以基于 SQLite 中的项目映射
  查询 GitHub commits、issues 和 pull requests。

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

## Roadmap

- HandoffTool：生成给 ChatGPT / Codex 的项目交接摘要。
- SaveProjectSummaryToMemory：把项目总结保存为长期记忆。
- Web / Docs Tool：只读查询官方文档。
- Milvus docker-compose 整理。
- 简单前端页面。

更多命令说明见 [docs/cmd_usage.md](docs/cmd_usage.md)。
