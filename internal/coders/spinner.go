package coders

import (
	"fmt"
	"math/rand"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

var loadingMessages = []string{
	"let me think",
	"let me see",
	"thinking",
	"loading",
	"hold on",
	"calculating",
	"processing",
	"please wait",
}

type Spinner struct {
	randMsg string
	spinner spinner.Model
}

func NewSpinner() *Spinner {
	spin := spinner.New()
	spin.Spinner = spinner.MiniDot

	return &Spinner{
		randMsg: loadingMessages[rand.Intn(len(loadingMessages))],
		spinner: spin,
	}
}

func (s *Spinner) Update(msg tea.Msg) (*Spinner, tea.Cmd) {
	var updateCmd tea.Cmd
	s.spinner, updateCmd = s.spinner.Update(msg)

	return s, updateCmd
}

func (s *Spinner) View() string {
	return fmt.Sprintf(
		"\n  %s %s...",
		s.spinner.View(),
		s.spinner.Style.Render(s.randMsg),
	)
}

func (s *Spinner) ViewWithMessage(prefix, suffix, spinnerMsg string) string {
	msg := fmt.Sprintf(
		"\n%s %s %s...",
		prefix,
		s.spinner.View(),
		s.spinner.Style.Render(spinnerMsg),
	)
	if len(suffix) > 0 {
		msg += "\n" + components.renderer.RenderContent(suffix)
	}
	return msg
}

func (s *Spinner) Tick() tea.Msg {
	return s.spinner.Tick()
}
