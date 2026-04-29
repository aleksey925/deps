package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/aleksey925/deps/internal/cli"
	"github.com/aleksey925/deps/internal/python"
)

type sortMode int

const (
	sortByName sortMode = iota
	sortByStatus
)

type viewMode int

const (
	viewTable viewMode = iota
	viewSearch
	viewPypiTable
	viewVersions
	viewConfirm
	viewUpdating
	viewReloadConfirm
	viewPackageInfo
)

type searchMode int

const (
	searchLocal searchMode = iota
	searchPypi
)

type packageStatus int

const (
	statusUnknown packageStatus = iota
	statusUpToDate
	statusOutdated
	statusUpdating
	statusError
)

type packageItem struct {
	pkg      python.Package
	status   packageStatus
	selected bool
	errMsg   string
}

type Model struct {
	build           cli.BuildInfo
	env             *python.Environment
	packages        []packageItem
	filtered        []int
	cursor          int
	offset          int
	height          int
	width           int
	mode            viewMode
	sortMode        sortMode
	search          textinput.Model
	versions        []string
	verCursor       int
	verOffset       int
	confirmPkgs     []int
	confirmOffset   int
	updateIdx       int
	updateDone      []updateResult
	errMessage      string
	searchMode      searchMode
	pypiIndex       *python.PackageIndex
	pypiResults     []string
	pypiCursor      int
	pypiOffset      int
	pypiInstall     bool
	pypiPackageName string
	pypiLoading     bool
	packageInfo     *python.PackageInfo
}

type updateResult struct {
	name    string
	to      string
	success bool
	errMsg  string
}

// messages

type packagesLoadedMsg struct {
	packages []python.Package
}

type latestVersionMsg struct {
	name    string
	version string
	err     error
}

type versionsLoadedMsg struct {
	versions []string
	err      error
}

type packageUpdatedMsg struct {
	name    string
	version string
	err     error
}

type pypiIndexLoadedMsg struct {
	index *python.PackageIndex
	err   error
}

type packageInfoMsg struct {
	info *python.PackageInfo
	err  error
}

func NewModel(env *python.Environment, build cli.BuildInfo) Model {
	ti := textinput.New()
	ti.Prompt = "  Search [local]: "
	ti.CharLimit = 100

	styles := textinput.DefaultDarkStyles()
	styles.Focused.Prompt = styleSearchPrompt
	styles.Focused.Text = styleSearch
	styles.Blurred.Prompt = styleSearchPrompt
	styles.Blurred.Text = styleSearch
	ti.SetStyles(styles)

	return Model{
		build:  build,
		env:    env,
		search: ti,
		mode:   viewTable,
		height: 24,
		width:  80,
	}
}

func (m *Model) setSearchPrompt() {
	switch m.searchMode {
	case searchPypi:
		if m.pypiIndex != nil {
			m.search.Prompt = fmt.Sprintf("  Search [PyPI] (index: %s): ", formatAge(m.pypiIndex.Age()))
		} else {
			m.search.Prompt = "  Search [PyPI]: "
		}
	default:
		m.search.Prompt = "  Search [local]: "
	}
}

func formatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func (m Model) Init() tea.Cmd {
	return loadPackages(m.env)
}

func loadPackages(env *python.Environment) tea.Cmd {
	return func() tea.Msg {
		pkgs, err := env.ListPackages()
		if err != nil {
			return packagesLoadedMsg{packages: nil}
		}
		return packagesLoadedMsg{packages: pkgs}
	}
}

func (m Model) reloadPackages() (tea.Model, tea.Cmd) {
	m.packages = nil
	m.filtered = nil
	m.cursor = 0
	m.offset = 0
	m.clearSelection()
	return m, loadPackages(m.env)
}

func fetchLatest(name string) tea.Cmd {
	return func() tea.Msg {
		ver, err := python.FetchLatestVersion(name)
		return latestVersionMsg{name: name, version: ver, err: err}
	}
}

func fetchVersions(name string) tea.Cmd {
	return func() tea.Msg {
		vers, err := python.FetchVersions(name)
		return versionsLoadedMsg{versions: vers, err: err}
	}
}

