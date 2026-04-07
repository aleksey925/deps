package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	colName      = 28
	colInstalled = 15
	colLatest    = 15
	colStatus    = 15
)

func (m Model) View() tea.View {
	var b strings.Builder

	m.renderHeader(&b)
	m.renderSearchBar(&b)
	m.renderTable(&b)
	m.renderFooter(&b)

	content := b.String()

	switch m.mode {
	case viewVersions:
		content = m.overlayVersions(content)
	case viewConfirm:
		content = m.overlayConfirm(content)
	case viewUpdating:
		content = m.overlayUpdating(content)
	case viewReloadConfirm:
		content = m.overlayReloadConfirm(content)
	case viewPackageInfo:
		content = m.overlayPackageInfo(content)
	case viewTable, viewSearch:
		// no overlay
	}

	v := tea.NewView(content)
	v.AltScreen = true
	if m.mode == viewSearch {
		v.Cursor = m.search.Cursor()
	}
	return v
}

func (m *Model) renderHeader(b *strings.Builder) {
	title := fmt.Sprintf(" deps (%s) - %s (%s) (%s)",
		m.version, m.env.PythonPath, m.env.PythonVersion, m.env.Manager)
	header := styleHeader.Width(m.width).Render(title)
	b.WriteString(header)
	b.WriteString("\n\n")
}

func (m *Model) renderSearchBar(b *strings.Builder) {
	switch {
	case m.errMessage != "":
		b.WriteString(styleError.Render("  " + m.errMessage))
	case m.pypiLoading && m.pypiPackageName != "":
		b.WriteString(styleUpdating.Render("  ⏳ Fetching versions for " + m.pypiPackageName + "…"))
	case m.pypiLoading:
		b.WriteString(styleUpdating.Render("  ⏳ Loading PyPI package index…"))
	case m.mode == viewSearch || m.search.Value() != "":
		b.WriteString(m.search.View())
	default:
		b.WriteString(styleDim.Render("  / to search"))
	}
	b.WriteString("\n\n")
}

func (m *Model) renderTable(b *strings.Builder) {
	if m.searchMode == searchPypi && (m.mode == viewSearch || m.prevMode == viewSearch) {
		m.renderPypiTable(b)
		return
	}
	m.renderLocalTable(b)
}

func (m *Model) renderLocalTable(b *strings.Builder) {
	sortLabel := "Name A→Z"
	if m.sortMode == sortByStatus {
		sortLabel = "Outdated first"
	}

	header := fmt.Sprintf("  %-*s %-*s %-*s %-*s",
		colName, "Package",
		colInstalled, "Installed",
		colLatest, "Latest",
		colStatus, "Status")
	b.WriteString(styleTableHeader.Render(header))
	b.WriteString(styleDim.Render("  Sort: " + sortLabel))
	b.WriteString("\n")
	b.WriteString(styleDim.Render(strings.Repeat("─", m.width)))
	b.WriteString("\n")

	th := m.tableHeight()
	end := min(m.offset+th, len(m.filtered))

	if len(m.filtered) == 0 {
		b.WriteString(styleDim.Render("  No packages found"))
		if m.mode == viewSearch && m.search.Value() != "" {
			b.WriteString(styleDim.Render("  —  press "))
			b.WriteString(styleSearchPrompt.Render("Tab"))
			b.WriteString(styleDim.Render(" to search PyPI"))
		}
		b.WriteString("\n")
	}

	for vi := m.offset; vi < end; vi++ {
		idx := m.filtered[vi]
		item := m.packages[idx]
		isCursor := vi == m.cursor

		prefix := m.rowPrefix(item.selected, isCursor)
		name := truncate(item.pkg.Name, colName-1)
		installed := truncate(item.pkg.InstalledVersion, colInstalled-1)
		latest := truncate(item.pkg.LatestVersion, colLatest-1)
		statusStr, statusStyle := formatStatus(item.status)

		line := fmt.Sprintf("%-*s %-*s %-*s %s",
			colName, name,
			colInstalled, installed,
			colLatest, latest,
			statusStyle.Render(statusStr))

		switch {
		case isCursor:
			line = styleCursor.Render(prefix) + " " + line
		case item.selected:
			line = styleSelected.Render(prefix) + " " + line
		default:
			line = styleDim.Render(prefix) + " " + line
		}

		b.WriteString(line)
		b.WriteString("\n")
	}
}

func (m *Model) renderPypiTable(b *strings.Builder) {
	b.WriteString(styleTableHeader.Render(fmt.Sprintf("  %-*s", colName+colInstalled+colLatest+colStatus, "Package")))
	b.WriteString("\n")
	b.WriteString(styleDim.Render(strings.Repeat("─", m.width)))
	b.WriteString("\n")

	if m.pypiLoading {
		return
	}

	th := m.tableHeight()

	if len(m.pypiResults) == 0 {
		if m.search.Value() == "" {
			b.WriteString(styleDim.Render("  Type to search PyPI packages\n"))
		} else {
			b.WriteString(styleDim.Render("  No packages found on PyPI\n"))
		}
		return
	}

	end := min(m.pypiOffset+th, len(m.pypiResults))
	for vi := m.pypiOffset; vi < end; vi++ {
		name := m.pypiResults[vi]
		isCursor := vi == m.pypiCursor

		prefix := "  "
		if isCursor {
			prefix = "▸ "
		}

		line := truncate(name, m.width-6)
		if isCursor {
			b.WriteString(styleCursor.Render(prefix+line) + "\n")
		} else {
			b.WriteString(styleDim.Render(prefix) + line + "\n")
		}
	}
}

