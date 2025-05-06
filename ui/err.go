package ui

import (
	"github.com/charmbracelet/lipgloss"
	"strings"
)

type ErrBox struct {
	height, width int
	err           error
	infoMessage   string
}

var errStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
	Light: "#FF0000",
	Dark:  "#FF0000",
})

var infoStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
	Light: "#008000",
	Dark:  "#00FF00",
})

func NewErrBox() *ErrBox {
	return &ErrBox{}
}

func (e *ErrBox) SetError(err error) {
	e.err = err
}

func (e *ErrBox) Clear() {
	e.err = nil
	e.infoMessage = ""
}

func (e *ErrBox) SetInfo(message string) {
	e.infoMessage = message
	e.err = nil
}

func (e *ErrBox) SetSize(width, height int) {
	e.width = width
	e.height = height
}

func (e *ErrBox) String() string {
	if e.err != nil {
		// Display error message
		errText := e.err.Error()
		lines := strings.Split(errText, "\n")
		errText = strings.Join(lines, "//")
		if len(errText) > e.width-3 && e.width-3 >= 0 {
			errText = errText[:e.width-3] + "..."
		}
		return lipgloss.Place(e.width, e.height, lipgloss.Center, lipgloss.Center, errStyle.Render(errText))
	} else if e.infoMessage != "" {
		// Display info message
		infoText := e.infoMessage
		lines := strings.Split(infoText, "\n")
		infoText = strings.Join(lines, "//")
		if len(infoText) > e.width-3 && e.width-3 >= 0 {
			infoText = infoText[:e.width-3] + "..."
		}
		return lipgloss.Place(e.width, e.height, lipgloss.Center, lipgloss.Center, infoStyle.Render(infoText))
	}
	// No message to display
	return lipgloss.Place(e.width, e.height, lipgloss.Center, lipgloss.Center, "")
}
