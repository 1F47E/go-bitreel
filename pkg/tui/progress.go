package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render

const (
	padding  = 2
	maxWidth = 80
)

type tickMsg time.Time

type mode int

const (
	spin mode = iota
	bar
)

type Bar struct {
	mode mode

	title string
	// spinner
	spinner spinner.Model
	// progress
	progress progress.Model
	percent  float64
	finished bool
}

func NewProgress() *Bar {
	return &Bar{
		progress: progress.New(progress.WithDefaultGradient()),
		percent:  0,
	}
}

func (b *Bar) UpdateProgress(title string, percent float64) {
	b.mode = bar
	b.title = title
	b.percent = percent
}

func (b *Bar) UpdateSpinner(title string) {
	b.mode = spin
	b.title = title
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
			m.finished = true
			// return m, tea.Quit
			return m, nil
		}

		// Note that you can also use progress.Model.SetPercent to set the
		// percentage value explicitly, too.
		// cmd := m.progress.IncrPercent(0.25)
		cmd := m.progress.SetPercent(m.percent)
		return m, tea.Batch(tickCmd(), cmd)
		// return m, cmd

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

	if m.mode == spin {
		return "spinner mode"

	} else if m.mode == bar {
		if m.finished {
			return "\n" + pad + "✅ Finished!\n"
		}
		return "\n" +
			pad + m.title + "\n\n" +
			pad + m.progress.View() + "\n" +
			pad + helpStyle("Press any key to quit")
	}
	return "-"
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
