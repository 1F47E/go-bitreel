package tui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type errMsg error

type Spinner struct {
	spinner  spinner.Model
	quitting bool
	err      error
	mode     string
}

func NewSpinner() *Spinner {
	s := spinner.New()
	s.Spinner = spinner.Line
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return &Spinner{spinner: s}
}

func (m *Spinner) Run() {
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func (m *Spinner) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *Spinner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q":
			m.mode = "next"
			return m, nil
		default:
			return m, nil
		}

	case errMsg:
		m.err = msg
		return m, nil

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *Spinner) View() string {
	if m.err != nil {
		return m.err.Error()
	}

	// next view
	if m.mode == "next" {
		return "next view"
	}

	str := fmt.Sprintf("\n\n   %s Loading forever...press q to next or ESC to exit\n\n", m.spinner.View())
	if m.quitting {
		return str + "\n"
	}
	return str
}
