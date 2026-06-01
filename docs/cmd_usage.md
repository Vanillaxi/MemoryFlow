# MemoryFlow 命令说明

本文档中的命令默认在 `backend/` 目录执行：

```bash
cd backend
```

## 正式服务

启动 HTTP 服务：

```bash
go run .
```

访问地址：

```text
http://localhost:8080
```

构建服务：

```bash
go build .
```

正式服务入口为 `backend/main.go`。HTTP 路由注册、配置读取、数据库初始化和
bootstrap 依赖组装都通过该入口完成。

服务启动前需要准备 `configs/config.yaml`，并确保 SQLite 路径、模型 API、
Embedding API 和 Milvus 地址可用。

## 调试入口

`backend/cmd/` 只保留两个调试入口：

```text
cmd/
├── chat_cmd/
│   └── main.go
└── knowledge_cmd/
    └── main.go
```

### chat_cmd

`chat_cmd` 用于单独测试用户侧的 `chat_pipeline`：

```bash
go run ./cmd/chat_cmd "最近一周我记录了什么"
go run ./cmd/chat_cmd --debug "和 Eino 有关的记忆有哪些"
go run ./cmd/chat_cmd --debug "总结一下五月份我主要做了什么"
```

不传问题时，命令会使用默认问题。

`--debug` 会输出对话链路的 trace。调试信息中可以看到
`get_current_time`、`query_long_term_memory`、`get_memory_detail`
等工具调用。

`chat_pipeline` 负责：

- 用户侧自然语言问答
- 工具调用
- 记忆搜索与详情查询
- 时间线回答
- 记忆总结

ReAct 是 `chat_pipeline` 内部的 tool-calling 执行策略，不单独提供 cmd。
Summary 也是 `chat_pipeline` 的能力，不单独提供 cmd。

### knowledge_cmd

`knowledge_cmd` 用于单独测试 `knowledge_pipeline`：

```bash
go run ./cmd/knowledge_cmd --batch-size=50
```

`--batch-size` 用于指定每批处理的记忆数量，默认值为 `50`。

`knowledge_pipeline` 负责验证完整的知识入库索引链路：

```text
loader -> transformer -> embedding -> indexer
```

该命令可以用于重建记忆索引，但不负责用户问答、Summary 或 ReAct。

## Workflow 说明

文字、图片和图文混合记忆统一通过 `internal/ai/workflow/memory_analyze`
进行分析。

`memory_analyze` 支持以下输入类型：

- `text`
- `image`
- `mixed`

分析结果包含摘要、标签、情绪和重要度。worker 在分析完成后会继续创建
Embedding 任务，并将向量写入 Milvus。

## 测试与构建

执行完整检查：

```bash
./scripts/test_memoryflow.sh
```

脚本默认运行：

```bash
go test ./...
go build .
go build ./cmd/chat_cmd
go build ./cmd/knowledge_cmd
```

额外测试真实 AI 对话链路：

```bash
RUN_AI=1 ./scripts/test_memoryflow.sh
```

额外测试真实知识索引链路：

```bash
RUN_INDEX=1 ./scripts/test_memoryflow.sh
```

同时测试真实 AI 和索引链路：

```bash
RUN_AI=1 RUN_INDEX=1 ./scripts/test_memoryflow.sh
```

额外检查 Docker 镜像构建：

```bash
RUN_DOCKER=1 ./scripts/test_memoryflow.sh
```

默认测试不会执行 Docker build，避免拖慢日常验证。

## Docker Compose

启动 Docker Compose 服务：

```bash
docker compose up --build -d
```

访问地址：

```text
http://localhost:18080
```

Docker Compose 将宿主机的 `18080` 端口映射到容器内部的 `8080` 端口，避免和本地
`go run .` 默认使用的 `8080` 端口冲突。

排查 Docker 服务日志和外部 API 请求：

```bash
docker compose logs -f memoryflow-backend
curl -s http://localhost:18080/api/agent/tools | jq
curl -s "http://localhost:18080/api/memories/search?q=Docker&debug=true" | jq
```

Docker Compose 默认不注入代理环境变量，容器会直接访问外网。如果容器请求
DashScope 或其他 OpenAI-compatible API 时出现 `EOF`、timeout、`connection reset`，
可以按需在 `backend/.env` 中配置：

```dotenv
HTTP_PROXY=http://host.docker.internal:7897
HTTPS_PROXY=http://host.docker.internal:7897
NO_PROXY=localhost,127.0.0.1,host.docker.internal
MILVUS_ADDRESS=host.docker.internal:19530
```

Clash Verge 端口以实际设置为准，当前示例使用 `7897`。Docker 容器应通过
`host.docker.internal:7897` 访问宿主机代理，不要使用 `127.0.0.1:7897`。

如果返回 `proxyconnect tcp ... connect: connection refused`，请在宿主机代理软件中
启用 `Allow LAN` 或“允许局域网连接”等价选项，确保 Docker VM 可以访问代理端口。
