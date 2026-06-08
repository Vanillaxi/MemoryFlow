package dispatcher

import "testing"

func TestDispatch(t *testing.T) {
	tests := []struct {
		message  string
		intent   string
		pipeline string
	}{
		{message: "我的 MemoryFlow 最近做到哪了？", intent: IntentProjectProgress, pipeline: PipelineProject},
		{message: "MemoryFlow 还有哪些 issue 没处理？", intent: IntentProjectIssueStatus, pipeline: PipelineProject},
		{message: "MemoryFlow 最近有哪些 PR？", intent: IntentProjectPRStatus, pipeline: PipelineProject},
		{message: "MemoryFlow 有哪些 pull requests 需要 review？", intent: IntentProjectPRStatus, pipeline: PipelineProject},
		{message: "MemoryFlow 有哪些待处理风险？", intent: IntentProjectIssueStatus, pipeline: PipelineProject},
		{message: "帮我查一下 Gin 官方文档怎么用 middleware", intent: IntentExternalKnowledge, pipeline: PipelineKnowledge},
		{message: "搜索 Go 1.26 release notes", intent: IntentExternalKnowledge, pipeline: PipelineKnowledge},
		{message: "帮我总结这个文档：https://example.com/docs", intent: IntentExternalKnowledge, pipeline: PipelineKnowledge},
		{message: "帮我总结这个页面：https://github.com/cloudwego/eino", intent: IntentExternalKnowledge, pipeline: PipelineKnowledge},
		{message: "帮我读取这个页面：http://127.0.0.1:8080/health", intent: IntentExternalKnowledge, pipeline: PipelineKnowledge},
		{message: "MemoryFlow 最近有哪些 release commits？", intent: IntentProjectProgress, pipeline: PipelineProject},
		{message: "帮我总结 MemoryFlow 当前进度，方便开启新聊天", intent: IntentProjectHandoff, pipeline: PipelineProject},
		{message: "生成一份 MemoryFlow 的项目交接摘要", intent: IntentProjectHandoff, pipeline: PipelineProject},
		{message: "总结当前项目状态，给 Codex / ChatGPT 无缝衔接", intent: IntentProjectHandoff, pipeline: PipelineProject},
		{message: "我之前记录了什么记忆？", intent: IntentMemoryQuery, pipeline: PipelineChat},
		{message: "帮我整理一份 codex 交接总结", intent: IntentGeneral, pipeline: PipelineChat},
		{message: "你好", intent: IntentGeneral, pipeline: PipelineChat},
	}

	for _, test := range tests {
		got := Dispatch(test.message)
		if got.Intent != test.intent || got.Pipeline != test.pipeline {
			t.Fatalf("Dispatch(%q) = %#v, want intent=%q pipeline=%q", test.message, got, test.intent, test.pipeline)
		}
	}
}