func (*Model) rowPrefix(selected, isCursor bool) string {
	switch {
	case selected && isCursor:
		return "▸◉"
	case selected:
		return " ◉"
	case isCursor:
		return "▸ "
	default:
		return "  "
	}
}

func formatStatus(s packageStatus) (string, lipgloss.Style) {
	switch s {
	case statusUpToDate:
		return "✓ up to date", styleUpToDate
	case statusOutdated:
		return "⬆ outdated", styleOutdated
	case statusUpdating:
		return "⏳ updating…", styleUpdating
	case statusError:
		return "✗ error", styleError
	default:
		return "… loading", styleDim
	}
}

func (m *Model) renderFooter(b *strings.Builder) {
	b.WriteString("\n")

	var hints string
	switch {
	case m.mode == viewSearch && m.searchMode == searchPypi:
		hints = "esc clear  tab local  ↑/↓ navigate  enter/→ versions  i info  ctrl+r reload"
	case m.mode == viewSearch:
		hints = "esc clear  tab PyPI  ↓ table  enter confirm"
	default:
		hints = "↑/↓ navigate  / search  s sort  space select  a outdated  enter update  → versions  i info  ctrl+r reload  q quit"
	}

	var right string
	if m.mode == viewSearch && m.searchMode == searchPypi {
		right = fmt.Sprintf("%d results", len(m.pypiResults))
	} else {
		selectedCount := 0
		for _, idx := range m.filtered {
			if m.packages[idx].selected {
				selectedCount++
			}
		}
		right = fmt.Sprintf("%d packages", len(m.filtered))
		if selectedCount > 0 {
			right = fmt.Sprintf("%d selected  %s", selectedCount, right)
		}
	}

	padding := max(1, m.width-lipgloss.Width(hints)-lipgloss.Width(right)-4)

	footer := fmt.Sprintf("  %s%s%s",
		styleFooter.Render(hints),
		strings.Repeat(" ", padding),
		styleFooter.Render(right))

	b.WriteString(footer)
}

func (m Model) overlayReloadConfirm(base string) string {
	title := stylePopupTitle.Render("Reload PyPI index?")

	ageStr := "unknown"
	if m.pypiIndex != nil {
		ageStr = formatAge(m.pypiIndex.Age())
	}

	lines := []string{
		title,
		"",
		fmt.Sprintf("  Current index loaded %s.", ageStr),
		"  This will download ~5MB of data.",
		"",
		styleFooter.Render("enter confirm  esc cancel"),
	}

	popup := stylePopupBorder.Render(strings.Join(lines, "\n"))
	return placeOverlay(m.width, m.height, popup, base)
}

func (m Model) overlayPackageInfo(base string) string {
	if m.packageInfo == nil {
		return base
	}

	info := m.packageInfo
	title := stylePopupTitle.Render(info.Name)

	lines := []string{title, ""}

	if info.Summary != "" {
		lines = append(lines, "  "+info.Summary, "")
	}

	if info.Version != "" {
		lines = append(lines, styleDim.Render("  Latest: ")+info.Version)
	}
	if info.Author != "" {
		lines = append(lines, styleDim.Render("  Author: ")+info.Author)
	}
	if info.License != "" {
		lines = append(lines, styleDim.Render("  License: ")+truncate(info.License, 40))
	}
	if info.RequiresPy != "" {
		lines = append(lines, styleDim.Render("  Python: ")+info.RequiresPy)
	}
	if info.HomePage != "" {
		lines = append(lines, styleDim.Render("  Home: ")+truncate(info.HomePage, 45))
	}

	lines = append(lines, "", styleFooter.Render("esc/enter close"))

	popup := stylePopupBorder.Render(strings.Join(lines, "\n"))
	return placeOverlay(m.width, m.height, popup, base)
}

