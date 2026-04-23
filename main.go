package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/xdm67x/simple-agent/agent"
	"github.com/xdm67x/simple-agent/tools"
)

//go:embed SYSTEM_PROMPT.md
var systemPrompt string

// tea messages for async agent events
type (
	agentResponseMsg string
	agentErrorMsg    string
	thinkingStartMsg struct{}
	thinkingEndMsg   struct{}
	toolCallMsg      struct {
		name string
		args map[string]any
	}
	toolResultMsg struct {
		name   string
		result string
	}
	safetyCheckMsg struct {
		cmd  string
		resp chan bool
	}
)

type model struct {
	agent    *agent.Agent
	renderer *glamour.TermRenderer

	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model

	messages []string
	thinking bool
	status   string

	awaitingSafetyCheck *safetyCheckMsg
	eventCh             chan tea.Msg

	width  int
	height int
}

func initialModel() (*model, error) {
	modelName := os.Getenv("OLLAMA_MODEL")
	if modelName == "" {
		modelName = "gemma4:31b-cloud"
	}

	a, err := agent.NewAgent(modelName, systemPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}
	a.RegisterTool(&tools.BashTool{})
	a.RegisterTool(&tools.AskUserTool{})

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create markdown renderer: %w", err)
	}

	ta := textarea.New()
	ta.Placeholder = "Send a message..."
	ta.Prompt = "┃ "
	ta.Focus()
	ta.CharLimit = 0
	ta.SetHeight(3)
	ta.MaxHeight = 5

	vp := viewport.New(80, 20)
	vp.SetContent(`Welcome to Simple Agent! Type a message and press Enter to send.
Commands:
  /new  - clear conversation context
  ctrl+c or esc - quit
`)

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return &model{
		agent:    a,
		renderer: renderer,
		textarea: ta,
		viewport: vp,
		spinner:  sp,
		eventCh:  make(chan tea.Msg, 100),
	}, nil
}

func (m *model) addMessage(role, content string) {
	var rendered string
	switch role {
	case "user":
		label := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4")).Render("You")
		rendered = label + "\n" + content
	case "agent":
		out, err := m.renderer.Render(content)
		if err != nil {
			out = content
		}
		out = strings.Trim(out, "\n")
		label := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#04B575")).Render("Agent")
		rendered = label + "\n" + out
	case "error":
		label := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF0000")).Render("Error")
		rendered = label + " " + content
	case "tool":
		rendered = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render(content)
	}
	m.messages = append(m.messages, rendered)
	m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
	m.viewport.GotoBottom()
}

func waitForEvent(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func (m model) runAgent(input string) tea.Cmd {
	return func() tea.Msg {
		m.agent.OnThinkingStart = func() {
			m.eventCh <- thinkingStartMsg{}
		}
		m.agent.OnThinkingEnd = func() {
			m.eventCh <- thinkingEndMsg{}
		}
		m.agent.OnToolCall = func(name string, args map[string]any) {
			m.eventCh <- toolCallMsg{name: name, args: args}
		}
		m.agent.OnToolResult = func(name string, result string) {
			m.eventCh <- toolResultMsg{name: name, result: result}
		}
		m.agent.OnSafetyCheck = func(cmd string) bool {
			req := safetyCheckMsg{cmd: cmd, resp: make(chan bool)}
			m.eventCh <- req
			return <-req.resp
		}

		resp, err := m.agent.Run(input)
		if err != nil {
			m.eventCh <- agentErrorMsg(err.Error())
		} else {
			m.eventCh <- agentResponseMsg(resp)
		}
		return nil
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		waitForEvent(m.eventCh),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.awaitingSafetyCheck != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyCtrlC {
				m.awaitingSafetyCheck.resp <- false
				return m, tea.Quit
			}
			if strings.ToLower(msg.String()) == "y" {
				m.awaitingSafetyCheck.resp <- true
			} else {
				m.awaitingSafetyCheck.resp <- false
			}
			m.awaitingSafetyCheck = nil
			return m, waitForEvent(m.eventCh)
		default:
			return m, nil
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		if msg.Height > 6 {
			m.viewport.Height = msg.Height - 6
		} else {
			m.viewport.Height = 1
		}
		m.textarea.SetWidth(msg.Width)
		return m, nil

	case thinkingStartMsg:
		m.thinking = true
		m.status = "Thinking..."
		return m, tea.Batch(m.spinner.Tick, waitForEvent(m.eventCh))

	case thinkingEndMsg:
		m.thinking = false
		m.status = ""
		return m, waitForEvent(m.eventCh)

	case toolCallMsg:
		argsJSON, _ := json.Marshal(msg.args)
		m.status = fmt.Sprintf("Tool: %s(%s)", msg.name, string(argsJSON))
		m.addMessage("tool", fmt.Sprintf("🔧 %s(%s)", msg.name, string(argsJSON)))
		return m, waitForEvent(m.eventCh)

	case toolResultMsg:
		summary := msg.result
		lines := strings.Split(summary, "\n")
		if len(lines) > 0 {
			summary = lines[0]
		}
		if len(summary) > 80 {
			summary = summary[:77] + "..."
		}
		m.status = fmt.Sprintf("Result: %s", summary)
		m.addMessage("tool", fmt.Sprintf("✅ %s: %s", msg.name, summary))
		return m, waitForEvent(m.eventCh)

	case safetyCheckMsg:
		m.awaitingSafetyCheck = &msg
		m.status = fmt.Sprintf("Safety check: %s", msg.cmd)
		return m, waitForEvent(m.eventCh)

	case agentResponseMsg:
		m.addMessage("agent", string(msg))
		return m, waitForEvent(m.eventCh)

	case agentErrorMsg:
		m.addMessage("error", string(msg))
		return m, waitForEvent(m.eventCh)

	case spinner.TickMsg:
		if m.thinking {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			input := strings.TrimSpace(m.textarea.Value())
			if input == "" {
				return m, nil
			}
			if input == "/new" {
				m.agent.Clear()
				m.messages = nil
				m.viewport.SetContent("")
				m.textarea.SetValue("")
				return m, nil
			}
			m.textarea.SetValue("")
			m.addMessage("user", input)
			return m, tea.Batch(m.runAgent(input), waitForEvent(m.eventCh))
		}

		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

	default:
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Width(m.width)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		Width(m.width)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Width(m.width)

	title := titleStyle.Render("🤖 Simple Agent")

	var status string
	if m.awaitingSafetyCheck != nil {
		status = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF0000")).
			Render(fmt.Sprintf("Safety check: %s  [Approve? y/n]", m.awaitingSafetyCheck.cmd))
	} else if m.thinking {
		status = statusStyle.Render(m.spinner.View() + " " + m.status)
	} else {
		status = statusStyle.Render(m.status)
	}

	help := helpStyle.Render("enter: send • shift+enter: newline • /new: clear • ctrl+c: quit")

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		m.viewport.View(),
		status,
		m.textarea.View(),
		help,
	)
}

func main() {
	m, err := initialModel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
