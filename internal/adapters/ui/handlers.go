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
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/urzeye/lazyssh/internal/core/domain"
)

// =============================================================================
// Event Handlers (handle user input/events)
// =============================================================================
const (
	ForwardTypeLocal   = "Local"
	ForwardTypeRemote  = "Remote"
	ForwardTypeDynamic = "Dynamic"

	ForwardModeOnlyForward = "Only forward"
	ForwardModeForwardSSH  = "Forward + SSH"
)

func (t *tui) handleGlobalKeys(event *tcell.EventKey) *tcell.EventKey {
	// Don't handle global keys when search has focus
	if t.searchBar != nil && t.searchBar.HasFocus() {
		return event
	}

	switch event.Rune() {
	case '0':
		t.handleSearchFocus()
		return nil
	case '1':
		t.handleServerListFocus()
		return nil
	case '2':
		t.handleDetailsFocus()
		return nil
	case 'q':
		t.handleQuit()
		return nil
	case '/':
		t.handleSearchFocus()
		return nil
	case 'a':
		t.handleServerAdd()
		return nil
	case 'C':
		t.handleServerClone()
		return nil
	case 'v':
		t.handlePasteSSH()
		return nil
	case 'P':
		t.handleSCP()
		return nil
	case 'F':
		t.handleSFTP()
		return nil
	case 'e':
		t.handleServerEdit()
		return nil
	case 'd':
		t.handleServerDelete()
		return nil
	case 'p':
		t.handleServerPin()
		return nil
	case 's':
		t.handleSortToggle()
		return nil
	case 'S':
		t.handleSortReverse()
		return nil
	case 'T':
		t.handleThemeToggle()
		return nil
	case 'c':
		t.handleCopyCommand()
		return nil
	case 'h':
		t.handleCopyHost()
		return nil
	case 'g':
		t.handlePingSelected()
		return nil
	case 'r':
		t.handleRefreshBackground()
		return nil
	case 't':
		t.handleTagsEdit()
		return nil
	case 'f':
		t.handlePortForward()
		return nil
	case 'x':
		t.handleStopForwarding()
		return nil
	case 'j':
		t.handleNavigateDown()
		return nil
	case 'k':
		t.handleNavigateUp()
		return nil
	}

	if event.Key() == tcell.KeyEnter {
		t.handleServerConnect()
		return nil
	}

	return event
}

func (t *tui) handleQuit() {
	t.app.Stop()
}

func (t *tui) handleServerPin() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		pinned := server.PinnedAt.IsZero()
		if err := t.serverService.SetPinned(server.Alias, pinned); err != nil {
			t.showStatusTempColor("Failed to update pin: "+err.Error(), CurrentTheme.Error)
			return
		}
		t.refreshServerList()
	}
}

func (t *tui) handleSortToggle() {
	t.sortMode = t.sortMode.ToggleField()
	t.showStatusTemp("Sort: " + t.sortMode.String())
	t.updateListTitle()
	t.persistSortMode()
	t.refreshServerList()
}

func (t *tui) handleSortReverse() {
	t.sortMode = t.sortMode.Reverse()
	t.showStatusTemp("Sort: " + t.sortMode.String())
	t.updateListTitle()
	t.persistSortMode()
	t.refreshServerList()
}

func (t *tui) handleThemeToggle() {
	t.themeMode = t.themeMode.Toggle()
	t.persistThemeMode()
	t.rebuildMainUI()
	t.showStatusTemp("Theme: " + t.themeMode.String())
}

func (t *tui) handleCopyCommand() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		cmd := BuildSSHCommand(server)
		if err := clipboard.WriteAll(cmd); err == nil {
			t.showStatusTemp("Copied: " + cmd)
		} else {
			t.showStatusTemp("Failed to copy to clipboard")
		}
	}
}

func (t *tui) handleCopyHost() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		host := strings.TrimSpace(server.Host)
		if host == "" {
			host = server.Alias
		}
		if err := clipboard.WriteAll(host); err == nil {
			t.showStatusTemp("Copied: " + host)
		} else {
			t.showStatusTemp("Failed to copy to clipboard")
		}
	}
}

