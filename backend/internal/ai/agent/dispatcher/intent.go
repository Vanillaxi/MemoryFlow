package dispatcher

const (
	IntentProjectProgress    = "project_progress"
	IntentProjectIssueStatus = "project_issue_status"
	IntentProjectPRStatus    = "project_pr_status"
	IntentMemoryQuery        = "memory_query"
	IntentExternalKnowledge  = "external_knowledge"
	IntentHandoff            = "handoff"
	IntentGeneral            = "general"
)

const (
	PipelineChat       = "chat_pipeline"
	PipelineKnowledge  = "knowledge_pipeline"
	PipelineProject    = "project_pipeline"
	PipelineReflection = "reflection_pipeline"
)
