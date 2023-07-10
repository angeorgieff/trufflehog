package textinputs

// from https://github.com/charmbracelet/bubbletea/blob/master/examples/textinputs/main.go

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle         = focusedStyle.Copy()
	noStyle             = lipgloss.NewStyle()
	helpStyle           = blurredStyle.Copy()
	cursorModeHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	focusedButton     = focusedStyle.Copy().Render("[ Next ]")
	blurredButton     = fmt.Sprintf("[ %s ]", blurredStyle.Render("Next"))
	focusedSkipButton = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("[ Run with defaults ]")
	blurredSkipButton = fmt.Sprintf("[ %s ]", lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Run with defaults"))
)

type model struct {
	focusIndex int
	inputs     []textinput.Model
	configs    []InputConfig
	cursorMode cursor.Mode
	skipButton bool
}

type InputConfig struct {
	Label       string
	Help        string
	Required    bool
	Placeholder string
}

func New(config []InputConfig) model {
	m := model{
		inputs: make([]textinput.Model, len(config)),
	}

	for i, conf := range config {
		input := textinput.New()
		input.Placeholder = conf.Placeholder

		if i == 0 {
			input.Focus()
			input.TextStyle = focusedStyle
			input.PromptStyle = focusedStyle
		}

		m.inputs[i] = input
	}

	m.configs = config
	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		// Change cursor mode
		case "ctrl+r":
			m.cursorMode++
			if m.cursorMode > cursor.CursorHide {
				m.cursorMode = cursor.CursorBlink
			}
			cmds := make([]tea.Cmd, len(m.inputs))
			for i := range m.inputs {
				cmds[i] = m.inputs[i].Cursor.SetMode(m.cursorMode)
			}
			return m, tea.Batch(cmds...)

		// Set focus to next input
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Did the user press enter while the submit button was focused?
			// If so, exit.
			if s == "enter" && m.focusIndex == len(m.inputs) {
				return m, tea.Quit
			}

			// Cycle indexes
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if !m.skipButton && m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			} else if m.skipButton && m.focusIndex < -1 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					// Set focused state
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
					continue
				}
				// Remove focused state
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = noStyle
				m.inputs[i].TextStyle = noStyle
			}

			return m, tea.Batch(cmds...)
		}
	}

	// Handle character input and blinking
	cmd := m.updateInputs(msg)

	return m, cmd
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m model) View() string {
	var b strings.Builder

	if m.skipButton {
		button := &blurredSkipButton
		if m.focusIndex == -1 {
			button = &focusedSkipButton
		}
		fmt.Fprintf(&b, "%s\n\n\n", *button)
	}

	for i := range m.inputs {
		if m.configs[i].Label != "" {
			b.WriteString(m.GetLabel(m.configs[i]))
		}

		b.WriteString(m.inputs[i].View())
		b.WriteRune('\n')
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if m.focusIndex == len(m.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	b.WriteString(helpStyle.Render("cursor mode is "))
	b.WriteString(cursorModeHelpStyle.Render(m.cursorMode.String()))
	b.WriteString(helpStyle.Render(" (ctrl+r to change style)"))

	return b.String()
}

func (m model) GetLabel(c InputConfig) string {
	var label strings.Builder

	label.WriteString(c.Label)
	if c.Required {
		label.WriteString("*")
	}

	if len(c.Help) > 0 {
		label.WriteString("\n" + helpStyle.Render(c.Help))
	}

	label.WriteString("\n")
	return label.String()
}

func (m model) SetSkip(skip bool) model {
	m.skipButton = skip
	if m.skipButton {
		if len(m.inputs) > 0 {
			m.inputs[0].Blur()
		}
		m.focusIndex = -1
	}
	return m
}