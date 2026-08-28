package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"bruncher-ai/pkg/hardware"
)

type OllamaClient struct {
	BaseURL string
	Model   string
	Client  *http.Client
}

func NewClient() *OllamaClient {
	profile := hardware.DetectSystem()
	fmt.Printf("🔍 System Check: %s\n", profile.Summary())

	return &OllamaClient{
		BaseURL: "http://localhost:11434",
		Model:   profile.RecommendedModel,
		Client:  &http.Client{Timeout: 120 * time.Second},
	}
}

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GenerateResponse struct {
	Response string `json:"response"`
}

func (c *OllamaClient) Query(prompt string) (string, error) {
	reqBody := GenerateRequest{
		Model:  c.Model,
		Prompt: prompt,
		Stream: false,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := c.Client.Post(c.BaseURL+"/api/generate", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("ollama connection failed: %w", err)
	}
	defer resp.Body.Close()

	var genResp GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", err
	}

	return genResp.Response, nil
}