func installPackage(env *python.Environment, name, version string) tea.Cmd {
	return func() tea.Msg {
		err := env.InstallPackage(name, version)
		return packageUpdatedMsg{name: name, version: version, err: err}
	}
}

func fetchPackageInfo(name string) tea.Cmd {
	return func() tea.Msg {
		info, err := python.FetchPackageInfo(name)
		return packageInfoMsg{info: info, err: err}
	}
}

func loadPypiIndex(forceRefresh bool) tea.Cmd {
	return func() tea.Msg {
		if !forceRefresh {
			idx, err := python.LoadIndex()
			if err == nil && !idx.IsExpired() {
				return pypiIndexLoadedMsg{index: idx}
			}
		}

		idx, err := python.FetchIndex()
		return pypiIndexLoadedMsg{index: idx, err: err}
	}
}

func (m *Model) applyFilter() {
	query := strings.ToLower(m.search.Value())
	m.filtered = m.filtered[:0]

	for i := range m.packages {
		if query == "" || strings.Contains(strings.ToLower(m.packages[i].pkg.Name), query) {
			m.filtered = append(m.filtered, i)
		}
	}

	m.applySort()

	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m *Model) applyPypiFilter() {
	query := strings.TrimSpace(m.search.Value())
	m.pypiResults = python.SearchIndex(m.pypiIndex, query)
	if m.pypiCursor >= len(m.pypiResults) {
		m.pypiCursor = max(0, len(m.pypiResults)-1)
	}
	m.pypiOffset = 0
}

func (m *Model) applySort() {
	switch m.sortMode {
	case sortByName:
		sort.Slice(m.filtered, func(i, j int) bool {
			return strings.ToLower(m.packages[m.filtered[i]].pkg.Name) <
				strings.ToLower(m.packages[m.filtered[j]].pkg.Name)
		})
	case sortByStatus:
		sort.Slice(m.filtered, func(i, j int) bool {
			si := m.packages[m.filtered[i]]
			sj := m.packages[m.filtered[j]]
			if si.status != sj.status {
				return statusOrder(si.status) < statusOrder(sj.status)
			}
			return strings.ToLower(si.pkg.Name) < strings.ToLower(sj.pkg.Name)
		})
	}
}

func statusOrder(s packageStatus) int {
	switch s {
	case statusOutdated:
		return 0
	case statusUnknown:
		return 1
	case statusUpToDate:
		return 2
	case statusError:
		return 3
	case statusUpdating:
		return 4
	default:
		return 5
	}
}

func (m *Model) selectedPackages() []int {
	var sel []int
	for _, idx := range m.filtered {
		if m.packages[idx].selected {
			sel = append(sel, idx)
		}
	}
	return sel
}

func (m *Model) outdatedPackages() []int {
	var out []int
	for _, idx := range m.filtered {
		if m.packages[idx].status == statusOutdated {
			out = append(out, idx)
		}
	}
	return out
}

func (m *Model) currentPackageIdx() int {
	if len(m.filtered) == 0 {
		return -1
	}
	return m.filtered[m.cursor]
}

func (m *Model) tableHeight() int {
	// header(2) + search(2) + table header(2) + footer(2) = 8 lines overhead
	return max(3, m.height-8)
}

func (m *Model) popupMaxItems() int {
	// border(2) + title(1) + empty line(2) + footer hint(1) = 6 lines overhead
	return max(3, m.height-6)
}

