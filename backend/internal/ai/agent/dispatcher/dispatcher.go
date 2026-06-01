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
	if containsAny(normalized, "记忆", "我之前", "最近干了啥") {
		return Decision{Intent: IntentMemoryQuery, Pipeline: PipelineChat}
	}
	if containsAny(normalized, "最近", "做到哪", "项目", "进展", "commit", "github", "仓库") {
		return Decision{Intent: IntentProjectProgress, Pipeline: PipelineProject}
	}
	return Decision{Intent: IntentGeneral, Pipeline: PipelineChat}
}

func containsAny(message string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(message, keyword) {
			return true
		}
	}
	return false
}