func (t *tui) handleTagsEdit() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		if server.Readonly {
			t.showStatusTempColor(fmt.Sprintf("Read-only: %s is managed in %s", server.Alias, server.SourceFile), CurrentTheme.Warning)
			return
		}
		t.showEditTagsForm(server)
	}
}

func (t *tui) handleNavigateDown() {
	if t.isServerListFocused() {
		itemCount := t.serverList.GetItemCount()
		if itemCount == 0 {
			return
		}
		currentIdx := t.serverList.GetCurrentItem()
		if currentIdx < itemCount-1 {
			t.serverList.SetCurrentItem(currentIdx + 1)
		} else {
			t.serverList.SetCurrentItem(0)
		}
		return
	}
	if t.app != nil && t.details != nil && t.app.GetFocus() == t.details {
		row, col := t.details.GetScrollOffset()
		t.details.ScrollTo(row+1, col)
	}
}

func (t *tui) handleNavigateUp() {
	if t.isServerListFocused() {
		itemCount := t.serverList.GetItemCount()
		if itemCount == 0 {
			return
		}
		currentIdx := t.serverList.GetCurrentItem()
		if currentIdx > 0 {
			t.serverList.SetCurrentItem(currentIdx - 1)
		} else {
			t.serverList.SetCurrentItem(itemCount - 1)
		}
		return
	}
	if t.app != nil && t.details != nil && t.app.GetFocus() == t.details {
		row, col := t.details.GetScrollOffset()
		if row > 0 {
			t.details.ScrollTo(row-1, col)
		}
	}
}

func (t *tui) handleSearchInput(query string) {
	filtered, err := t.serverService.ListServers(query)
	if err != nil {
		if t.logger != nil {
			t.logger.Errorw("failed to filter servers", "query", query, "error", err)
		}
		t.showStatusTempColor("Search failed: "+err.Error(), CurrentTheme.Error)
		return
	}
	if strings.TrimSpace(query) == "" {
		sortServersForUI(filtered, t.sortMode)
	}
	t.serverList.UpdateServers(filtered)
	_ = t.serverList.SelectIndex(0) // A changed search selects its best ranked match.
	if len(filtered) == 0 {
		t.details.ShowEmpty()
	}
}

func (t *tui) isServerListFocused() bool {
	if t == nil || t.app == nil || t.serverList == nil {
		return false
	}
	focus := t.app.GetFocus()
	return focus == t.serverList || focus == t.serverList.List
}

func (t *tui) handleSearchFocus() {
	if t.app != nil && t.searchBar != nil {
		if !t.searchVisible {
			t.showSearchBar()
			return
		}
		t.app.SetFocus(t.searchBar)
	}
}

func (t *tui) handleServerListFocus() {
	if t.app != nil && t.serverList != nil {
		t.app.SetFocus(t.serverList)
	}
}

func (t *tui) handleDetailsFocus() {
	if t.app != nil && t.details != nil {
		t.app.SetFocus(t.details)
	}
}

func (t *tui) handleSearchNavigate(direction int) {
	if t.serverList != nil {
		t.app.SetFocus(t.serverList)

		currentIdx := t.serverList.GetCurrentItem()
		itemCount := t.serverList.GetItemCount()

		if itemCount == 0 {
			return
		}

		if direction > 0 {
			if currentIdx < itemCount-1 {
				t.serverList.SetCurrentItem(currentIdx + 1)
			} else {
				t.serverList.SetCurrentItem(0)
			}
		} else {
			if currentIdx > 0 {
				t.serverList.SetCurrentItem(currentIdx - 1)
			} else {
				t.serverList.SetCurrentItem(itemCount - 1)
			}
		}

		if server, ok := t.serverList.GetSelectedServer(); ok {
			t.details.UpdateServer(server)
		}
	}
}

func (t *tui) handleReturnToSearch() {
	t.handleSearchFocus()
}

func (t *tui) handleServerConnect() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		t.runSSHSession(server.Alias, nil)
	}
}

