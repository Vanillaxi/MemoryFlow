package project_pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"memoryflow/internal/ai/tools"
	githubtool "memoryflow/internal/ai/tools/github"
	memorytool "memoryflow/internal/ai/tools/memory"
	systemtool "memoryflow/internal/ai/tools/system"
	webtool "memoryflow/internal/ai/tools/web"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type executionScope struct {
	repository string
	days       int
	limit      int
	recorder   *toolCallRecorder
}

type executionScopeKey struct{}

type toolCallRecorder struct {
	mu    sync.Mutex
	calls []ToolCallLog
}

func (r *toolCallRecorder) add(call ToolCallLog) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, call)
}

func (r *toolCallRecorder) snapshot() []ToolCallLog {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ToolCallLog(nil), r.calls...)
}

type Adapter struct {
	tool       tools.Tool
	parameters map[string]*schema.ParameterInfo
}

func NewAdapter(currentTool tools.Tool) *Adapter {
	return &Adapter{tool: currentTool, parameters: parametersFor(currentTool.Name())}
}

func (a *Adapter) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        a.tool.Name(),
		Desc:        a.tool.Description(),
		ParamsOneOf: schema.NewParamsOneOfByParams(a.parameters),
	}, nil
}

func (a *Adapter) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...einotool.Option) (string, error) {
	args := make(map[string]any)
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("%s: invalid tool arguments json: %w", a.tool.Name(), err)
	}
	args = sanitizeToolArgs(args)
	scope, _ := ctx.Value(executionScopeKey{}).(*executionScope)
	if scope != nil && isProjectGitHubTool(a.tool.Name()) {
		args["repository"] = scope.repository
		if scope.days > 0 && a.tool.Name() != githubtool.ToolGetPullRequests {
			args["days"] = scope.days
		}
		if scope.limit > 0 {
			args["limit"] = scope.limit
		}
	}

	result, err := a.tool.Call(ctx, args)
	if scope != nil && scope.recorder != nil {
		call := ToolCallLog{Name: a.tool.Name(), Args: args, Result: result}
		if err != nil {
			call.Error = err.Error()
			call.Result = toolErrorResult(err)
		}
		scope.recorder.add(call)
	}
	if err != nil {
		return toolErrorResult(err), nil
	}
	return result, nil
}

func isProjectGitHubTool(name string) bool {
	switch name {
	case githubtool.ToolGetRecentCommits, githubtool.ToolGetRecentIssues, githubtool.ToolGetPullRequests:
		return true
	default:
		return false
	}
}

func toolErrorResult(err error) string {
	bytes, marshalErr := json.Marshal(map[string]string{"error": err.Error()})
	if marshalErr != nil {
		return `{"error":"tool failed"}`
	}
	return string(bytes)
}

func sanitizeToolArgs(args map[string]any) map[string]any {
	sanitized := make(map[string]any, len(args))
	for key, value := range args {
		switch strings.ToLower(key) {
		case "token", "api_key", "apikey", "authorization", "secret":
			continue
		}
		if nested, ok := value.(map[string]any); ok {
			sanitized[key] = sanitizeToolArgs(nested)
			continue
		}
		sanitized[key] = value
	}
	return sanitized
}

func parametersFor(name string) map[string]*schema.ParameterInfo {
	switch name {
	case githubtool.ToolGetRecentCommits:
		return map[string]*schema.ParameterInfo{
			"repository": {Type: schema.String, Desc: "仓库 owner/repo。由当前项目上下文提供。", Required: true},
			"limit":      {Type: schema.Integer, Desc: "返回数量，可选，最大 20。"},
			"days":       {Type: schema.Integer, Desc: "最近天数，可选。"},
			"since":      {Type: schema.String, Desc: "RFC3339 时间，可选；优先于 days。"},
		}
	case githubtool.ToolGetRecentIssues:
		return map[string]*schema.ParameterInfo{
			"repository": {Type: schema.String, Desc: "仓库 owner/repo。由当前项目上下文提供。", Required: true},
			"state":      {Type: schema.String, Desc: "issue 状态：open/closed/all，可选，默认 open。"},
			"limit":      {Type: schema.Integer, Desc: "返回数量，可选，最大 20。"},
			"days":       {Type: schema.Integer, Desc: "最近天数，可选。"},
			"since":      {Type: schema.String, Desc: "RFC3339 时间，可选；优先于 days。"},
			"labels":     {Type: schema.String, Desc: "逗号分隔 labels，可选。"},
			"sort":       {Type: schema.String, Desc: "created/updated/comments，可选，默认 updated。"},
			"direction":  {Type: schema.String, Desc: "asc/desc，可选，默认 desc。"},
		}
	case githubtool.ToolGetPullRequests:
		return map[string]*schema.ParameterInfo{
			"repository": {Type: schema.String, Desc: "仓库 owner/repo。由当前项目上下文提供。", Required: true},
			"state":      {Type: schema.String, Desc: "PR 状态：open/closed/all，可选，默认 open。"},
			"limit":      {Type: schema.Integer, Desc: "返回数量，可选，最大 20。"},
			"sort":       {Type: schema.String, Desc: "created/updated/popularity/long-running，可选，默认 updated。"},
			"direction":  {Type: schema.String, Desc: "asc/desc，可选，默认 desc。"},
			"base":       {Type: schema.String, Desc: "目标分支，可选。"},
			"head":       {Type: schema.String, Desc: "来源分支或 user:branch，可选。"},
		}
	case memorytool.ToolQueryLongTermMemory:
		return map[string]*schema.ParameterInfo{
			"query": {Type: schema.String, Desc: "自然语言查询。semantic 模式必填。"},
			"from":  {Type: schema.String, Desc: "开始日期 YYYY-MM-DD，可选。"},
			"to":    {Type: schema.String, Desc: "结束日期 YYYY-MM-DD，可选。"},
			"mode":  {Type: schema.String, Desc: "semantic/timeline/aggregate，可选。"},
			"limit": {Type: schema.Integer, Desc: "返回数量，可选。"},
		}
	case systemtool.ToolGetCurrentTime:
		return nil
	case webtool.ToolWebSearch:
		return map[string]*schema.ParameterInfo{
			"query": {Type: schema.String, Desc: "搜索查询。", Required: true},
			"limit": {Type: schema.Integer, Desc: "返回数量，可选，默认 5，最大 10。"},
		}
	case webtool.ToolWebFetch:
		return map[string]*schema.ParameterInfo{
			"url": {Type: schema.String, Desc: "要读取的 http/https 公网 URL。", Required: true},
		}
	default:
		return nil
	}
}
