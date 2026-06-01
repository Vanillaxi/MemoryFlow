package dispatcher

import "testing"

func TestDispatch(t *testing.T) {
	tests := []struct {
		message  string
		intent   string
		pipeline string
	}{
		{message: "我的 MemoryFlow 最近做到哪了？", intent: IntentProjectProgress, pipeline: PipelineProject},
		{message: "我之前记录了什么记忆？", intent: IntentMemoryQuery, pipeline: PipelineChat},
		{message: "帮我整理一份 codex 交接总结", intent: IntentHandoff, pipeline: PipelineProject},
		{message: "你好", intent: IntentGeneral, pipeline: PipelineChat},
	}

	for _, test := range tests {
		got := Dispatch(test.message)
		if got.Intent != test.intent || got.Pipeline != test.pipeline {
			t.Fatalf("Dispatch(%q) = %#v, want intent=%q pipeline=%q", test.message, got, test.intent, test.pipeline)
		}
	}
}
