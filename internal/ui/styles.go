package ui

import "charm.land/lipgloss/v2"

var (
	colorWhite = lipgloss.Color("#E0E0E0")
	colorDim   = lipgloss.Color("#6B6B6B")
	colorBlue  = lipgloss.Color("#6B9BF2")
	colorGreen = lipgloss.Color("#5CB85C")
	colorAmber = lipgloss.Color("#D4A03C")
	colorRed   = lipgloss.Color("#D96459")
	colorCyan  = lipgloss.Color("#5BB8C9")
	colorBdr   = lipgloss.Color("#444444")

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite).
			Padding(0, 1)

	styleSearch = lipgloss.NewStyle().
			Foreground(colorWhite)

	styleSearchPrompt = lipgloss.NewStyle().
				Foreground(colorBlue).
				Bold(true)

	styleTableHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorDim)

	styleSelected = lipgloss.NewStyle().
			Foreground(colorBlue).
			Bold(true)

	styleCursor = lipgloss.NewStyle().
			Foreground(colorWhite).
			Bold(true)

	styleUpToDate = lipgloss.NewStyle().
			Foreground(colorGreen)

	styleOutdated = lipgloss.NewStyle().
			Foreground(colorAmber)

	styleError = lipgloss.NewStyle().
			Foreground(colorRed)

	styleUpdating = lipgloss.NewStyle().
			Foreground(colorCyan)

	styleFooter = lipgloss.NewStyle().
			Foreground(colorDim)

	stylePopupBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBdr).
				Padding(0, 1)

	stylePopupTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite)

	styleDim = lipgloss.NewStyle().
			Foreground(colorDim)
)
