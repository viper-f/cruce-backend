package GeminiClient

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

type GeminiClient struct {
	client *genai.Client
	model  string
}

func NewGeminiClient(apiKey string, model string) *GeminiClient {
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil
	}
	return &GeminiClient{
		client: client,
		model:  model,
	}
}

func (g *GeminiClient) Call(ctx context.Context, prompt string) (string, error) {
	result, err := g.client.Models.GenerateContent(ctx, g.model, genai.Text(prompt), nil)
	if err != nil {
		return "", err
	}

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		return result.Candidates[0].Content.Parts[0].Text, nil
	}
	return "", fmt.Errorf("empty response from Gemini")
}
