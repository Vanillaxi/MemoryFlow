package mcp

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"memoryflow/internal/ai/agent"
	"memoryflow/internal/ai/agent/dispatcher"
	"memoryflow/internal/ai/agent/project_pipeline"
	"memoryflow/internal/ai/tools"
	githubtool "memoryflow/internal/ai/tools/github"
	memorytool "memoryflow/internal/ai/tools/memory"
	webtool "memoryflow/internal/ai/tools/web"
	"memoryflow/internal/domain/model"

	"github.com/gin-gonic/gin"
)

const (
	MethodProjectHandoffSummary  = "project_handoff_summary"
	MethodProjectProgressSummary = "project_progress_summary"
	MethodProjectIssueStatus     = "project_issue_status"
	MethodProjectPRStatus        = "project_pr_status"
	MethodQueryLongTermMemory    = memorytool.ToolQueryLongTermMemory
	MethodGetMemoryDetail        = memorytool.ToolGetMemoryDetail
	MethodAggregateMemory        = memorytool.ToolAggregateMemory
	MethodRecentMemories         = "recent_memories"
	MethodTimeline               = "timeline"
	MethodProjectMemories        = "project_memories"
	MethodImageMemories          = "image_memories"
	MethodWebFetch               = webtool.ToolWebFetch
	MethodWebSearch              = webtool.ToolWebSearch
	MethodGetRecentCommits       = githubtool.ToolGetRecentCommits
	MethodGetRecentIssues        = githubtool.ToolGetRecentIssues
	MethodGetPullRequests        = githubtool.ToolGetPullRequests

	PipelineMCPToolRegistry = "mcp_tool_registry"
)

var readOnlyToolMethods = map[string]string{
	MethodQueryLongTermMemory: memorytool.ToolQueryLongTermMemory,
	MethodGetMemoryDetail:     memorytool.ToolGetMemoryDetail,
	MethodAggregateMemory:     memorytool.ToolAggregateMemory,
	MethodRecentMemories:      memorytool.ToolQueryLongTermMemory,
	MethodTimeline:            memorytool.ToolQueryLongTermMemory,
	MethodProjectMemories:     memorytool.ToolQueryLongTermMemory,
	MethodImageMemories:       memorytool.ToolQueryLongTermMemory,
	MethodWebFetch:            webtool.ToolWebFetch,
	MethodWebSearch:           webtool.ToolWebSearch,
	MethodGetRecentCommits:    githubtool.ToolGetRecentCommits,
	MethodGetRecentIssues:     githubtool.ToolGetRecentIssues,
	MethodGetPullRequests:     githubtool.ToolGetPullRequests,
}

type Server struct {
	agent    *agent.Agent
	registry *tools.ToolRegistry
	token    string
}

type Request struct {
	Method    string         `json:"method,omitempty"`
	Message   string         `json:"message"`
	Intent    string         `json:"intent,omitempty"`
	ProjectID *uint          `json:"project_id,omitempty"`
	Days      int            `json:"days,omitempty"`
	Limit     int            `json:"limit,omitempty"`
	Token     string         `json:"token,omitempty"`
	MemoryID  uint           `json:"memory_id,omitempty"`
	Params    map[string]any `json:"params,omitempty"`
}

type Response struct {
	Answer       string                         `json:"answer"`
	Intent       string                         `json:"intent,omitempty"`
	UsedTools    []string                       `json:"used_tools"`
	Evidence     []project_pipeline.Evidence    `json:"evidence"`
	RawToolCalls []project_pipeline.ToolCallLog `json:"raw_tool_calls"`
	Pipeline     string                         `json:"pipeline"`
	Project      *ProjectRef                    `json:"project,omitempty"`
}

