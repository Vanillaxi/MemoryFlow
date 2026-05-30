package embedder

import (
	"context"

	einoembedding "github.com/cloudwego/eino/components/embedding"
)

var _ einoembedding.Embedder = (*EinoAdapter)(nil)

type EinoAdapter struct {
	client *Client
}

func NewEinoAdapter(client *Client) *EinoAdapter {
	return &EinoAdapter{client: client}
}

func (a *EinoAdapter) EmbedStrings(ctx context.Context, texts []string, _ ...einoembedding.Option) ([][]float64, error) {
	embeddings := make([][]float64, 0, len(texts))
	for _, text := range texts {
		vector, err := a.client.Embed(ctx, text)
		if err != nil {
			return nil, err
		}

		converted := make([]float64, len(vector))
		for i, value := range vector {
			converted[i] = float64(value)
		}
		embeddings = append(embeddings, converted)
	}
	return embeddings, nil
}
