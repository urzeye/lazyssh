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
		_ = t.serverService.SetPinned(server.Alias, pinned)
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
		host := server.Host
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
		currentIdx := t.serverList.GetCurrentItem()
		itemCount := t.serverList.GetItemCount()
		if currentIdx < itemCount-1 {
			t.serverList.SetCurrentItem(currentIdx + 1)
		} else {
			t.serverList.SetCurrentItem(0)
		}
	}
}

func (t *tui) handleNavigateUp() {
	if t.isServerListFocused() {
		currentIdx := t.serverList.GetCurrentItem()
		if currentIdx > 0 {
			t.serverList.SetCurrentItem(currentIdx - 1)
		} else {
			t.serverList.SetCurrentItem(t.serverList.GetItemCount() - 1)
		}
	}
}

func (t *tui) handleSearchInput(query string) {
	filtered, _ := t.serverService.ListServers(query)
	if strings.TrimSpace(query) == "" {
		sortServersForUI(filtered, t.sortMode)
	}
	t.serverList.UpdateServers(filtered)
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
	var sshErr error
	if !t.app.Suspend(func() {
		if len(extraArgs) == 0 {
			sshErr = t.serverService.SSH(alias)
		} else {
			sshErr = t.serverService.SSHWithArgs(alias, extraArgs)
		}
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
			SetText(tview.Escape(fmt.Sprintf("SSH to %s failed:\n\n%v", alias, sshErr))).
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
		OnSave(t.handleServerSave).
		OnCancel(t.handleFormCancel)
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
			OnSave(t.handleServerSave).
			OnCancel(t.handleFormCancel)
		t.app.SetRoot(form, true)
	}
}

func (t *tui) handleServerSave(server domain.Server, original *domain.Server) {
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
			SetText(fmt.Sprintf("Save failed: %v", err)).
			AddButtons([]string{"Close"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) { t.handleModalClose() })
		t.app.SetRoot(modal, true)
		return
	}

	t.refreshServerList()
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
	currentIdx := t.serverList.GetCurrentItem()
	query := ""
	if t.searchBar != nil {
		query = t.searchBar.InputField.GetText()
	}

	t.showStatusTemp("Refreshing…")

	go func(prevIdx int, q string) {
		servers, err := t.serverService.ListServers(q)
		if err != nil {
			t.app.QueueUpdateDraw(func() {
				t.showStatusTempColor(fmt.Sprintf("Refresh failed: %v", err), CurrentTheme.Error)
			})
			return
		}
		sessions, sessionsErr := t.serverService.ListActiveSessions()
		if strings.TrimSpace(q) == "" {
			sortServersForUI(servers, t.sortMode)
		}
		t.app.QueueUpdateDraw(func() {
			t.serverList.UpdateServers(servers)
			if sessionsErr == nil && t.sessions != nil {
				t.sessions.SetSessions(sessions)
			}
			// Try to restore selection if still valid
			if prevIdx >= 0 && prevIdx < t.serverList.List.GetItemCount() {
				t.serverList.SetCurrentItem(prevIdx)
				if srv, ok := t.serverList.GetSelectedServer(); ok {
					t.details.UpdateServer(srv)
				}
			}
			if sessionsErr != nil {
				t.showStatusTempColor(fmt.Sprintf("Refreshed %d servers; active sessions failed", len(servers)), CurrentTheme.Warning)
				return
			}
			t.showStatusTemp(fmt.Sprintf("Refreshed %d servers, %d active sessions", len(servers), len(sessions)))
		})
	}(currentIdx, query)
}

// =============================================================================
// UI Display Functions (show UI elements/modals)
// =============================================================================

func (t *tui) showDeleteConfirmModal(server domain.Server) {
	msg := fmt.Sprintf("Delete server %s (%s@%s:%d)?\n\nThis action cannot be undone.",
		server.Alias, server.User, server.Host, server.Port)

	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"[yellow]C[-]ancel", "[yellow]D[-]elete"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonIndex == 1 {
				_ = t.serverService.DeleteServer(server)
				t.refreshServerList()
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
			_ = t.serverService.DeleteServer(server)
			t.refreshServerList()
			t.handleModalClose()
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

	form.AddButton("Start", func() {
		if err := validatePort(portVal); err != nil {
			t.showStatusTempColor("Invalid port: "+err.Error(), CurrentTheme.Error)
			return
		}
		if bindAddrVal != "" {
			if err := validateBindAddress(bindAddrVal); err != nil {
				t.showStatusTempColor("Invalid bind address: "+err.Error(), CurrentTheme.Error)
				return
			}
		}

		ft := typeChoices[currentTypeIdx]
		var args []string
		if ft == ForwardTypeDynamic {
			spec := portVal
			if bindAddrVal != "" {
				spec = bindAddrVal + ":" + portVal
			}
			args = append(args, "-D", spec)
		} else {
			if err := validateHost(hostVal); err != nil {
				t.showStatusTempColor("Invalid host: "+err.Error(), CurrentTheme.Error)
				return
			}
			if err := validatePort(hostPortVal); err != nil {
				t.showStatusTempColor("Invalid host port: "+err.Error(), CurrentTheme.Error)
				return
			}
			spec := portVal + ":" + hostVal + ":" + hostPortVal
			if bindAddrVal != "" {
				spec = bindAddrVal + ":" + spec
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

func (t *tui) refreshServerList() {
	query := ""
	if t.searchBar != nil {
		query = t.searchBar.InputField.GetText()
	}
	filtered, _ := t.serverService.ListServers(query)
	if strings.TrimSpace(query) == "" {
		sortServersForUI(filtered, t.sortMode)
	}
	t.serverList.UpdateServers(filtered)
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
	t.statusBar.SetText("[" + color + "]" + msg + "[-]")
	time.AfterFunc(2*time.Second, func() {
		if t.app != nil {
			t.app.QueueUpdateDraw(func() {
				if t.statusBar != nil {
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