func (m *Model) ensureCursorVisible() {
	th := m.tableHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+th {
		m.offset = m.cursor - th + 1
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case packagesLoadedMsg:
		return m.handlePackagesLoaded(msg)

	case latestVersionMsg:
		m.updateLatestVersion(msg)
		m.applyFilter()
		return m, nil

	case versionsLoadedMsg:
		m.pypiLoading = false
		if msg.err != nil {
			m.errMessage = fmt.Sprintf("Failed to fetch versions: %v", msg.err)
			m.pypiInstall = false
			m.pypiPackageName = ""
			m.mode = viewSearch
			return m, m.search.Focus()
		}
		m.versions = msg.versions
		m.verCursor = 0
		m.verOffset = 0
		m.mode = viewVersions
		return m, nil

	case packageInfoMsg:
		m.pypiLoading = false
		m.pypiPackageName = ""
		if msg.err != nil {
			m.errMessage = fmt.Sprintf("Failed to fetch package info: %v", msg.err)
		} else {
			m.packageInfo = msg.info
			m.mode = viewPackageInfo
		}
		return m, nil

	case pypiIndexLoadedMsg:
		m.pypiLoading = false
		if msg.err != nil {
			m.errMessage = fmt.Sprintf("Failed to load PyPI index: %v", msg.err)
		}
		if msg.index != nil {
			m.pypiIndex = msg.index
			m.setSearchPrompt()
			m.applyPypiFilter()
		}
		return m, nil

	case packageUpdatedMsg:
		return m.handlePackageUpdated(msg)

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	if m.mode == viewSearch {
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		m.applyFilter()
		return m, cmd
	}

	return m, nil
}

func (m Model) handlePackagesLoaded(msg packagesLoadedMsg) (tea.Model, tea.Cmd) {
	m.packages = make([]packageItem, len(msg.packages))
	for i, p := range msg.packages {
		m.packages[i] = packageItem{pkg: p, status: statusUnknown}
	}
	m.filtered = make([]int, len(m.packages))
	for i := range m.packages {
		m.filtered[i] = i
	}
	m.applyFilter()

	cmds := make([]tea.Cmd, 0, len(msg.packages))
	for _, p := range msg.packages {
		cmds = append(cmds, fetchLatest(p.Name))
	}
	return m, tea.Batch(cmds...)
}

func (m *Model) updateLatestVersion(msg latestVersionMsg) {
	for i := range m.packages {
		if m.packages[i].pkg.Name != msg.name {
			continue
		}
		if msg.err != nil {
			m.packages[i].pkg.LatestVersion = "?"
			m.packages[i].status = statusError
		} else {
			m.packages[i].pkg.LatestVersion = msg.version
			if m.packages[i].pkg.InstalledVersion == msg.version {
				m.packages[i].status = statusUpToDate
			} else {
				m.packages[i].status = statusOutdated
			}
		}
		return
	}
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.errMessage = ""

	switch m.mode {
	case viewSearch:
		return m.handleSearchKey(msg)
	case viewPypiTable:
		return m.handlePypiTableKey(msg)
	case viewVersions:
		return m.handleVersionsKey(msg)
	case viewConfirm:
		return m.handleConfirmKey(msg)
	case viewUpdating:
		return m.handleUpdatingKey(msg)
	case viewReloadConfirm:
		return m.handleReloadConfirmKey(msg)
	case viewPackageInfo:
		return m.handlePackageInfoKey(msg)
	case viewTable:
		return m.handleTableKey(msg)
	}
	return m, nil
}

func (m Model) handleTableKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// avoid races while async info fetch is in flight (see handlePypiTableKey)
	if m.pypiLoading {
		if key.Matches(msg, keys.Quit) {
			return m, tea.Quit
		}
		return m, nil
	}

	// vim-style navigation only in table mode
	switch msg.String() {
	case "k":
		m.moveUp()
		return m, nil
	case "j":
		m.moveDown()
		return m, nil
	case "l":
		return m.openVersions()
	case "h":
		// no-op in table, same as left
	case "i":
		return m.showPackageInfo()
	}

	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, keys.Up):
		m.moveUp()
	case key.Matches(msg, keys.Down):
		m.moveDown()
	case key.Matches(msg, keys.Search):
		return m.enterSearch()
	case key.Matches(msg, keys.Sort):
		m.toggleSort()
	case key.Matches(msg, keys.Select):
		m.toggleSelect()
	case key.Matches(msg, keys.SelectAll):
		m.toggleSelectOutdated()
	case key.Matches(msg, keys.SelectAllUp):
		m.toggleSelectAll()
	case key.Matches(msg, keys.Enter):
		return m.startUpdate()
	case key.Matches(msg, keys.Right):
		return m.openVersions()
	case key.Matches(msg, keys.Reload):
		return m.reloadPackages()
	case key.Matches(msg, keys.Escape):
		m.clearSelection()
	}
	return m, nil
}

