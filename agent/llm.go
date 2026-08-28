package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type LLMClient struct {
	LocalEndpoint string
	ModelName     string
	HTTPClient    *http.Client
	ServerCmd     *exec.Cmd
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResponse struct {
	Message ollamaMessage `json:"message"`
}

func EnsureModelExists() error {
	modelsDir := "models"
	modelPath := filepath.Join(modelsDir, "qwen2.5-coder-3b.gguf")

	if _, err := os.Stat(modelPath); err == nil {
		return nil
	}

	fmt.Println("📦 First run detected! Downloading Qwen2.5-Coder 3B model (~1.9 GB)...")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return err
	}

	downloadURL := "https://huggingface.co/Qwen/Qwen2.5-Coder-3B-Instruct-GGUF/resolve/main/qwen2.5-coder-3b-instruct-q4_k_m.gguf"

	out, err := os.Create(modelPath)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	return err
}

func NewLocalLLMClient(model string) *LLMClient {
	if model == "" {
		model = "qwen2.5-coder:7b"
	}

	client := &LLMClient{
		LocalEndpoint: "http://127.0.0.1:11434/api/chat",
		ModelName:     model,
		HTTPClient:    &http.Client{Timeout: 90 * time.Second},
	}

	client.ensureServerRunning()
	return client
}

func (c *LLMClient) ensureServerRunning() {
	pingClient := http.Client{Timeout: 1 * time.Second}
	_, err := pingClient.Get("http://127.0.0.1:11434/")
	if err == nil {
		return
	}

	cmd := exec.Command("ollama", "serve")
	if err := cmd.Start(); err == nil {
		c.ServerCmd = cmd
		time.Sleep(2 * time.Second)
		return
	}

	c.startBundledEngine()
}

func (c *LLMClient) startBundledEngine() {
	exeDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return
	}

	serverPath := filepath.Join(exeDir, "llama-server.exe")
	modelPath := filepath.Join(exeDir, "models", "qwen2.5-coder-3b.gguf")

	if _, err := os.Stat(serverPath); err == nil {
		if _, err := os.Stat(modelPath); err == nil {
			cmd := exec.Command(serverPath, "-m", modelPath, "--port", "11434", "-c", "4096")
			cmd.Stdout = nil
			cmd.Stderr = nil
			if err := cmd.Start(); err == nil {
				c.ServerCmd = cmd
				time.Sleep(2 * time.Second)
			}
		}
	}
}

func (c *LLMClient) Query(messages []map[string]interface{}, tools []map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"model":       c.ModelName,
		"messages":    messages,
		"stream":      false,
		"temperature": 0.1,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.LocalEndpoint, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("engine not reachable (ensure Ollama is running): %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama API error (%d): %s", resp.StatusCode, string(body))
	}

	var oResp ollamaResponse
	if err := json.Unmarshal(body, &oResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	standardizedResult := map[string]interface{}{
		"choices": []map[string]interface{}{
			{
				"message": map[string]interface{}{
					"role":    oResp.Message.Role,
					"content": oResp.Message.Content,
				},
			},
		},
	}

	return standardizedResult, nil
}