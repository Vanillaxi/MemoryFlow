package dispatcher

import "strings"

type Decision struct {
	Intent   string
	Pipeline string
}

func Dispatch(message string) Decision {
	normalized := strings.ToLower(strings.TrimSpace(message))

	if containsAny(normalized, "总结", "开启新聊天", "交接", "handoff", "codex") {
		return Decision{Intent: IntentHandoff, Pipeline: PipelineProject}
	}
	if isProjectIssueQuestion(normalized) {
		return Decision{Intent: IntentProjectIssueStatus, Pipeline: PipelineProject}
	}
	if isProjectPRQuestion(normalized) {
		return Decision{Intent: IntentProjectPRStatus, Pipeline: PipelineProject}
	}
	if isProjectProgressQuestion(normalized) || containsAny(normalized, "github", "仓库", "changelog", "release") {
		return Decision{Intent: IntentProjectProgress, Pipeline: PipelineProject}
	}
	if containsAny(normalized, "记忆", "我之前", "最近干了啥") {
		return Decision{Intent: IntentMemoryQuery, Pipeline: PipelineChat}
	}
	return Decision{Intent: IntentGeneral, Pipeline: PipelineChat}
}

func ProjectIntent(message string) string {
	normalized := strings.ToLower(strings.TrimSpace(message))
	switch {
	case isProjectIssueQuestion(normalized):
		return IntentProjectIssueStatus
	case isProjectPRQuestion(normalized):
		return IntentProjectPRStatus
	default:
		return IntentProjectProgress
	}
}

func isProjectIssueQuestion(message string) bool {
	return containsAny(message, "issue", "issues", "未处理", "待处理", "bug", "风险", "阻塞")
}

func isProjectPRQuestion(message string) bool {
	return containsAny(message, "pull request", "pull requests", "合并", "review") || containsToken(message, "pr")
}

func isProjectProgressQuestion(message string) bool {
	return containsAny(message, "commit", "commits", "项目进展", "做到哪", "进展", "最近")
}

func containsAny(message string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(message, keyword) {
			return true
		}
	}
	return false
}

func containsToken(message string, token string) bool {
	fields := strings.FieldsFunc(message, func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})
	for _, field := range fields {
		if field == token {
			return true
		}
	}
	return false
}
