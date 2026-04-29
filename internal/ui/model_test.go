package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/aleksey925/deps/internal/cli"
	"github.com/aleksey925/deps/internal/python"
)

func TestSearchKey_PypiInfoHotkeyGoesToInput(t *testing.T) {
	// arrange
	m := newPypiSearchModel(t, []string{"requests", "rich"})

	// act
	updated, _ := m.handleSearchKey(keyRune('i'))

	// assert
	got := updated.(Model)
	if got.mode != viewSearch {
		t.Errorf("mode = %v, want viewSearch", got.mode)
	}
	if got.search.Value() != "reqi" {
		t.Errorf("search value = %q, want %q (i appended to input, not intercepted)", got.search.Value(), "reqi")
	}
	if got.packageInfo != nil || got.pypiLoading {
		t.Errorf("info should not be triggered: packageInfo=%v pypiLoading=%v", got.packageInfo, got.pypiLoading)
	}
}

func TestSearchKey_PypiDownTransitionsToPypiTable(t *testing.T) {
	// arrange
	m := newPypiSearchModel(t, []string{"requests"})

	// act
	updated, _ := m.handleSearchKey(keyCode(tea.KeyDown))

	// assert
	got := updated.(Model)
	if got.mode != viewPypiTable {
		t.Errorf("mode = %v, want viewPypiTable", got.mode)
	}
}

func TestSearchKey_PypiEnterTransitionsToPypiTable(t *testing.T) {
	// arrange
	m := newPypiSearchModel(t, []string{"requests"})

	// act
	updated, _ := m.handleSearchKey(keyCode(tea.KeyEnter))

	// assert
	got := updated.(Model)
	if got.mode != viewPypiTable {
		t.Errorf("mode = %v, want viewPypiTable", got.mode)
	}
}

func TestSearchKey_PypiDownNoOpOnEmptyResults(t *testing.T) {
	// arrange
	m := newPypiSearchModel(t, nil)

	// act
	updated, _ := m.handleSearchKey(keyCode(tea.KeyDown))

	// assert
	got := updated.(Model)
	if got.mode != viewSearch {
		t.Errorf("mode = %v, want viewSearch", got.mode)
	}
}

func TestPypiTableKey_InfoOpensLoadingState(t *testing.T) {
	// arrange
	m := newPypiTableModel(t, []string{"requests"})

	// act
	updated, cmd := m.handlePypiTableKey(keyRune('i'))

	// assert
	got := updated.(Model)
	if !got.pypiLoading {
		t.Errorf("pypiLoading = false, want true")
	}
	if got.pypiPackageName != "requests" {
		t.Errorf("pypiPackageName = %q, want %q", got.pypiPackageName, "requests")
	}
	if cmd == nil {
		t.Errorf("expected fetch cmd, got nil")
	}
}

func TestPypiTableKey_SlashRefocusesInput(t *testing.T) {
	// arrange
	m := newPypiTableModel(t, []string{"requests"})

	// act
	updated, _ := m.handlePypiTableKey(keyRune('/'))

	// assert
	got := updated.(Model)
	if got.mode != viewSearch {
		t.Errorf("mode = %v, want viewSearch", got.mode)
	}
	if got.search.Value() != "req" {
		t.Errorf("search value = %q, want preserved %q", got.search.Value(), "req")
	}
}

func TestPypiTableKey_TabSwitchesToLocalSearch(t *testing.T) {
	// arrange
	m := newPypiTableModel(t, []string{"requests"})

	// act
	updated, _ := m.handlePypiTableKey(keyCode(tea.KeyTab))

	// assert
	got := updated.(Model)
	if got.mode != viewSearch {
		t.Errorf("mode = %v, want viewSearch", got.mode)
	}
	if got.searchMode != searchLocal {
		t.Errorf("searchMode = %v, want searchLocal", got.searchMode)
	}
	if got.search.Value() != "req" {
		t.Errorf("search value = %q, want preserved %q", got.search.Value(), "req")
	}
}

func TestPypiTableKey_EscFullyExits(t *testing.T) {
	// arrange
	m := newPypiTableModel(t, []string{"requests"})

	// act
	updated, _ := m.handlePypiTableKey(keyCode(tea.KeyEscape))

	// assert
	got := updated.(Model)
	if got.mode != viewTable {
		t.Errorf("mode = %v, want viewTable", got.mode)
	}
	if got.searchMode != searchLocal {
		t.Errorf("searchMode = %v, want searchLocal", got.searchMode)
	}
	if got.search.Value() != "" {
		t.Errorf("search value = %q, want empty", got.search.Value())
	}
	if len(got.pypiResults) != 0 {
		t.Errorf("pypiResults len = %d, want 0", len(got.pypiResults))
	}
}

func TestVersionsKey_CancelDuringPypiInstallReturnsToPypiTable(t *testing.T) {
	// arrange
	m := newPypiTableModel(t, []string{"requests"})
	m.mode = viewVersions
	m.pypiInstall = true
	m.pypiPackageName = "requests"
	m.versions = []string{"2.31.0", "2.30.0"}

	// act
	updated, _ := m.handleVersionsKey(keyCode(tea.KeyEscape))

	// assert
	got := updated.(Model)
	if got.mode != viewPypiTable {
		t.Errorf("mode = %v, want viewPypiTable", got.mode)
	}
	if got.searchMode != searchPypi {
		t.Errorf("searchMode = %v, want searchPypi (preserved)", got.searchMode)
	}
	if got.pypiInstall {
		t.Errorf("pypiInstall should be reset to false")
	}
}

