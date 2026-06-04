// Copyright 2025.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ui

import (
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/rivo/tview"
	"github.com/urzeye/lazyssh/internal/core/ports"
)

type App interface {
	Run() error
}

type tui struct {
	logger *zap.SugaredLogger

	version string
	commit  string

	app           *tview.Application
	serverService ports.ServerService
	settings      *settingsManager

	header     *AppHeader
	hintBar    *tview.TextView
	searchBar  *SearchBar
	serverList *ServerList
	sessions   *ActiveSessions
	details    *ServerDetails
	statusBar  *tview.TextView

	root    *tview.Flex
	left    *tview.Flex
	content *tview.Flex

	sortMode      SortMode
	themeMode     ThemeMode
	searchVisible bool

	activeSessionsRefreshStop chan struct{}
}

type mainUIState struct {
	query         string
	selectedAlias string
	selectedIndex int
	searchVisible bool
	focus         string
}

const (
	focusSearch     = "search"
	focusServerList = "server-list"
	focusDetails    = "details"
)

const activeSessionsRefreshInterval = 10 * time.Second

func NewTUI(logger *zap.SugaredLogger, ss ports.ServerService, version, commit string) App {
	return &tui{
		logger:        logger,
		app:           tview.NewApplication(),
		serverService: ss,
		version:       version,
		commit:        commit,
		settings:      newSettingsManager(logger),
		sortMode:      SortByAliasAsc,
		themeMode:     ThemeDark,
	}
}

func (t *tui) Run() error {
	defer func() {
		if r := recover(); r != nil {
			t.logger.Errorw("panic recovered", "error", r)
		}
	}()
	t.app.EnableMouse(true)
	t.loadPreferences().initializeTheme().buildComponents().buildLayout().bindEvents()
	t.loadInitialData()
	t.app.SetRoot(t.root, true)
	t.startActiveSessionsRefresh()
	defer t.stopActiveSessionsRefresh()
	t.logger.Infow("starting TUI application", "version", t.version, "commit", t.commit)
	if err := t.app.Run(); err != nil {
		t.logger.Errorw("application run error", "error", err)
		return err
	}
	return nil
}

func (t *tui) initializeTheme() *tui {
	SetCurrentTheme(t.themeMode)
	tview.Styles.PrimitiveBackgroundColor = CurrentTheme.PrimitiveBackground
	tview.Styles.ContrastBackgroundColor = CurrentTheme.ContrastBackground
	tview.Styles.BorderColor = CurrentTheme.Border
	tview.Styles.TitleColor = CurrentTheme.Title
	tview.Styles.PrimaryTextColor = CurrentTheme.PrimaryText
	tview.Styles.TertiaryTextColor = CurrentTheme.TertiaryText
	tview.Styles.SecondaryTextColor = CurrentTheme.SecondaryText
	tview.Styles.GraphicsColor = CurrentTheme.Graphics
	return t
}

func (t *tui) buildComponents() *tui {
	t.header = NewAppHeader(t.version, t.commit, RepoURL)
	t.hintBar = NewHintBar()
	t.searchBar = NewSearchBar().
		OnSearch(t.handleSearchInput).
		OnEscape(t.hideSearchBar).
		OnNavigate(t.handleSearchNavigate)
	IsForwarding = t.serverService.IsForwarding

	t.serverList = NewServerList().
		OnSelectionChange(t.handleServerSelectionChange).
		OnReturnToSearch(t.handleReturnToSearch)
	t.sessions = NewActiveSessions()
	t.details = NewServerDetails()
	t.statusBar = NewStatusBar()

	t.searchVisible = false

	return t
}

func (t *tui) loadPreferences() *tui {
	if t.settings == nil {
		return t
	}

	if mode, err := t.settings.LoadSortMode(); err == nil {
		t.sortMode = mode
	} else {
		t.logger.Warnw("failed to load sort mode preference", "error", err)
	}
	if mode, err := t.settings.LoadThemeMode(); err == nil {
		t.themeMode = mode
	} else {
		t.logger.Warnw("failed to load theme preference", "error", err)
	}

	return t
}

func (t *tui) buildLayout() *tui {
	t.left = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.hintBar, 1, 0, false).
		AddItem(t.serverList, 0, 1, true).
		AddItem(t.sessions, 6, 0, false)

	right := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.details, 0, 1, false)

	t.content = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(t.left, 0, 3, true).
		AddItem(right, 0, 2, false)

	t.root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t.header, 2, 0, false).
		AddItem(t.content, 0, 1, true).
		AddItem(t.statusBar, 1, 0, false)
	return t
}

func (t *tui) bindEvents() {
	t.root.SetInputCapture(t.handleGlobalKeys)
}

