package agent

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/pbnjay/memory"
)

type SystemSpecs struct {
	TotalRAMGB      uint64
	HasNVIDIA       bool
	HasAppleSilicon bool
}

type OllamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

// GetInstalledModels fetches the list of locally pulled models from Ollama API
func GetInstalledModels() []string {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var tagsResp OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil
	}

	var models []string
	for _, m := range tagsResp.Models {
		models = append(models, m.Name)
	}
	return models
}

// DetectSpecs returns system RAM and basic GPU availability
func DetectSpecs() SystemSpecs {
	totalRAMBytes := memory.TotalMemory()
	ramGB := totalRAMBytes / (1024 * 1024 * 1024)

	specs := SystemSpecs{
		TotalRAMGB: ramGB,
	}

	if runtime.GOOS == "darwin" && strings.Contains(runtime.GOARCH, "arm") {
		specs.HasAppleSilicon = true
	}

	if runtime.GOOS == "windows" {
		cmd := exec.Command("wmic", "path", "win32_videocard", "get", "name")
		out, err := cmd.Output()
		if err == nil && strings.Contains(strings.ToLower(string(out)), "nvidia") {
			specs.HasNVIDIA = true
		}
	}

	return specs
}

// SelectOptimalModel checks installed models first, falls back to hardware specs
func SelectOptimalModel(specs SystemSpecs) string {
	installed := GetInstalledModels()

	// If models are installed, pick the best installed match
	if len(installed) > 0 {
		for _, m := range installed {
			if strings.Contains(m, "qwen2.5-coder:7b") || strings.Contains(m, "7b") {
				return m
			}
		}
		for _, m := range installed {
			if strings.Contains(m, "qwen2.5-coder") {
				return m
			}
		}
		// Return the first available model installed locally
		return installed[0]
	}

	// Fallback if no models are installed locally yet
	if specs.TotalRAMGB >= 16 || specs.HasNVIDIA {
		return "qwen2.5-coder:7b"
	}
	return "qwen2.5-coder:1.5b"
}