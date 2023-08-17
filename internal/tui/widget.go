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

const (
	padding  = 2
	maxWidth = 80
)

type tickMsg time.Time

type mode int

const (
	spin mode = iota
	bar
	text
)

type Widget struct {
	mode     mode
	title    string
	spinner  spinner.Model
	progress progress.Model
	percent  float64
}

func NewWidget() *Widget {
	s := spinner.New()
	s.Spinner = spinner.Line
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &Widget{
		spinner:  s,
		progress: progress.New(progress.WithDefaultGradient()),
		percent:  0,
	}
}

func (w *Widget) SetProgress(title string, percent float64) {
	w.mode = bar
	w.title = title
	w.percent = percent
}

func (w *Widget) SetSpinner(title string) {
	w.mode = spin
	w.title = title
}

func (w *Widget) SetText(title string) {
	w.mode = text
	w.title = title
}

func (w *Widget) Run() {
	if _, err := tea.NewProgram(w).Run(); err != nil {
		fmt.Println("Oh no!", err)
		os.Exit(1)
	}
}

func (w *Widget) Init() tea.Cmd {
	return tea.Batch(tickCmd(), w.spinner.Tick)
}

func (w *Widget) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return w, tea.Quit

	case tea.WindowSizeMsg:
		w.progress.Width = msg.Width - padding*2 - 4
		if w.progress.Width > maxWidth {
			w.progress.Width = maxWidth
		}
		return w, nil

	case tickMsg:

		cmd := w.progress.SetPercent(w.percent)
		return w, tea.Batch(tickCmd(), cmd)

	// FrameMsg is sent when the progress bar wants to animate itself
	case progress.FrameMsg:
		progressModel, cmd := w.progress.Update(msg)
		w.progress = progressModel.(progress.Model)
		return w, cmd

	default:
		var cmd tea.Cmd
		w.spinner, cmd = w.spinner.Update(msg)
		return w, cmd
	}

}

func (w *Widget) View() string {
	pad := strings.Repeat(" ", padding)

	if w.mode == text {
		return fmt.Sprintf("\n\n%s%s\n\n", pad, w.title)
	} else if w.mode == spin {
		return fmt.Sprintf("\n\n%s%s %s\n\n", pad, w.spinner.View(), w.title)

	} else if w.mode == bar {
		return "\n" +
			pad + w.title + "\n\n" +
			pad + w.progress.View() + "\n"
	}
	return ""
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Microsecond*100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