func (m *Model) moveUp() {
	if m.cursor > 0 {
		m.cursor--
		m.ensureCursorVisible()
	}
}

func (m *Model) moveDown() {
	if m.cursor < len(m.filtered)-1 {
		m.cursor++
		m.ensureCursorVisible()
	}
}

func (m Model) enterSearch() (tea.Model, tea.Cmd) {
	m.mode = viewSearch
	return m, m.search.Focus()
}

func (m *Model) toggleSort() {
	if m.sortMode == sortByName {
		m.sortMode = sortByStatus
	} else {
		m.sortMode = sortByName
	}
	m.applyFilter()
}

func (m *Model) toggleSelect() {
	idx := m.currentPackageIdx()
	if idx >= 0 {
		m.packages[idx].selected = !m.packages[idx].selected
		m.moveDown()
	}
}

func (m *Model) toggleSelectOutdated() {
	outdated := m.outdatedPackages()
	allSelected := true
	for _, idx := range outdated {
		if !m.packages[idx].selected {
			allSelected = false
			break
		}
	}
	for _, idx := range outdated {
		m.packages[idx].selected = !allSelected
	}
}

func (m *Model) toggleSelectAll() {
	allSelected := true
	for _, idx := range m.filtered {
		if !m.packages[idx].selected {
			allSelected = false
			break
		}
	}
	for _, idx := range m.filtered {
		m.packages[idx].selected = !allSelected
	}
}

func (m Model) openVersions() (tea.Model, tea.Cmd) {
	idx := m.currentPackageIdx()
	if idx >= 0 {
		return m, fetchVersions(m.packages[idx].pkg.Name)
	}
	return m, nil
}

func (m Model) showPackageInfo() (tea.Model, tea.Cmd) {
	var name string
	if m.mode == viewPypiTable && len(m.pypiResults) > 0 {
		name = m.pypiResults[m.pypiCursor]
	} else {
		idx := m.currentPackageIdx()
		if idx < 0 {
			return m, nil
		}
		name = m.packages[idx].pkg.Name
	}
	m.pypiLoading = true
	m.pypiPackageName = name
	return m, fetchPackageInfo(name)
}

func (m *Model) clearSelection() {
	for i := range m.packages {
		m.packages[i].selected = false
	}
}

func (m Model) handleSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.mode = viewTable
		m.search.Blur()
		m.search.SetValue("")
		m.searchMode = searchLocal
		m.setSearchPrompt()
		m.applyFilter()
		return m, nil

	case key.Matches(msg, keys.Tab):
		return m.toggleSearchMode()

	case key.Matches(msg, keys.Reload):
		if m.searchMode == searchPypi && m.pypiIndex != nil {
			m.mode = viewReloadConfirm
			return m, nil
		}

	case key.Matches(msg, keys.Down), key.Matches(msg, keys.Enter):
		if m.searchMode == searchPypi {
			if len(m.pypiResults) == 0 {
				return m, nil
			}
			m.mode = viewPypiTable
			m.search.Blur()
			return m, nil
		}
		m.mode = viewTable
		m.search.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	switch m.searchMode {
	case searchLocal:
		m.applyFilter()
	case searchPypi:
		m.applyPypiFilter()
	}
	return m, cmd
}

func (m Model) toggleSearchMode() (tea.Model, tea.Cmd) {
	if m.searchMode == searchLocal {
		m.searchMode = searchPypi
		m.setSearchPrompt()
		m.applyPypiFilter()
		if m.pypiIndex == nil {
			m.pypiLoading = true
			return m, loadPypiIndex(false)
		}
		return m, nil
	}
	m.searchMode = searchLocal
	m.setSearchPrompt()
	m.applyFilter()
	return m, nil
}

func (m Model) installPypiFromSearch() (tea.Model, tea.Cmd) {
	name := m.pypiResults[m.pypiCursor]
	m.pypiInstall = true
	m.pypiLoading = true
	m.pypiPackageName = name
	m.search.Blur()
	m.mode = viewPypiTable
	return m, fetchVersions(name)
}

