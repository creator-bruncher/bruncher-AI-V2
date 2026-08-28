//go:build darwin

package tools

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type ToolExecutor struct{}

type ActionExecutor = ToolExecutor

func NewToolExecutor() *ToolExecutor {
	return &ToolExecutor{}
}

func NewActionExecutor() *ToolExecutor {
	return NewToolExecutor()
}

func (e *ToolExecutor) Execute(action string, params map[string]interface{}) (string, error) {
	switch action {
	case "zsh", "bash", "command":
		cmdStr, _ := params["command"].(string)
		return e.runShell(cmdStr)
	default:
		return "", fmt.Errorf("action '%s' not implemented for macOS yet", action)
	}
}

func (e *ToolExecutor) runShell(command string) (string, error) {
	trimmed := strings.TrimSpace(command)
	execCmd := exec.Command("zsh", "-c", trimmed)

	var out bytes.Buffer
	var stderr bytes.Buffer
	execCmd.Stdout = &out
	execCmd.Stderr = &stderr

	err := execCmd.Run()
	if err != nil {
		errStr := stderr.String()
		if errStr == "" {
			errStr = err.Error()
		}
		return "", fmt.Errorf("macOS execution error: %s", errStr)
	}
	return out.String(), nil
}