func TestVersionsKey_CancelOnLocalUpdateReturnsToTable(t *testing.T) {
	// arrange
	m := newLocalModel(t)
	m.mode = viewVersions
	m.versions = []string{"1.0.0"}

	// act
	updated, _ := m.handleVersionsKey(keyCode(tea.KeyEscape))

	// assert
	got := updated.(Model)
	if got.mode != viewTable {
		t.Errorf("mode = %v, want viewTable", got.mode)
	}
}

func TestVersionsKey_ConfirmInstallFlipsToLocalSynchronously(t *testing.T) {
	// arrange
	m := newPypiTableModel(t, []string{"requests"})
	m.mode = viewVersions
	m.pypiInstall = true
	m.pypiPackageName = "requests"
	m.versions = []string{"2.31.0"}
	m.verCursor = 0

	// act
	updated, cmd := m.handleVersionsKey(keyCode(tea.KeyEnter))

	// assert
	got := updated.(Model)
	if got.searchMode != searchLocal {
		t.Errorf("searchMode = %v, want searchLocal (synchronous flip)", got.searchMode)
	}
	if got.mode != viewTable {
		t.Errorf("mode = %v, want viewTable", got.mode)
	}
	if got.search.Value() != "" {
		t.Errorf("search value = %q, want empty", got.search.Value())
	}
	if cmd == nil {
		t.Errorf("expected installPackage cmd, got nil")
	}
}

func TestPypiTableKey_EnterStartsInstallAndKeepsPypiContext(t *testing.T) {
	// arrange
	m := newPypiTableModel(t, []string{"requests"})

	// act
	updated, cmd := m.handlePypiTableKey(keyCode(tea.KeyEnter))

	// assert
	got := updated.(Model)
	if got.mode != viewPypiTable {
		t.Errorf("mode = %v, want viewPypiTable (consistent with searchMode during versions fetch)", got.mode)
	}
	if got.searchMode != searchPypi {
		t.Errorf("searchMode = %v, want searchPypi", got.searchMode)
	}
	if !got.pypiInstall || got.pypiPackageName != "requests" {
		t.Errorf("install state not set: pypiInstall=%v pypiPackageName=%q", got.pypiInstall, got.pypiPackageName)
	}
	if cmd == nil {
		t.Errorf("expected fetchVersions cmd, got nil")
	}
}

func TestPypiTableKey_BlocksKeysWhileLoading(t *testing.T) {
	// arrange — pypi install fetch in flight (versions loading)
	m := newPypiTableModel(t, []string{"requests"})
	m.pypiInstall = true
	m.pypiLoading = true
	m.pypiPackageName = "requests"

	// act — Tab during loading window (would otherwise toggle searchMode and break invariants)
	updated, cmd := m.handlePypiTableKey(keyCode(tea.KeyTab))

	// assert
	got := updated.(Model)
	if got.searchMode != searchPypi {
		t.Errorf("searchMode = %v, want unchanged searchPypi while loading", got.searchMode)
	}
	if got.mode != viewPypiTable {
		t.Errorf("mode = %v, want unchanged viewPypiTable while loading", got.mode)
	}
	if cmd != nil {
		t.Errorf("expected no cmd while loading, got %v", cmd)
	}
}

func TestPypiTableKey_QuitWorksWhileLoading(t *testing.T) {
	// arrange
	m := newPypiTableModel(t, []string{"requests"})
	m.pypiLoading = true

	// act
	_, cmd := m.handlePypiTableKey(keyRune('q'))

	// assert
	if cmd == nil {
		t.Errorf("expected quit cmd, got nil")
	}
}

func TestUpdate_DispatchesToPypiTableHandler(t *testing.T) {
	// arrange
	m := newPypiTableModel(t, []string{"requests"})

	// act
	updatedModel, _ := m.Update(keyRune('i'))

	// assert
	got := updatedModel.(Model)
	if !got.pypiLoading || got.pypiPackageName != "requests" {
		t.Errorf("Update did not dispatch i to handlePypiTableKey: pypiLoading=%v pypiPackageName=%q",
			got.pypiLoading, got.pypiPackageName)
	}
}

func TestPackageInfoKey_EscReturnsToOriginatingTable(t *testing.T) {
	tests := []struct {
		name       string
		searchMode searchMode
		wantMode   viewMode
	}{
		{name: "from local", searchMode: searchLocal, wantMode: viewTable},
		{name: "from pypi", searchMode: searchPypi, wantMode: viewPypiTable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			m := newLocalModel(t)
			m.searchMode = tt.searchMode
			m.mode = viewPackageInfo
			m.packageInfo = &python.PackageInfo{Name: "x"}

			// act
			updated, _ := m.handlePackageInfoKey(keyCode(tea.KeyEscape))

			// assert
			got := updated.(Model)
			if got.mode != tt.wantMode {
				t.Errorf("mode = %v, want %v", got.mode, tt.wantMode)
			}
			if got.packageInfo != nil {
				t.Errorf("packageInfo should be cleared")
			}
		})
	}
}

// helpers

func newLocalModel(_ *testing.T) Model {
	env := &python.Environment{Manager: python.ManagerPip}
	m := NewModel(env, cli.BuildInfo{Version: "test"})
	return m
}

func newPypiSearchModel(t *testing.T, results []string) Model {
	m := newLocalModel(t)
	m.mode = viewSearch
	m.searchMode = searchPypi
	m.search.Focus()
	m.search.SetValue("req")
	m.pypiResults = results
	return m
}

func newPypiTableModel(t *testing.T, results []string) Model {
	m := newPypiSearchModel(t, results)
	m.mode = viewPypiTable
	m.search.Blur()
	return m
}

func keyRune(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

func keyCode(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}