func (t *tui) runSSHSession(alias string, extraArgs []string) {
	t.runInteractiveSession("SSH to "+alias, func() error {
		if len(extraArgs) == 0 {
			return t.serverService.SSH(alias)
		}
		return t.serverService.SSHWithArgs(alias, extraArgs)
	})
}

func (t *tui) handleSFTP() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		t.runInteractiveSession("SFTP to "+server.Alias, func() error { return t.serverService.SFTP(server.Alias) })
	}
}

func (t *tui) runInteractiveSession(label string, run func() error) {
	var sshErr error
	if !t.app.Suspend(func() {
		sshErr = run()
	}) {
		sshErr = fmt.Errorf("could not suspend the terminal for SSH")
	}
	if t.options.ExitAfterSession {
		t.sessionErr = sshErr
		t.app.Stop()
		return
	}
	t.refreshServerList()
	t.returnToMain()
	if sshErr != nil {
		modal := tview.NewModal().
			SetText(tview.Escape(fmt.Sprintf("%s failed:\n\n%v", label, sshErr))).
			AddButtons([]string{"Close"}).
			SetDoneFunc(func(_ int, _ string) { t.returnToMain() })
		t.app.SetRoot(modal, true)
	}
}

func (t *tui) handleServerSelectionChange(server domain.Server) {
	t.details.UpdateServer(server)
}

func (t *tui) handleServerAdd() {
	t.showAddServerForm(nil)
}

func (t *tui) showAddServerForm(prefill *domain.Server) {
	if t.serverService.IsReadOnly() {
		t.showStatusTempColor("Read-only mode: SSH config changes are disabled", CurrentTheme.Warning)
		return
	}

	form := NewServerForm(ServerFormAdd, prefill).
		SetApp(t.app).
		SetVersionInfo(t.version, t.commit).
		OnCancel(t.handleFormCancel)
	form.OnSave(func(server domain.Server, original *domain.Server) { t.handleServerSave(server, original, form) })
	t.app.SetRoot(form, true)
}

func (t *tui) handleServerClone() {
	if t.serverService.IsReadOnly() {
		t.showStatusTempColor("Read-only mode: SSH config changes are disabled", CurrentTheme.Warning)
		return
	}
	server, ok := t.serverList.GetSelectedServer()
	if !ok {
		return
	}
	servers, err := t.serverService.ListServers("")
	if err != nil {
		t.showStatusTempColor("Could not load hosts: "+err.Error(), CurrentTheme.Error)
		return
	}
	clone := cloneServer(server, servers)
	t.showAddServerForm(&clone)
}

func (t *tui) handlePasteSSH() {
	if t.serverService.IsReadOnly() {
		t.showStatusTempColor("Read-only mode: SSH config changes are disabled", CurrentTheme.Warning)
		return
	}
	text, err := clipboard.ReadAll()
	if err != nil {
		t.showActionError("Read clipboard", err)
		return
	}
	server, err := ParseSSHCommand(text)
	if err != nil {
		t.showActionError("Import SSH command", err)
		return
	}
	servers, err := t.serverService.ListServers("")
	if err != nil {
		t.showActionError("Load hosts", err)
		return
	}
	server.Alias = uniqueAlias(server.Alias, servers)
	t.showAddServerForm(&server)
}

func (t *tui) showActionError(action string, err error) {
	modal := tview.NewModal().SetText(tview.Escape(fmt.Sprintf("%s failed:\n\n%v", action, err))).
		AddButtons([]string{"Close"}).SetDoneFunc(func(_ int, _ string) { t.returnToMain() })
	t.app.SetRoot(modal, true)
}

func (t *tui) handleServerEdit() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		if server.Readonly {
			t.showStatusTempColor(fmt.Sprintf("Read-only: %s is managed in %s", server.Alias, server.SourceFile), CurrentTheme.Warning)
			return
		}
		form := NewServerForm(ServerFormEdit, &server).
			SetApp(t.app).
			SetVersionInfo(t.version, t.commit).
			OnCancel(t.handleFormCancel)
		form.OnSave(func(updated domain.Server, original *domain.Server) { t.handleServerSave(updated, original, form) })
		t.app.SetRoot(form, true)
	}
}

