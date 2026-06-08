package dispatcher

const (
	IntentProjectProgress    = "project_progress"
	IntentProjectIssueStatus = "project_issue_status"
	IntentProjectPRStatus    = "project_pr_status"
	IntentProjectHandoff     = "project_handoff"
	IntentMemoryQuery        = "memory_query"
	IntentExternalKnowledge  = "external_knowledge"
	IntentGeneral            = "general"
)

const (
	PipelineChat       = "chat_pipeline"
	PipelineKnowledge  = "knowledge_pipeline"
	PipelineProject    = "project_pipeline"
	PipelineReflection = "reflection_pipeline"
)