func (t *tui) loadInitialData() {
	t.loadServerList("")
	t.loadActiveSessions()
}

func (t *tui) loadServerList(query string) {
	servers, _ := t.serverService.ListServers(query)
	if strings.TrimSpace(query) == "" {
		sortServersForUI(servers, t.sortMode)
	}
	t.updateListTitle()
	t.serverList.UpdateServers(servers)
}

func (t *tui) loadActiveSessions() {
	if t.sessions == nil {
		return
	}
	sessions, err := t.serverService.ListActiveSessions()
	if err != nil {
		t.sessions.SetText(colorText(CurrentTheme.Error, "Failed to load active sessions"))
		return
	}
	t.sessions.SetSessions(sessions)
}

func (t *tui) startActiveSessionsRefresh() {
	if t.app == nil || t.serverService == nil || t.activeSessionsRefreshStop != nil {
		return
	}

	stop := make(chan struct{})
	t.activeSessionsRefreshStop = stop

	go func() {
		ticker := time.NewTicker(activeSessionsRefreshInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				t.refreshActiveSessionsFromBackground()
			case <-stop:
				return
			}
		}
	}()
}

func (t *tui) stopActiveSessionsRefresh() {
	if t.activeSessionsRefreshStop == nil {
		return
	}
	close(t.activeSessionsRefreshStop)
	t.activeSessionsRefreshStop = nil
}

func (t *tui) refreshActiveSessionsFromBackground() {
	sessions, err := t.serverService.ListActiveSessions()
	if t.app == nil {
		return
	}

	t.app.QueueUpdateDraw(func() {
		if t.sessions == nil {
			return
		}
		if err != nil {
			t.sessions.SetText(colorText(CurrentTheme.Error, "Failed to load active sessions"))
			return
		}
		t.sessions.SetSessions(sessions)
	})
}

func (t *tui) updateListTitle() {
	if t.serverList != nil {
		t.serverList.SetTitle(" 1 Servers — Sort: " + t.sortMode.String() + " ")
	}
}

func (t *tui) persistSortMode() {
	if t.settings == nil {
		return
	}

	if err := t.settings.SaveSortMode(t.sortMode); err != nil {
		t.logger.Warnw("failed to save sort mode preference", "error", err)
	}
}

func (t *tui) persistThemeMode() {
	if t.settings == nil {
		return
	}

	if err := t.settings.SaveThemeMode(t.themeMode); err != nil {
		t.logger.Warnw("failed to save theme preference", "error", err)
	}
}

func (t *tui) rebuildMainUI() {
	state := t.captureMainUIState()
	t.initializeTheme().buildComponents().buildLayout().bindEvents()
	t.restoreMainUIState(state)
	if t.app != nil {
		t.app.SetRoot(t.root, true)
	}
}

func (t *tui) captureMainUIState() mainUIState {
	state := mainUIState{selectedIndex: -1}
	if t.searchBar != nil {
		state.query = t.searchBar.GetText()
	}
	if t.serverList != nil {
		state.selectedIndex = t.serverList.GetCurrentItem()
		if server, ok := t.serverList.GetSelectedServer(); ok {
			state.selectedAlias = server.Alias
		}
	}
	state.searchVisible = t.searchVisible

	if t.app != nil {
		focus := t.app.GetFocus()
		switch {
		case focus == t.searchBar:
			state.focus = focusSearch
		case t.isServerListFocused():
			state.focus = focusServerList
		case focus == t.details:
			state.focus = focusDetails
		}
	}

	return state
}

func (t *tui) restoreMainUIState(state mainUIState) {
	if state.searchVisible {
		t.showSearchBar()
	}
	if t.searchBar != nil && state.query != "" {
		t.searchBar.SetText(state.query)
	}

	t.loadServerList(state.query)
	t.loadActiveSessions()
	t.restoreSelectedServer(state)
	t.restoreFocus(state)
}

func (t *tui) restoreSelectedServer(state mainUIState) {
	if t.serverList == nil {
		return
	}
	if state.selectedAlias != "" && t.serverList.SelectServerByAlias(state.selectedAlias) {
		return
	}
	_ = t.serverList.SelectIndex(state.selectedIndex)
}

func (t *tui) restoreFocus(state mainUIState) {
	if t.app == nil {
		return
	}
	switch state.focus {
	case focusSearch:
		if t.searchVisible && t.searchBar != nil {
			t.app.SetFocus(t.searchBar)
		}
	case focusDetails:
		if t.details != nil {
			t.app.SetFocus(t.details)
		}
	case focusServerList:
		if t.serverList != nil {
			t.app.SetFocus(t.serverList)
		}
	}
}