func (t *tui) handleServerSave(server domain.Server, original *domain.Server, form *ServerForm) {
	var err error
	if original != nil {
		// Edit mode
		err = t.serverService.UpdateServer(*original, server)
	} else {
		// Add mode
		err = t.serverService.AddServer(server)
	}
	if err != nil {
		// Stay on form; show a small modal with the error
		modal := tview.NewModal().
			SetText(tview.Escape(fmt.Sprintf("Save failed: %v", err))).
			AddButtons([]string{"Close"}).
			SetDoneFunc(func(_ int, _ string) { t.app.SetRoot(form, true) })
		t.app.SetRoot(modal, true)
		return
	}

	t.refreshServerList()
	t.serverList.SelectServerByAlias(server.Alias)
	t.handleFormCancel()
}

func (t *tui) handleServerDelete() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		if server.Readonly {
			t.showStatusTempColor(fmt.Sprintf("Read-only: %s is managed in %s", server.Alias, server.SourceFile), CurrentTheme.Error)
			return
		}
		t.showDeleteConfirmModal(server)
	}
}

func (t *tui) handleFormCancel() {
	t.returnToMain()
}

func (t *tui) handlePingSelected() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		alias := server.Alias

		t.showStatusTemp(fmt.Sprintf("Pinging %s…", alias))
		go func() {
			up, dur, err := t.serverService.Ping(server)
			t.app.QueueUpdateDraw(func() {
				if err != nil {
					t.showStatusTempColor(fmt.Sprintf("Ping %s: DOWN (%v)", alias, err), CurrentTheme.Error)
					return
				}
				if up {
					t.showStatusTempColor(fmt.Sprintf("Ping %s: UP (%s)", alias, dur), CurrentTheme.Success)
				} else {
					t.showStatusTempColor(fmt.Sprintf("Ping %s: DOWN", alias), CurrentTheme.Error)
				}
			})
		}()
	}
}

func (t *tui) handleModalClose() {
	t.returnToMain()
}

// handleRefreshBackground refreshes the server list in the background without leaving the current screen.
// It preserves the current search query and selection, shows transient status, and avoids concurrent runs.
func (t *tui) handleRefreshBackground() {
	if !t.refreshInProgress.CompareAndSwap(false, true) {
		return
	}

	query := ""
	if t.searchBar != nil {
		query = t.searchBar.InputField.GetText()
	}
	sortMode := t.sortMode

	t.showStatusTemp("Refreshing…")

	go func(q string, mode SortMode) {
		defer t.refreshInProgress.Store(false)

		servers, err := t.serverService.ListServers(q)
		if t.app == nil {
			return
		}
		if err != nil {
			t.app.QueueUpdateDraw(func() {
				t.showStatusTempColor(fmt.Sprintf("Refresh failed: %v", err), CurrentTheme.Error)
			})
			return
		}
		sessions, sessionsErr := t.serverService.ListActiveSessions()
		if strings.TrimSpace(q) == "" {
			sortServersForUI(servers, mode)
		}
		t.app.QueueUpdateDraw(func() {
			currentQuery := ""
			if t.searchBar != nil {
				currentQuery = t.searchBar.InputField.GetText()
			}
			// Re-sort if the user changed sort mode while refresh was running,
			// and avoid overwriting results for a newer search query.
			if currentQuery == q {
				if strings.TrimSpace(q) == "" && t.sortMode != mode {
					sortServersForUI(servers, t.sortMode)
				}
				t.serverList.UpdateServers(servers)
				if len(servers) == 0 {
					t.details.ShowEmpty()
				}
			}
			if sessionsErr == nil && t.sessions != nil {
				t.sessions.SetSessions(sessions)
			}
			if sessionsErr != nil {
				t.showStatusTempColor(fmt.Sprintf("Refreshed %d servers; active sessions failed", len(servers)), CurrentTheme.Warning)
				return
			}
			t.showStatusTemp(fmt.Sprintf("Refreshed %d servers, %d active sessions", len(servers), len(sessions)))
		})
	}(query, sortMode)
}

