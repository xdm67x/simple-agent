package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	askUserMsg struct {
		question string
		resp     chan string
	}
)

type model struct {
	agent    *agent.Agent
	renderer *glamour.TermRenderer
	config   agent.Config

	viewport viewport.Model
	textarea textarea.Model
	spinner  spinner.Model

	messages []string
	thinking bool
	status   string

	awaitingSafetyCheck *safetyCheckMsg
	awaitingAskUser     *askUserMsg
	drawerTextarea      textarea.Model

	eventCh chan tea.Msg

	width  int
	height int
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home + "/.simple-agent/config.json"
}

func loadConfig() agent.Config {
	path := configPath()
	if path == "" {
		return agent.Config{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return agent.Config{}
	}
	var c agent.Config
	_ = json.Unmarshal(data, &c)
	return c
}

func saveConfig(c agent.Config) error {
	path := configPath()
	if path == "" {
		return fmt.Errorf("unable to determine config path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func newDrawerTextarea() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Type your answer..."
	ta.Prompt = "> "
	ta.Focus()
	ta.CharLimit = 0
	ta.SetHeight(2)
	ta.MaxHeight = 4
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	ta.BlurredStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	return ta
}

func initialModel() (*model, error) {
	cfg := loadConfig()
	if envModel := os.Getenv("OLLAMA_MODEL"); envModel != "" {
		cfg.Model = envModel
	}
	if cfg.Model == "" {
		cfg.Model = "gemma4:31b-cloud"
	}

	a, err := agent.NewAgent(cfg, systemPrompt)
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
	ta.Placeholder = "Message..."
	ta.Prompt = "> "
	ta.Focus()
	ta.CharLimit = 0
	ta.SetHeight(2)
	ta.MaxHeight = 4
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	ta.BlurredStyle.Prompt = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	vp := viewport.New(80, 20)
	vp.SetContent("")

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return &model{
		agent:          a,
		renderer:       renderer,
		config:         cfg,
		textarea:       ta,
		viewport:       vp,
		spinner:        sp,
		drawerTextarea: newDrawerTextarea(),
		eventCh:        make(chan tea.Msg, 100),
	}, nil
}

func (m *model) addMessage(role, content string) {
	var rendered string
	switch role {
	case "user":
		prefix := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("› ")
		rendered = prefix + content
	case "agent":
		out, err := m.renderer.Render(content)
		if err != nil {
			out = content
		}
		rendered = strings.Trim(out, "\n")
	case "error":
		prefix := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("! ")
		rendered = prefix + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(content)
	case "tool":
		rendered = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true).Render("· " + content)
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
		m.agent.OnAskUser = func(question string) string {
			req := askUserMsg{question: question, resp: make(chan string)}
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
	// Drawer is active: safety check
	if m.awaitingSafetyCheck != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyCtrlC {
				m.awaitingSafetyCheck.resp <- false
				return m, tea.Quit
			}
			if msg.Type == tea.KeyEsc {
				m.awaitingSafetyCheck.resp <- false
				m.awaitingSafetyCheck = nil
				m.recalcViewport()
				return m, waitForEvent(m.eventCh)
			}
			if strings.ToLower(msg.String()) == "y" {
				m.awaitingSafetyCheck.resp <- true
				m.awaitingSafetyCheck = nil
				m.recalcViewport()
				return m, waitForEvent(m.eventCh)
			}
			if strings.ToLower(msg.String()) == "n" {
				m.awaitingSafetyCheck.resp <- false
				m.awaitingSafetyCheck = nil
				m.recalcViewport()
				return m, waitForEvent(m.eventCh)
			}
			return m, nil
		default:
			return m, nil
		}
	}

	// Drawer is active: ask user
	if m.awaitingAskUser != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyCtrlC:
				m.awaitingAskUser.resp <- "User cancelled"
				m.awaitingAskUser = nil
				m.drawerTextarea.SetValue("")
				return m, tea.Quit
			case tea.KeyEsc:
				m.awaitingAskUser.resp <- "User cancelled"
				m.awaitingAskUser = nil
				m.drawerTextarea.SetValue("")
				m.recalcViewport()
				return m, waitForEvent(m.eventCh)
			case tea.KeyEnter:
				answer := strings.TrimSpace(m.drawerTextarea.Value())
				m.awaitingAskUser.resp <- answer
				m.awaitingAskUser = nil
				m.drawerTextarea.SetValue("")
				m.recalcViewport()
				m.addMessage("user", answer)
				return m, waitForEvent(m.eventCh)
			}
			var cmd tea.Cmd
			m.drawerTextarea, cmd = m.drawerTextarea.Update(msg)
			return m, cmd
		default:
			var cmd tea.Cmd
			m.drawerTextarea, cmd = m.drawerTextarea.Update(msg)
			return m, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.textarea.SetWidth(msg.Width)
		m.drawerTextarea.SetWidth(msg.Width - 4)
		m.recalcViewport()
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
		m.recalcViewport()
		return m, waitForEvent(m.eventCh)

	case askUserMsg:
		m.awaitingAskUser = &msg
		m.status = fmt.Sprintf("Question: %s", msg.question)
		m.addMessage("agent", msg.question)
		m.drawerTextarea.SetValue("")
		m.drawerTextarea.Focus()
		m.recalcViewport()
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
			if input == "/model" {
				models, err := m.agent.ListModels()
				if err != nil {
					m.addMessage("error", fmt.Sprintf("Failed to list models: %v", err))
				} else {
					var b strings.Builder
					b.WriteString("Available models:\n")
					for _, name := range models {
						if name == m.agent.Model {
							b.WriteString(fmt.Sprintf("  > %s (current)\n", name))
						} else {
							b.WriteString(fmt.Sprintf("  • %s\n", name))
						}
					}
					b.WriteString("\nUse /model <name> to switch.")
					m.addMessage("agent", b.String())
				}
				m.textarea.SetValue("")
				return m, nil
			}
			if strings.HasPrefix(input, "/model ") {
				newModel := strings.TrimSpace(strings.TrimPrefix(input, "/model "))
				if newModel != "" {
					m.config.Model = newModel
					if err := m.agent.SetModel(newModel); err != nil {
						m.addMessage("error", fmt.Sprintf("Failed to switch model: %v", err))
					} else {
						if err := saveConfig(m.config); err != nil {
							m.addMessage("error", fmt.Sprintf("Failed to save config: %v", err))
						} else {
							m.addMessage("agent", fmt.Sprintf("Switched to model: %s", newModel))
						}
					}
				}
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

func (m *model) recalcViewport() {
	drawerHeight := m.drawerHeight()
	inputHeight := 3
	if m.awaitingSafetyCheck != nil || m.awaitingAskUser != nil {
		inputHeight = 0
	}
	helpHeight := 1
	sepHeight := 1
	available := m.height - drawerHeight - inputHeight - helpHeight - sepHeight
	if available < 1 {
		available = 1
	}
	m.viewport.Height = available
}

func (m model) drawerHeight() int {
	if m.awaitingSafetyCheck != nil {
		return 7
	}
	if m.awaitingAskUser != nil {
		return 7 + m.drawerTextarea.Height()
	}
	return 0
}

func (m model) drawerView() string {
	if m.awaitingSafetyCheck != nil {
		return m.safetyDrawerView()
	}
	if m.awaitingAskUser != nil {
		return m.questionDrawerView()
	}
	return ""
}

func (m model) safetyDrawerView() string {
	width := m.width - 2
	if width < 10 {
		width = 10
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("3")).
		Padding(0, 1).
		Width(width).
		Align(lipgloss.Center)

	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("3")).
		Padding(0, 1).
		Width(m.width)

	cmdStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")).
		Italic(true)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	cmd := m.awaitingSafetyCheck.cmd
	if len(cmd) > width-4 {
		cmd = cmd[:width-7] + "..."
	}

	title := titleStyle.Render("Permission Required")
	content := cmdStyle.Render(cmd)
	help := helpStyle.Render("[y] approve  [n] decline  [esc] cancel")

	inner := lipgloss.JoinVertical(lipgloss.Left, title, "", content, "", help)
	return boxStyle.Render(inner)
}

func (m model) questionDrawerView() string {
	width := m.width - 2
	if width < 10 {
		width = 10
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("12")).
		Padding(0, 1).
		Width(width).
		Align(lipgloss.Center)

	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(0, 1).
		Width(m.width)

	questionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Bold(true)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	q := m.awaitingAskUser.question
	if len(q) > width-4 {
		q = q[:width-7] + "..."
	}

	title := titleStyle.Render("Agent Question")
	question := questionStyle.Render(q)
	input := m.drawerTextarea.View()
	help := helpStyle.Render("enter submit · esc cancel")

	inner := lipgloss.JoinVertical(lipgloss.Left, title, "", question, input, help)
	return boxStyle.Render(inner)
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(strings.Repeat("─", m.width))

	var status string
	if m.thinking {
		status = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render(m.spinner.View() + " ...")
	}

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(fmt.Sprintf("enter send · shift+enter newline · /new clear · /model models · ctrl+c quit · %s", m.agent.Model))

	components := []string{
		m.viewport.View(),
		separator,
	}

	if drawer := m.drawerView(); drawer != "" {
		components = append(components, drawer)
	} else {
		components = append(components, m.textarea.View())
	}

	if status != "" {
		components = append(components, status)
	}
	components = append(components, help)

	return lipgloss.JoinVertical(lipgloss.Left, components...)
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
