// Package ui contains the Bubble Tea TUI layer for AzCockpit.
package ui

import "github.com/charmbracelet/lipgloss"

// Azure brand color palette.
var (
	PrimaryColor   = lipgloss.Color("#0078D4") // Azure blue
	SecondaryColor = lipgloss.Color("#50E6FF") // light cyan accent
	SuccessColor   = lipgloss.Color("#16C60C") // green (enabled/active)
	WarningColor   = lipgloss.Color("#FFB900") // yellow
	ErrorColor     = lipgloss.Color("#F03A17") // red
	MutedColor     = lipgloss.Color("#767676") // grey
)

// TitleStyle is used for the app title bar.
var TitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FFFFFF")).
	Background(PrimaryColor).
	Padding(0, 1)

// SelectedStyle highlights the currently selected item.
var SelectedStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#000000")).
	Background(SecondaryColor).
	Padding(0, 1)

// ActiveStyle highlights active/enabled subscriptions.
var ActiveStyle = lipgloss.NewStyle().
	Foreground(SuccessColor)

// DisabledStyle dims disabled subscriptions.
var DisabledStyle = lipgloss.NewStyle().
	Foreground(MutedColor)

// ErrorStyle formats error messages.
var ErrorStyle = lipgloss.NewStyle().
	Foreground(ErrorColor).
	Bold(true)

// HelpStyle is for the bottom help bar.
var HelpStyle = lipgloss.NewStyle().
	Foreground(MutedColor).
	Padding(0, 1)

// SpinnerStyle is used for the loading indicator.
var SpinnerStyle = lipgloss.NewStyle().
	Foreground(PrimaryColor)

// TenantStyle highlights tenant names.
var TenantStyle = lipgloss.NewStyle().
	Foreground(SecondaryColor).
	Bold(true)

// SelectedStyle is used for the cursor row of the FOCUSED pane
// (full cyan background). Defined above.

// CursorInactiveStyle marks the cursor row of a pane that does NOT
// currently have focus — a subtle outline instead of a full highlight,
// so the user can see where they left off in each pane.
var CursorInactiveStyle = lipgloss.NewStyle().
	Foreground(SecondaryColor).
	Bold(true)

// ActiveTenantStyle marks the tenant whose subscriptions are currently
// shown — a green check + bold so the active context is unmistakable.
var ActiveTenantStyle = lipgloss.NewStyle().
	Foreground(SuccessColor).
	Bold(true)

// PaneTitleFocusedStyle marks the header of the pane that has keyboard focus.
var PaneTitleFocusedStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FFFFFF")).
	Background(PrimaryColor).
	Bold(true).
	Padding(0, 1)

// PaneTitleBlurredStyle marks the header of an unfocused pane.
var PaneTitleBlurredStyle = lipgloss.NewStyle().
	Foreground(MutedColor).
	Bold(true)

// FilterStyle highlights the active filter indicator.
var FilterStyle = lipgloss.NewStyle().Foreground(WarningColor).Bold(true).Padding(0, 1)

// ButtonStyle is used for actionable button items in lists.
var ButtonStyle = lipgloss.NewStyle().Foreground(SuccessColor).Bold(true)