// =============================================================================
// UI Display Functions (show UI elements/modals)
// =============================================================================

func (t *tui) showDeleteConfirmModal(server domain.Server) {
	msg := tview.Escape(fmt.Sprintf("Delete server %s (%s@%s:%d)?\n\nThis action cannot be undone.",
		server.Alias, server.User, server.Host, server.Port))

	confirmDelete := func() {
		if err := t.serverService.DeleteServer(server); err != nil {
			t.showActionError("Delete server", err)
			return
		}
		t.refreshServerList()
		t.handleModalClose()
	}

	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"[yellow]C[-]ancel", "[yellow]D[-]elete"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 1 {
				confirmDelete()
				return
			}
			t.handleModalClose()
		})

	// Add keyboard shortcuts for the modal
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Rune() {
		case 'c', 'C':
			// Cancel
			t.handleModalClose()
			return nil
		case 'd', 'D':
			// Delete
			confirmDelete()
			return nil
		}
		// ESC key already handled by default modal behavior
		return event
	})

	t.app.SetRoot(modal, true)
}

func (t *tui) showEditTagsForm(server domain.Server) {
	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(fmt.Sprintf(" Edit Tags: %s ", server.Alias)).
		SetTitleAlign(tview.AlignCenter)

	defaultTags := strings.Join(server.Tags, ", ")
	form.AddInputField("Tags (comma):", defaultTags, 40, nil, nil)

	form.AddButton("Save", func() {
		text := strings.TrimSpace(form.GetFormItem(0).(*tview.InputField).GetText())
		var tags []string

		for _, part := range strings.Split(text, ",") {
			if s := strings.TrimSpace(part); s != "" {
				tags = append(tags, s)
			}
		}

		if err := t.serverService.UpdateTags(server, tags); err != nil {
			modal := tview.NewModal().
				SetText(tview.Escape(fmt.Sprintf("Save tags failed: %v", err))).
				AddButtons([]string{"Close"}).
				SetDoneFunc(func(_ int, _ string) { t.app.SetRoot(form, true) })
			t.app.SetRoot(modal, true)
			return
		}
		// Refresh UI and go back
		t.refreshServerList()
		t.returnToMain()
		t.showStatusTemp("Tags updated")
	})
	form.AddButton("Cancel", func() { t.returnToMain() })
	form.SetCancelFunc(func() { t.returnToMain() })

	t.app.SetRoot(form, true)
	toFocus := form
	t.app.SetFocus(toFocus)
}

func (t *tui) handlePortForward() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		t.showPortForwardForm(server)
	}
}