func (m Model) movePypiCursor(delta int) (tea.Model, tea.Cmd) {
	m.pypiCursor += delta
	if m.pypiCursor < 0 {
		m.pypiCursor = 0
	}
	if m.pypiCursor >= len(m.pypiResults) {
		m.pypiCursor = max(0, len(m.pypiResults)-1)
	}
	th := m.tableHeight()
	if m.pypiCursor < m.pypiOffset {
		m.pypiOffset = m.pypiCursor
	}
	if m.pypiCursor >= m.pypiOffset+th {
		m.pypiOffset = m.pypiCursor - th + 1
	}
	return m, nil
}

func (m Model) handlePypiTableKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// avoid races while async fetch (versions/info) is in flight: late-arriving
	// messages would otherwise overwrite mode/state changed by intermediate keystrokes
	if m.pypiLoading {
		if key.Matches(msg, keys.Quit) {
			return m, tea.Quit
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Escape):
		m.mode = viewTable
		m.search.SetValue("")
		m.searchMode = searchLocal
		m.setSearchPrompt()
		m.pypiResults = nil
		m.pypiCursor = 0
		m.pypiOffset = 0
		m.applyFilter()
		return m, nil

	case key.Matches(msg, keys.Search):
		m.mode = viewSearch
		return m, m.search.Focus()

	case key.Matches(msg, keys.Tab):
		m.searchMode = searchLocal
		m.setSearchPrompt()
		m.applyFilter()
		m.mode = viewSearch
		return m, m.search.Focus()

	case key.Matches(msg, keys.Reload):
		if m.pypiIndex != nil {
			m.mode = viewReloadConfirm
		}
		return m, nil

	case key.Matches(msg, keys.Up):
		return m.movePypiCursor(-1)

	case key.Matches(msg, keys.Down):
		return m.movePypiCursor(1)

	case key.Matches(msg, keys.Info):
		if len(m.pypiResults) > 0 {
			return m.showPackageInfo()
		}
		return m, nil

	case key.Matches(msg, keys.Enter), key.Matches(msg, keys.Right):
		if len(m.pypiResults) > 0 {
			return m.installPypiFromSearch()
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handlePackageInfoKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape), msg.Code == tea.KeyEnter:
		m.packageInfo = nil
		if m.searchMode == searchPypi {
			m.mode = viewPypiTable
		} else {
			m.mode = viewTable
		}
		return m, nil

	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) handleReloadConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEnter:
		m.pypiLoading = true
		m.mode = viewSearch
		return m, tea.Batch(m.search.Focus(), loadPypiIndex(true))

	case key.Matches(msg, keys.Escape):
		m.mode = viewSearch
		return m, m.search.Focus()

	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) handleVersionsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	maxVisible := m.popupMaxItems()

	switch {
	case key.Matches(msg, keys.Escape), key.Matches(msg, keys.Left):
		wasPypiInstall := m.pypiInstall
		m.pypiInstall = false
		m.pypiPackageName = ""
		if wasPypiInstall {
			m.mode = viewPypiTable
		} else {
			m.mode = viewTable
		}
		return m, nil

	case key.Matches(msg, keys.Up):
		if m.verCursor > 0 {
			m.verCursor--
			if m.verCursor < m.verOffset {
				m.verOffset = m.verCursor
			}
		}

	case key.Matches(msg, keys.Down):
		if m.verCursor < len(m.versions)-1 {
			m.verCursor++
			if m.verCursor >= m.verOffset+maxVisible {
				m.verOffset = m.verCursor - maxVisible + 1
			}
		}

	case msg.Code == tea.KeyEnter:
		if m.verCursor >= len(m.versions) {
			break
		}
		ver := m.versions[m.verCursor]

		if m.pypiInstall {
			m.searchMode = searchLocal
			m.setSearchPrompt()
			m.search.SetValue("")
			m.applyFilter()
			m.mode = viewTable
			return m, installPackage(m.env, m.pypiPackageName, ver)
		}

		idx := m.currentPackageIdx()
		if idx >= 0 {
			m.packages[idx].status = statusUpdating
			m.mode = viewTable
			return m, installPackage(m.env, m.packages[idx].pkg.Name, ver)
		}

	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == tea.KeyEnter:
		return m.beginBulkUpdate()

	case key.Matches(msg, keys.Up):
		if m.confirmOffset > 0 {
			m.confirmOffset--
		}
		return m, nil

	case key.Matches(msg, keys.Down):
		maxVis := m.popupMaxItems()
		if m.confirmOffset+maxVis < len(m.confirmPkgs) {
			m.confirmOffset++
		}
		return m, nil

	case key.Matches(msg, keys.Escape):
		m.mode = viewTable
		return m, nil

	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) handleUpdatingKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape), msg.Code == tea.KeyEnter:
		m.mode = viewTable
		return m, nil

	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) startUpdate() (tea.Model, tea.Cmd) {
	selected := m.selectedPackages()

	if len(selected) == 0 {
		idx := m.currentPackageIdx()
		if idx < 0 || m.packages[idx].status != statusOutdated {
			return m, nil
		}
		m.packages[idx].status = statusUpdating
		return m, installPackage(m.env, m.packages[idx].pkg.Name, m.packages[idx].pkg.LatestVersion)
	}

	if len(selected) == 1 {
		idx := selected[0]
		if m.packages[idx].status != statusOutdated {
			return m, nil
		}
		m.packages[idx].status = statusUpdating
		m.packages[idx].selected = false
		return m, installPackage(m.env, m.packages[idx].pkg.Name, m.packages[idx].pkg.LatestVersion)
	}

	m.confirmPkgs = selected
	m.confirmOffset = 0
	m.mode = viewConfirm
	return m, nil
}

