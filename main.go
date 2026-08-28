package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"bruncher-ai/agent"
	"bruncher-ai/tools"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

type chatLog struct {
	sender string
	text   string
}

type model struct {
	viewport      viewport.Model
	textarea      textarea.Model
	chatLogs      []chatLog
	loading       bool
	err           error
	llmClient     *agent.LLMClient
	executor      *tools.ToolExecutor
	history       []map[string]interface{}
	selectedModel string
	width         int
	height        int
}

type responseMsg struct {
	text string
	err  error
}

func getSystemPrompt() string {
	if runtime.GOOS == "darwin" {
		return "You are a helpful macOS desktop AI assistant.\n\n" +
			"1. If the user greets you or asks a general question, respond naturally with text.\n" +
			"2. ONLY if the user explicitly asks to open an app, run a command, or perform a system action, respond with a single macOS terminal/zsh command inside a markdown code block.\n\n" +
			"Examples for action requests:\n" +
			"- Steam: open -a Steam\n" +
			"- Discord: open -a Discord\n" +
			"- Calculator: open -a Calculator\n" +
			"- Browser: open https://google.com\n\n" +
			"Do NOT spawn new terminal commands to reply to simple text greetings like 'hi' or '?'."
	}

	return "You are a helpful Windows desktop AI assistant.\n\n" +
		"1. If the user greets you or asks a general question, respond naturally with text.\n" +
		"2. ONLY if the user explicitly asks to open an app, run a command, or perform a system action, respond with a single PowerShell command inside a markdown code block.\n\n" +
		"Examples for action requests:\n" +
		"- Steam: start 'steam://'\n" +
		"- Discord: Start-Process \"$env:LOCALAPPDATA\\Discord\\Update.exe\" -ArgumentList '--processStart Discord.exe'\n" +
		"- Calculator: start calc\n" +
		"- Browser: start https://google.com\n\n" +
		"Do NOT spawn new PowerShell windows to reply to simple text greetings like 'hi' or '?'."
}

func initialModel(customModel string) model {
	var optimalModel string

	if customModel != "" {
		optimalModel = customModel
	} else {
		// Detect system specs and check for locally installed Ollama models
		specs := agent.DetectSpecs()
		optimalModel = agent.SelectOptimalModel(specs)
	}

	ta := textarea.New()
	ta.Placeholder = "Type a task (e.g. 'open calculator') and press Enter..."
	ta.Focus()
	ta.CharLimit = 280
	ta.SetWidth(80)
	ta.SetHeight(3)

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false

	vp := viewport.New(80, 20)

	llm := agent.NewLocalLLMClient(optimalModel)
	exec := tools.NewToolExecutor()

	sysMsg := map[string]interface{}{
		"role":    "system",
		"content": getSystemPrompt(),
	}

	return model{
		textarea:      ta,
		viewport:      vp,
		chatLogs:      []chatLog{},
		loading:       false,
		llmClient:     llm,
		executor:      exec,
		selectedModel: optimalModel,
		history:       []map[string]interface{}{sysMsg},
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func sendLLMQuery(llm *agent.LLMClient, executor *tools.ToolExecutor, history []map[string]interface{}) tea.Cmd {
	return func() tea.Msg {
		res, err := llm.Query(history, nil)
		if err != nil {
			return responseMsg{err: err}
		}

		choices, ok := res["choices"].([]map[string]interface{})
		if !ok || len(choices) == 0 {
			if rawChoices, okRaw := res["choices"].([]interface{}); okRaw && len(rawChoices) > 0 {
				if choiceMap, okMap := rawChoices[0].(map[string]interface{}); okMap {
					if msgMap, okMsg := choiceMap["message"].(map[string]interface{}); okMsg {
						if content, okTxt := msgMap["content"].(string); okTxt && content != "" {
							return responseMsg{text: content}
						}
					}
				}
			}
			return responseMsg{text: "No response received from local LLM."}
		}

		msgMap, okMsg := choices[0]["message"].(map[string]interface{})
		if !okMsg {
			return responseMsg{text: "No response received from local LLM."}
		}

		content, _ := msgMap["content"].(string)
		if content == "" {
			return responseMsg{text: "No response received from local LLM."}
		}

		return responseMsg{text: content}
	}
}

func runSystemCommand(command string) string {
	var cmd *exec.Cmd

	if runtime.GOOS == "darwin" {
		cmd = exec.Command("zsh", "-c", command)
	} else {
		cmd = exec.Command("powershell", "-Command", command)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Execution error: %v\n%s", err, string(output))
	}
	if len(output) == 0 {
		return "Command executed successfully."
	}
	return string(output)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - m.textarea.Height() - 6
		m.textarea.SetWidth(msg.Width - 4)

		m.renderViewport()

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.loading {
				return m, nil
			}

			userInput := m.textarea.Value()
			if userInput == "" {
				return m, nil
			}

			m.chatLogs = append(m.chatLogs, chatLog{sender: "User", text: userInput})
			m.history = append(m.history, map[string]interface{}{
				"role":    "user",
				"content": userInput,
			})

			m.textarea.Reset()
			m.loading = true
			m.renderViewport()

			return m, tea.Batch(tiCmd, vpCmd, sendLLMQuery(m.llmClient, m.executor, m.history))
		}

	case responseMsg:
		m.loading = false
		if msg.err != nil {
			m.chatLogs = append(m.chatLogs, chatLog{sender: "Agent", text: fmt.Sprintf("Error: %v", msg.err)})
		} else {
			m.chatLogs = append(m.chatLogs, chatLog{sender: "Agent", text: msg.text})
			m.history = append(m.history, map[string]interface{}{
				"role":    "assistant",
				"content": msg.text,
			})

			if strings.Contains(msg.text, "```") {
				parts := strings.Split(msg.text, "```")
				if len(parts) >= 3 {
					rawCmd := strings.TrimSpace(parts[1])
					if idx := strings.Index(rawCmd, "\n"); idx != -1 {
						rawCmd = rawCmd[idx+1:]
					}
					execOutput := runSystemCommand(rawCmd)
					m.chatLogs = append(m.chatLogs, chatLog{sender: "System", text: execOutput})
				}
			}
		}
		m.renderViewport()
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m *model) renderViewport() {
	var sb string

	header := fmt.Sprintf("🤖 bruncher-AI Active (OS: %s | Arch: %s | Model: %s)\n", runtime.GOOS, runtime.GOARCH, m.selectedModel)
	header += "Created by @creator-bruncher on GitHub\n"
	header += "--------------------------------------------------------\n\n"

	sb += header

	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	for _, log := range m.chatLogs {
		prefix := fmt.Sprintf("👤 %s: ", log.sender)
		if log.sender == "Agent" {
			prefix = "🤖 Agent: "
		} else if log.sender == "System" {
			prefix = "⚙️ System: "
		}
		wrapped := wordwrap.String(prefix+log.text, width)
		sb += wrapped + "\n\n"
	}

	if m.loading {
		sb += "⏳ Thinking...\n"
	}

	m.viewport.SetContent(sb)
	m.viewport.GotoBottom()
}

func (m model) View() string {
	return fmt.Sprintf(
		"%s\n\n%s",
		m.viewport.View(),
		m.textarea.View(),
	)
}

func main() {
	modelFlag := flag.String("model", "", "Specify custom Ollama model name (e.g. -model qwen2.5-coder:7b)")
	flag.Parse()

	p := tea.NewProgram(initialModel(*modelFlag), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}