func (t *tui) showPortForwardForm(server domain.Server) {
	typeChoices := []string{ForwardTypeLocal, ForwardTypeRemote, ForwardTypeDynamic}
	modeChoices := []string{ForwardModeOnlyForward, ForwardModeForwardSSH}

	currentTypeIdx := 0
	currentModeIdx := 0
	portVal := ""
	hostVal := "localhost"
	hostPortVal := ""
	bindAddrVal := ""

	form := tview.NewForm()
	form.SetBorder(true).
		SetTitle(fmt.Sprintf(" Port Forwarding: %s ", server.Alias)).
		SetTitleAlign(tview.AlignCenter)

	dd := tview.NewDropDown()
	hostField := tview.NewInputField()
	hostPortField := tview.NewInputField()
	portField := tview.NewInputField()
	bindAddrField := tview.NewInputField()

	dd.SetOptions(typeChoices, func(text string, index int) {
		currentTypeIdx = index
		// Toggle fields when switching type
		isDynamic := typeChoices[currentTypeIdx] == ForwardTypeDynamic
		if isDynamic {
			hostField.SetText("").SetDisabled(true)
			hostPortField.SetText("").SetDisabled(true)
		} else {
			hostField.SetDisabled(false)
			hostPortField.SetDisabled(false)
		}
	})
	dd.SetCurrentOption(currentTypeIdx)
	form.AddFormItem(dd.SetLabel("Type"))

	portField.SetLabel("Port").SetText(portVal).SetFieldWidth(8).SetChangedFunc(func(text string) { portVal = strings.TrimSpace(text) })
	form.AddFormItem(portField)

	hostField.SetLabel("Host").SetText(hostVal).SetFieldWidth(40).SetChangedFunc(func(text string) { hostVal = strings.TrimSpace(text) })
	form.AddFormItem(hostField)

	hostPortField.SetLabel("Host Port").SetText(hostPortVal).SetFieldWidth(8).SetChangedFunc(func(text string) { hostPortVal = strings.TrimSpace(text) })
	form.AddFormItem(hostPortField)

	bindAddrField.SetLabel("Bind Address (optional)").SetText(bindAddrVal).SetFieldWidth(40).SetChangedFunc(func(text string) { bindAddrVal = strings.TrimSpace(text) })
	form.AddFormItem(bindAddrField)

	mode := tview.NewDropDown().SetOptions(modeChoices, func(text string, index int) { currentModeIdx = index })
	mode.SetCurrentOption(currentModeIdx)
	form.AddFormItem(mode.SetLabel("Mode"))

	isDynamic := typeChoices[currentTypeIdx] == ForwardTypeDynamic
	if isDynamic {
		hostField.SetText("").SetDisabled(true)
		hostPortField.SetText("").SetDisabled(true)
	}

	showFormError := func(msg string) {
		modal := tview.NewModal().
			SetText(tview.Escape(msg)).
			AddButtons([]string{"Close"}).
			SetDoneFunc(func(_ int, _ string) {
				t.app.SetRoot(form, true)
				t.app.SetFocus(form)
			})
		t.app.SetRoot(modal, true)
	}

	form.AddButton("Start", func() {
		if strings.TrimSpace(portVal) == "" {
			showFormError("Port is required")
			return
		}
		if err := validatePort(portVal); err != nil {
			showFormError("Invalid port: " + err.Error())
			return
		}
		if bindAddrVal != "" {
			if err := validateBindAddress(bindAddrVal); err != nil {
				showFormError("Invalid bind address: " + err.Error())
				return
			}
		}

		ft := typeChoices[currentTypeIdx]
		var args []string
		if ft == ForwardTypeDynamic {
			spec := portVal
			if bindAddrVal != "" {
				spec = formatForwardAddrPort(bindAddrVal, portVal)
			}
			args = append(args, "-D", spec)
		} else {
			if err := validateHost(hostVal); err != nil {
				showFormError("Invalid host: " + err.Error())
				return
			}
			if strings.TrimSpace(hostPortVal) == "" {
				showFormError("Host Port is required")
				return
			}
			if err := validatePort(hostPortVal); err != nil {
				showFormError("Invalid host port: " + err.Error())
				return
			}
			spec := portVal + ":" + formatForwardAddrPort(hostVal, hostPortVal)
			if bindAddrVal != "" {
				spec = formatForwardBindPrefix(bindAddrVal) + ":" + spec
			}
			if ft == ForwardTypeLocal {
				args = append(args, "-L", spec)
			} else {
				args = append(args, "-R", spec)
			}
		}

		onlyForward := modeChoices[currentModeIdx] == ForwardModeOnlyForward
		alias := server.Alias
		if onlyForward {
			t.returnToMain()
			t.showStatusTemp("Starting port forward…")
			go func() {
				pid, err := t.serverService.StartForward(alias, args)
				t.app.QueueUpdateDraw(func() {
					if err != nil {
						t.showStatusTempColor("Forward failed: "+err.Error(), CurrentTheme.Error)
					} else {
						t.refreshServerList()
						t.showStatusTemp(fmt.Sprintf("Port forwarding started (pid %d)", pid))
					}
				})
			}()
			return
		}

		t.runSSHSession(alias, args)
	})
	form.AddButton("Cancel", func() { t.returnToMain() })
	form.SetCancelFunc(func() { t.returnToMain() })

	t.app.SetRoot(form, true)
	t.app.SetFocus(form)
}