func (m Model) beginBulkUpdate() (tea.Model, tea.Cmd) {
	m.mode = viewUpdating
	m.updateIdx = 0
	m.updateDone = nil

	for _, idx := range m.confirmPkgs {
		m.packages[idx].status = statusUpdating
		m.packages[idx].selected = false
	}

	if len(m.confirmPkgs) > 0 {
		idx := m.confirmPkgs[0]
		return m, installPackage(m.env, m.packages[idx].pkg.Name, m.packages[idx].pkg.LatestVersion)
	}

	return m, nil
}

func (m Model) handlePackageUpdated(msg packageUpdatedMsg) (tea.Model, tea.Cmd) {
	wasNewInstall := m.pypiInstall
	m.applyPackageUpdate(msg)

	if m.mode == viewUpdating {
		result := updateResult{
			name:    msg.name,
			to:      msg.version,
			success: msg.err == nil,
		}
		if msg.err != nil {
			result.errMsg = msg.err.Error()
		}
		m.updateDone = append(m.updateDone, result)

		m.updateIdx++
		if m.updateIdx < len(m.confirmPkgs) {
			idx := m.confirmPkgs[m.updateIdx]
			return m, installPackage(m.env, m.packages[idx].pkg.Name, m.packages[idx].pkg.LatestVersion)
		}
	}

	m.applyFilter()

	if wasNewInstall {
		m.searchMode = searchLocal
		m.setSearchPrompt()
		if msg.err == nil {
			m.search.SetValue("")
			m.applyFilter()
			return m, fetchLatest(msg.name)
		}
		m.errMessage = fmt.Sprintf("Failed to install %s: %v", msg.name, msg.err)
		m.search.SetValue("")
		m.applyFilter()
	}

	return m, nil
}

func (m *Model) applyPackageUpdate(msg packageUpdatedMsg) {
	for i := range m.packages {
		if m.packages[i].pkg.Name != msg.name {
			continue
		}
		if msg.err != nil {
			m.packages[i].status = statusError
			m.packages[i].errMsg = msg.err.Error()
		} else {
			m.packages[i].pkg.InstalledVersion = msg.version
			if msg.version == m.packages[i].pkg.LatestVersion {
				m.packages[i].status = statusUpToDate
			} else {
				m.packages[i].status = statusOutdated
			}
		}
		m.pypiInstall = false
		m.pypiPackageName = ""
		return
	}

	// package not in list — new install from PyPI
	if msg.err == nil {
		m.packages = append(m.packages, packageItem{
			pkg: python.Package{
				Name:             msg.name,
				InstalledVersion: msg.version,
				LatestVersion:    "…",
			},
			status: statusUnknown,
		})
	}
	m.pypiInstall = false
	m.pypiPackageName = ""
}
