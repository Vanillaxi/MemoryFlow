package chat_pipeline

import (
	"memoryflow/internal/ai/embedder"

	einoembedding "github.com/cloudwego/eino/components/embedding"
)

func NewEmbedding(client *embedder.Client) einoembedding.Embedder {
	return embedder.NewEmbedderAdapter(client)
}