// =============================================================================
// UI State Management (hide UI elements)
// =============================================================================

func (t *tui) showSearchBar() {
	if t.left == nil || t.searchBar == nil || t.serverList == nil {
		return
	}

	t.left.Clear()
	t.left.AddItem(t.searchBar, 3, 0, true)
	t.left.AddItem(t.serverList, 0, 1, false)
	if t.sessions != nil {
		t.left.AddItem(t.sessions, 6, 0, false)
	}
	t.searchVisible = true
	if t.app != nil {
		t.app.SetFocus(t.searchBar)
	}
}

func (t *tui) hideSearchBar() {
	if t.searchBar != nil && t.searchBar.GetText() != "" {
		t.searchBar.SetText("")
	}

	if t.left == nil || t.hintBar == nil || t.serverList == nil {
		return
	}

	t.left.Clear()
	t.left.AddItem(t.hintBar, 1, 0, false)
	t.left.AddItem(t.serverList, 0, 1, true)
	if t.sessions != nil {
		t.left.AddItem(t.sessions, 6, 0, false)
	}
	t.searchVisible = false
	if t.app != nil {
		t.app.SetFocus(t.serverList)
	}
}

// =============================================================================
// Internal Operations (perform actual work)
// =============================================================================

func formatForwardAddrPort(addr, port string) string {
	if strings.Contains(addr, ":") && !strings.HasPrefix(addr, "[") {
		return "[" + addr + "]:" + port
	}
	return addr + ":" + port
}

func formatForwardBindPrefix(addr string) string {
	if strings.Contains(addr, ":") && !strings.HasPrefix(addr, "[") {
		return "[" + addr + "]"
	}
	return addr
}

func (t *tui) refreshServerList() {
	query := ""
	if t.searchBar != nil {
		query = t.searchBar.InputField.GetText()
	}
	filtered, err := t.serverService.ListServers(query)
	if err != nil {
		if t.logger != nil {
			t.logger.Errorw("failed to refresh server list", "query", query, "error", err)
		}
		t.showStatusTempColor("Failed to refresh servers: "+err.Error(), CurrentTheme.Error)
		return
	}
	if strings.TrimSpace(query) == "" {
		sortServersForUI(filtered, t.sortMode)
	}
	t.serverList.UpdateServers(filtered)
	if len(filtered) == 0 && t.details != nil {
		t.details.ShowEmpty()
	}
	t.loadActiveSessions()
}

func (t *tui) returnToMain() {
	t.app.SetRoot(t.root, true)
}

// showStatusTemp displays a temporary message in the status bar (default green) and then restores the default text.
func (t *tui) showStatusTemp(msg string) {
	if t.statusBar == nil {
		return
	}
	t.showStatusTempColor(msg, CurrentTheme.Success)
}

// showStatusTempColor displays a temporary colored message in the status bar and restores default text after 2s.
func (t *tui) showStatusTempColor(msg string, color string) {
	if t.statusBar == nil {
		return
	}
	seq := t.statusSeq.Add(1)
	t.statusBar.SetText("[" + color + "]" + tview.Escape(msg) + "[-]")
	time.AfterFunc(2*time.Second, func() {
		if t.app != nil {
			t.app.QueueUpdateDraw(func() {
				if t.statusBar != nil && t.statusSeq.Load() == seq {
					t.statusBar.SetText(DefaultStatusText())
				}
			})
		}
	})
}

// Stop any active port forwarding for the selected server.
func (t *tui) handleStopForwarding() {
	if server, ok := t.serverList.GetSelectedServer(); ok {
		alias := server.Alias
		go func() {
			err := t.serverService.StopForwarding(alias)
			t.app.QueueUpdateDraw(func() {
				if err != nil {
					t.showStatusTempColor("Failed to stop forwarding: "+err.Error(), CurrentTheme.Error)
				} else {
					t.showStatusTemp("Stopped forwarding for " + alias)
				}
				t.refreshServerList()
			})
		}()
	}
}