func (m Model) overlayVersions(base string) string {
	var pkgName, installedVer, latestVer string

	if m.pypiInstall {
		pkgName = m.pypiPackageName
		if len(m.versions) > 0 {
			latestVer = m.versions[0]
		}
	} else {
		idx := m.currentPackageIdx()
		if idx < 0 {
			return base
		}
		pkg := m.packages[idx]
		pkgName = pkg.pkg.Name
		installedVer = pkg.pkg.InstalledVersion
		latestVer = pkg.pkg.LatestVersion
	}

	titlePrefix := "Versions"
	if m.pypiInstall {
		titlePrefix = "Install"
	}
	title := stylePopupTitle.Render(titlePrefix + " — " + pkgName)

	maxVisible := m.popupMaxItems()
	end := min(m.verOffset+maxVisible, len(m.versions))

	lines := make([]string, 0, end-m.verOffset+4)
	lines = append(lines, title, "")

	for i := m.verOffset; i < end; i++ {
		v := m.versions[i]
		prefix := "  "
		if i == m.verCursor {
			prefix = "▸ "
		}

		suffix := ""
		switch {
		case v == latestVer:
			suffix = styleDim.Render(" (latest)")
		case v == installedVer && installedVer != "":
			suffix = styleDim.Render(" (installed)")
		}

		if i == m.verCursor {
			lines = append(lines, styleCursor.Render(prefix+v)+suffix)
		} else {
			lines = append(lines, prefix+v+suffix)
		}
	}

	lines = append(lines, "", styleFooter.Render("↑/↓ select  enter install  ← back"))

	popup := stylePopupBorder.Render(strings.Join(lines, "\n"))
	return placeOverlay(m.width, m.height, popup, base)
}

func (m Model) overlayConfirm(base string) string {
	title := stylePopupTitle.Render(fmt.Sprintf("Update %d packages?", len(m.confirmPkgs)))

	maxVis := m.popupMaxItems()
	end := min(m.confirmOffset+maxVis, len(m.confirmPkgs))

	lines := make([]string, 0, end-m.confirmOffset+4)
	lines = append(lines, title, "")

	for _, idx := range m.confirmPkgs[m.confirmOffset:end] {
		pkg := m.packages[idx]
		lines = append(lines, fmt.Sprintf("  %s  %s → %s",
			pkg.pkg.Name,
			styleDim.Render(pkg.pkg.InstalledVersion),
			styleOutdated.Render(pkg.pkg.LatestVersion)))
	}

	lines = append(lines, "")

	hint := "enter confirm  esc cancel"
	if len(m.confirmPkgs) > maxVis {
		hint = "↑/↓ scroll  " + hint
	}
	lines = append(lines, styleFooter.Render(hint))

	popup := stylePopupBorder.Render(strings.Join(lines, "\n"))
	return placeOverlay(m.width, m.height, popup, base)
}

func (m Model) overlayUpdating(base string) string {
	allDone := len(m.updateDone) >= len(m.confirmPkgs)

	titleText := fmt.Sprintf("Updating %d packages…", len(m.confirmPkgs))
	if allDone {
		titleText = "Update complete"
	}
	title := stylePopupTitle.Render(titleText)

	maxVis := m.popupMaxItems()

	// auto-scroll to keep the currently updating package visible
	offset := 0
	if len(m.confirmPkgs) > maxVis {
		offset = min(m.updateIdx, len(m.confirmPkgs)-maxVis)
	}
	end := min(offset+maxVis, len(m.confirmPkgs))

	lines := make([]string, 0, end-offset+4)
	lines = append(lines, title, "")

	for i := offset; i < end; i++ {
		idx := m.confirmPkgs[i]
		pkg := m.packages[idx]
		statusIcon := m.updateStatusIcon(i)
		lines = append(lines, fmt.Sprintf("  %s %s  %s → %s",
			statusIcon,
			pkg.pkg.Name,
			styleDim.Render(pkg.pkg.InstalledVersion),
			styleOutdated.Render(pkg.pkg.LatestVersion)))
	}

	lines = append(lines, "")
	if allDone {
		lines = append(lines, styleFooter.Render("enter dismiss"))
	} else {
		lines = append(lines, styleFooter.Render("esc cancel remaining"))
	}

	popup := stylePopupBorder.Render(strings.Join(lines, "\n"))
	return placeOverlay(m.width, m.height, popup, base)
}

func (m *Model) updateStatusIcon(i int) string {
	switch {
	case i < len(m.updateDone) && m.updateDone[i].success:
		return styleUpToDate.Render("✓")
	case i < len(m.updateDone):
		return styleError.Render("✗")
	case i == len(m.updateDone):
		return styleUpdating.Render("⏳")
	default:
		return styleDim.Render("○")
	}
}

func placeOverlay(width, height int, popup, base string) string {
	popupLines := strings.Split(popup, "\n")
	baseLines := strings.Split(base, "\n")

	popupW := lipgloss.Width(popup)
	startX := max(0, (width-popupW)/2)
	startY := max(0, (height-len(popupLines))/2)

	for len(baseLines) < startY+len(popupLines) {
		baseLines = append(baseLines, "")
	}

	for i, pLine := range popupLines {
		y := startY + i
		if y >= len(baseLines) {
			break
		}

		baseLine := baseLines[y]
		baseW := ansi.StringWidth(baseLine)
		if baseW < startX {
			baseLine += strings.Repeat(" ", startX-baseW)
		}

		// use ansi-aware truncation to correctly handle escape codes
		left := ansi.Truncate(baseLine, startX, "")
		baseLines[y] = left + pLine
	}

	return strings.Join(baseLines, "\n")
}

func truncate(s string, maxLen int) string {
	if ansi.StringWidth(s) <= maxLen {
		return s
	}
	return ansi.Truncate(s, maxLen, "…")
}
