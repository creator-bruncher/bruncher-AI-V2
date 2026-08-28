package hardware

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type SystemProfile struct {
	OS            string
	Arch          string
	IsIntelMac    bool
	RecommendedModel string
}

// DetectSystem inspects CPU and OS to pick the optimal Ollama model
func DetectSystem() SystemProfile {
	profile := SystemProfile{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	// Check if running on an Intel Mac
	if profile.OS == "darwin" {
		out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output()
		if err == nil && strings.Contains(string(out), "Intel") {
			profile.IsIntelMac = true
		}
	}

	// Determine recommended model to preserve thermals & speed
	if profile.IsIntelMac || profile.Arch == "386" {
		profile.RecommendedModel = "qwen2.5-coder:3b"
	} else {
		profile.RecommendedModel = "qwen2.5-coder:7b"
	}

	return profile
}

func (s SystemProfile) Summary() string {
	return fmt.Sprintf("OS: %s | Arch: %s | Intel Mac: %t -> Optimal Model: %s",
		s.OS, s.Arch, s.IsIntelMac, s.RecommendedModel)
}