type ProjectRef struct {
	ID   uint   `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

func NewServer(currentAgent *agent.Agent, registry *tools.ToolRegistry, token string) *Server {
	return &Server{
		agent:    currentAgent,
		registry: registry,
		token:    strings.TrimSpace(token),
	}
}

func (s *Server) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req Request
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := s.authenticate(c, req.Token); err != nil {
			status := http.StatusUnauthorized
			if err == errTokenNotConfigured {
				status = http.StatusServiceUnavailable
			}
			c.JSON(status, gin.H{"error": err.Error()})
			return
		}
		resp, err := s.Invoke(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, resp)
	}
}

var errTokenNotConfigured = fmt.Errorf("mcp token is not configured")

func (s *Server) authenticate(c *gin.Context, token string) error {
	if s == nil || s.token == "" {
		return errTokenNotConfigured
	}
	if token == "" {
		token = bearerToken(c.GetHeader("Authorization"))
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.token)) != 1 {
		return fmt.Errorf("invalid mcp token")
	}
	return nil
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[len("bearer "):])
	}
	return ""
}

func (s *Server) Invoke(ctx context.Context, req Request) (*Response, error) {
	method := normalizeMethod(req)
	if isProjectMethod(method) {
		return s.invokeProject(ctx, method, req)
	}
	toolName, ok := readOnlyToolMethods[method]
	if !ok {
		return nil, fmt.Errorf("unsupported read-only mcp method %q", method)
	}
	return s.invokeTool(ctx, method, toolName, req)
}

func normalizeMethod(req Request) string {
	if method := strings.TrimSpace(req.Method); method != "" {
		return normalizeIntentMethod(method)
	}
	return normalizeIntentMethod(req.Intent)
}

func normalizeIntentMethod(value string) string {
	switch strings.TrimSpace(value) {
	case dispatcher.IntentProjectHandoff, "handoff":
		return MethodProjectHandoffSummary
	case dispatcher.IntentProjectProgress:
		return MethodProjectProgressSummary
	case dispatcher.IntentProjectIssueStatus:
		return MethodProjectIssueStatus
	case dispatcher.IntentProjectPRStatus:
		return MethodProjectPRStatus
	case dispatcher.IntentMemoryQuery:
		return MethodQueryLongTermMemory
	case dispatcher.IntentExternalKnowledge:
		return MethodWebSearch
	default:
		return strings.TrimSpace(value)
	}
}

func isProjectMethod(method string) bool {
	switch method {
	case MethodProjectHandoffSummary, MethodProjectProgressSummary, MethodProjectIssueStatus, MethodProjectPRStatus:
		return true
	default:
		return false
	}
}

func (s *Server) invokeProject(ctx context.Context, method string, req Request) (*Response, error) {
	if s == nil || s.agent == nil {
		return nil, fmt.Errorf("agent is not initialized")
	}
	intent := projectIntent(method)
	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = defaultProjectMessage(method)
	}
	output, err := s.agent.Chat(ctx, agent.ChatInput{
		Message:   message,
		Intent:    intent,
		ProjectID: req.ProjectID,
		Days:      req.Days,
		Limit:     req.Limit,
		Pipeline:  "project",
	})
	if err != nil {
		return nil, err
	}
	return fromChatOutput(output), nil
}

func projectIntent(method string) string {
	switch method {
	case MethodProjectHandoffSummary:
		return dispatcher.IntentProjectHandoff
	case MethodProjectIssueStatus:
		return dispatcher.IntentProjectIssueStatus
	case MethodProjectPRStatus:
		return dispatcher.IntentProjectPRStatus
	default:
		return dispatcher.IntentProjectProgress
	}
}

func defaultProjectMessage(method string) string {
	switch method {
	case MethodProjectHandoffSummary:
		return "生成当前项目的 Project Handoff Summary"
	case MethodProjectIssueStatus:
		return "总结当前项目 issue 状态"
	case MethodProjectPRStatus:
		return "总结当前项目 PR 状态"
	default:
		return "总结当前项目进展"
	}
}

func fromChatOutput(output *agent.ChatOutput) *Response {
	if output == nil {
		return &Response{}
	}
	return &Response{
		Answer:       output.Answer,
		Intent:       output.Intent,
		UsedTools:    output.UsedTools,
		Evidence:     sanitizeEvidence(output.Evidence),
		RawToolCalls: sanitizeToolCalls(output.RawToolCalls),
		Pipeline:     output.Pipeline,
		Project:      projectRef(output.Project),
	}
}

func projectRef(project *model.Project) *ProjectRef {
	if project == nil {
		return nil
	}
	return &ProjectRef{ID: project.ID, Name: project.Name}
}

func (s *Server) invokeTool(ctx context.Context, method string, toolName string, req Request) (*Response, error) {
	if s == nil || s.registry == nil {
		return nil, fmt.Errorf("tool registry is not initialized")
	}
	currentTool, ok := s.registry.Get(toolName)
	if !ok {
		return nil, fmt.Errorf("read-only tool %q is not registered", toolName)
	}
	args := buildToolArgs(method, req)
	result, err := currentTool.Call(ctx, args)
	call := project_pipeline.ToolCallLog{Name: toolName, Args: sanitizeArgs(args)}
	if err != nil {
		call.Error = err.Error()
		return &Response{
			Answer:       err.Error(),
			Intent:       method,
			UsedTools:    []string{toolName},
			Evidence:     []project_pipeline.Evidence{{Source: toolName, Detail: err.Error()}},
			RawToolCalls: []project_pipeline.ToolCallLog{call},
			Pipeline:     PipelineMCPToolRegistry,
			Project:      projectIDRef(req.ProjectID),
		}, nil
	}
	sanitizedResult := sanitizeJSON(result)
	call.Result = sanitizedResult
	return &Response{
		Answer:       toolAnswer(method, sanitizedResult),
		Intent:       method,
		UsedTools:    []string{toolName},
		Evidence:     []project_pipeline.Evidence{{Source: toolName, Detail: sanitizedResult}},
		RawToolCalls: []project_pipeline.ToolCallLog{call},
		Pipeline:     PipelineMCPToolRegistry,
		Project:      projectIDRef(req.ProjectID),
	}, nil
}

func buildToolArgs(method string, req Request) map[string]any {
	args := make(map[string]any, len(req.Params)+6)
	for key, value := range req.Params {
		if isSecretKey(key) {
			continue
		}
		args[key] = value
	}
	if req.Limit > 0 {
		args["limit"] = req.Limit
	}
	applyDateWindow(args, req.Days, method)
	switch method {
	case MethodQueryLongTermMemory:
		args["query"] = firstNonEmpty(stringArg(args, "query"), req.Message)
	case MethodRecentMemories, MethodTimeline:
		args["mode"] = memorytool.ModeTimeline
	case MethodProjectMemories:
		args["query"] = firstNonEmpty(stringArg(args, "query"), req.Message)
		if args["query"] == "" {
			args["query"] = "项目记忆"
		}
		args["mode"] = memorytool.ModeSemantic
	case MethodImageMemories:
		args["query"] = firstNonEmpty(stringArg(args, "query"), req.Message)
		if args["query"] == "" {
			args["query"] = "图片记忆 image memory"
		}
		args["mode"] = memorytool.ModeSemantic
	case MethodGetMemoryDetail:
		if req.MemoryID > 0 {
			args["memory_id"] = req.MemoryID
		}
	case MethodAggregateMemory:
	case MethodWebSearch:
		args["query"] = firstNonEmpty(stringArg(args, "query"), req.Message)
	case MethodWebFetch:
		if url := firstNonEmpty(stringArg(args, "url"), firstURL(req.Message)); url != "" {
			args["url"] = url
		}
	}
	return args
}

func applyDateWindow(args map[string]any, days int, method string) {
	if days <= 0 || !isMemoryMethod(method) || stringArg(args, "from") != "" || stringArg(args, "to") != "" {
		return
	}
	now := time.Now()
	args["from"] = now.AddDate(0, 0, -days).Format("2006-01-02")
	args["to"] = now.Format("2006-01-02")
}

func isMemoryMethod(method string) bool {
	switch method {
	case MethodQueryLongTermMemory, MethodRecentMemories, MethodTimeline, MethodProjectMemories, MethodImageMemories, MethodAggregateMemory:
		return true
	default:
		return false
	}
}

func toolAnswer(method string, result string) string {
	switch method {
	case MethodWebFetch:
		return "已读取网页并返回只读证据。"
	case MethodWebSearch:
		return "已执行 Web 搜索并返回只读证据。"
	case MethodGetRecentCommits, MethodGetRecentIssues, MethodGetPullRequests:
		return "已查询 GitHub 只读状态并返回证据。"
	case MethodGetMemoryDetail:
		return "已查询长期记忆详情并返回安全摘要。"
	case MethodAggregateMemory:
		return "已聚合长期记忆并返回证据。"
	default:
		return "已查询长期记忆并返回证据。"
	}
}

func projectIDRef(projectID *uint) *ProjectRef {
	if projectID == nil {
		return nil
	}
	return &ProjectRef{ID: *projectID}
}

func sanitizeEvidence(items []project_pipeline.Evidence) []project_pipeline.Evidence {
	out := make([]project_pipeline.Evidence, 0, len(items))
	for _, item := range items {
		item.Detail = sanitizeJSON(item.Detail)
		out = append(out, item)
	}
	return out
}

func sanitizeToolCalls(calls []project_pipeline.ToolCallLog) []project_pipeline.ToolCallLog {
	out := make([]project_pipeline.ToolCallLog, 0, len(calls))
	for _, call := range calls {
		call.Args = sanitizeArgs(call.Args)
		call.Result = sanitizeJSON(call.Result)
		out = append(out, call)
	}
	return out
}

func sanitizeArgs(args map[string]any) map[string]any {
	if len(args) == 0 {
		return args
	}
	copied := make(map[string]any, len(args))
	for key, value := range args {
		if isSecretKey(key) {
			continue
		}
		copied[key] = sanitizeValue(value)
	}
	return copied
}

func sanitizeJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return raw
	}
	value = sanitizeValue(value)
	bytes, err := json.Marshal(value)
	if err != nil {
		return raw
	}
	return string(bytes)
}

func sanitizeValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return sanitizeMap(typed)
	case []any:
		for i, item := range typed {
			typed[i] = sanitizeValue(item)
		}
		return typed
	default:
		return value
	}
}

func sanitizeMap(item map[string]any) map[string]any {
	for key, value := range item {
		item[key] = sanitizeValue(value)
	}
	if _, ok := item["image_url"]; ok {
		delete(item, "content_text")
		if _, ok := item["memory_id"]; !ok {
			if id, ok := item["id"]; ok {
				item["memory_id"] = id
			}
		}
		if _, ok := item["caption"]; !ok {
			if summary, ok := item["summary"]; ok {
				item["caption"] = summary
			}
		}
	}
	for key := range item {
		if isSecretKey(key) {
			delete(item, key)
		}
	}
	return item
}

func isSecretKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "token", "api_key", "apikey", "authorization", "password":
		return true
	default:
		return false
	}
}

func firstURL(message string) string {
	for _, field := range strings.Fields(message) {
		trimmed := strings.Trim(field, " \t\r\n\"'<>，。")
		if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
			return trimmed
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func stringArg(args map[string]any, key string) string {
	value, _ := args[key].(string)
	return strings.TrimSpace(value)
}
