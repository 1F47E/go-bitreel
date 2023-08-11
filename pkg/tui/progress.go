package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	padding  = 2
	maxWidth = 80
)

type tickMsg time.Time

type Bar struct {
	progress progress.Model
}

func NewProgress() *Bar {
	return &Bar{
		progress: progress.New(progress.WithDefaultGradient()),
	}
}

func (m *Bar) Run() {
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Oh no!", err)
		os.Exit(1)
	}
}

func (m *Bar) Init() tea.Cmd {
	return tickCmd()
}

func (m *Bar) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
		return m, nil

	case tickMsg:
		if m.progress.Percent() == 1.0 {
			return m, tea.Quit
		}

		// Note that you can also use progress.Model.SetPercent to set the
		// percentage value explicitly, too.
		cmd := m.progress.IncrPercent(0.25)
		return m, tea.Batch(tickCmd(), cmd)

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	default:
		return m, nil
	}
}

func (m *Bar) View() string {
	pad := strings.Repeat(" ", padding)
	return "\n" +
		pad + m.progress.View() + "\n"
	// pad + helpStyle("Press any key to quit")
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
