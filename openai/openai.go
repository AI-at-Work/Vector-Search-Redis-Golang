package openai

import (
	"context"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"strings"
	"time"
)

const (
	apiKeyEnvVar = "API_KEY"
	timeout      = 30 * time.Second
)

func ApiEmbedding(input string) ([]float32, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("input is empty")
	}

	apiKey := apiKeyEnvVar
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key not found in environment variable %s", apiKeyEnvVar)
	}

	client := openai.NewClient(apiKey)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req := openai.EmbeddingRequest{
		Input: []string{input},
		Model: openai.AdaEmbeddingV2,
	}

	resp, err := client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("error creating embeddings: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings found for the input")
	}

	embedding := resp.Data[0].Embedding
	fmt.Printf("Embedding length: %d\n", len(embedding))

	return embedding, nil
}
