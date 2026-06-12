// Package style is nole's shared stylesheet: lipgloss styles for static
// output and a matching huh.Theme for interactive prompts, so that printed
// text and forms/confirms share one consistent look.
package style

import (
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	cyanColor    = lipgloss.AdaptiveColor{Light: "6", Dark: "14"}
	greenColor   = lipgloss.AdaptiveColor{Light: "2", Dark: "10"}
	yellowColor  = lipgloss.AdaptiveColor{Light: "3", Dark: "11"}
	redColor     = lipgloss.AdaptiveColor{Light: "1", Dark: "9"}
	magentaColor = lipgloss.AdaptiveColor{Light: "5", Dark: "13"}
	faintColor   = lipgloss.AdaptiveColor{Light: "243", Dark: "243"}
)

// Bold, Faint, and the colour styles below are the building blocks for all
// static CLI output: "  Bold header" section titles, faint secondary text,
// and Cyan/Green/Yellow/Red/Magenta for icons and status text.
var (
	Bold    = lipgloss.NewStyle().Bold(true)
	Faint   = lipgloss.NewStyle().Foreground(faintColor)
	Cyan    = lipgloss.NewStyle().Foreground(cyanColor)
	Green   = lipgloss.NewStyle().Foreground(greenColor)
	Yellow  = lipgloss.NewStyle().Foreground(yellowColor)
	Red     = lipgloss.NewStyle().Foreground(redColor)
	Magenta = lipgloss.NewStyle().Foreground(magentaColor)
)

// Theme returns the huh.Theme used by all interactive prompts (confirms,
// selects, inputs), built on the same palette as the static styles above.
func Theme() *huh.Theme {
	t := huh.ThemeBase()

	t.Focused.Title = t.Focused.Title.Foreground(cyanColor).Bold(true)
	t.Focused.Description = t.Focused.Description.Foreground(faintColor)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(redColor)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(redColor)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(cyanColor)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(cyanColor)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(cyanColor)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(cyanColor)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(greenColor)
	t.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(greenColor).SetString("✓ ")
	t.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(faintColor).SetString("  ")
	t.Focused.UnselectedOption = lipgloss.NewStyle()
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(lipgloss.Color("0")).Background(cyanColor)
	t.Focused.BlurredButton = lipgloss.NewStyle().Foreground(faintColor)

	t.Group.Title = t.Focused.Title
	t.Group.Description = t.Focused.Description

	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())

	return t
}

// IsTerminal reports whether stdout is an interactive terminal. When it
// isn't (piped output, no /dev/tty), forms must run in accessible mode,
// since huh's normal TUI rendering requires a real TTY.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// RunForm runs a huh.Form with the shared theme, falling back to huh's
// accessible (plain stdin/stdout prompting) mode when stdout isn't a
// terminal.
func RunForm(form *huh.Form) error {
	return form.WithTheme(Theme()).WithAccessible(!IsTerminal()).Run()
}

// Spin runs action while showing a themed spinner with the given title
// (e.g. "  Evaluating config"). Falls back to a static line when stdout
// isn't a terminal.
func Spin(title string, action func() error) error {
	var actionErr error
	s := spinner.New().
		Title(title).
		TitleStyle(Faint).
		Style(Cyan).
		Accessible(!IsTerminal()).
		Action(func() {
			actionErr = action()
		})
	if err := s.Run(); err != nil {
		return err
	}
	return actionErr
}

// Confirm prompts the user with a themed y/N question and returns true for
// "Yes". A non-interactive terminal or an aborted prompt is treated as "No".
func Confirm(prompt string) bool {
	var ok bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(prompt).
				Affirmative("Yes").
				Negative("No").
				Value(&ok),
		),
	)
	_ = RunForm(form)
	return ok
}
