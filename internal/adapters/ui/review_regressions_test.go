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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/urzeye/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

func TestEditFormHasNoUnsavedChangesInitially(t *testing.T) {
	original := domain.Server{
		Alias:      "prod",
		Aliases:    []string{"prod", "prod-alt"},
		Host:       "10.0.0.1",
		User:       "deploy",
		Port:       22,
		SourceFile: "/home/user/.ssh/config",
		Readonly:   false,
	}
	form := NewServerForm(ServerFormEdit, &original).SetVersionInfo("test", "test")
	if form.hasUnsavedChanges() {
		t.Fatal("unchanged edit form reported unsaved changes")
	}
}

func TestServerFormShortcutsInInputFieldPreserveBackspace(t *testing.T) {
	app := tview.NewApplication()
	original := domain.Server{Alias: "prod", Host: "10.0.0.1", Port: 22}
	form := NewServerForm(ServerFormEdit, &original).SetApp(app).SetVersionInfo("test", "test")

	// Focus the first InputField (Alias).
	basicForm := form.forms["Basic"]
	app.SetRoot(form, true)
	app.SetFocus(basicForm.GetFormItem(0))
	if !form.isInputFieldFocused() {
		t.Fatal("expected focus on input field")
	}

	// Ctrl+H (ASCII 8 / Backspace on many terminals) must pass through for text editing.
	ctrlH := tcell.NewEventKey(tcell.KeyCtrlH, 0, tcell.ModCtrl)
	if got := form.handleFormShortcut(ctrlH); got != ctrlH {
		t.Fatal("Ctrl+H was consumed while input field had focus")
	}

	// Ctrl+Right / Ctrl+Left / Ctrl+L should switch tabs even when an input field has focus.
	ctrlRight := tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModCtrl)
	if got := form.handleFormShortcut(ctrlRight); got != nil || form.currentTab != "Connection" {
		t.Fatalf("Ctrl+Right did not advance tab: got=%v tab=%q", got, form.currentTab)
	}
	ctrlLeft := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModCtrl)
	if got := form.handleFormShortcut(ctrlLeft); got != nil || form.currentTab != "Basic" {
		t.Fatalf("Ctrl+Left did not return to previous tab: got=%v tab=%q", got, form.currentTab)
	}

	// Ctrl+S should save even when an input field has focus.
	saved := false
	form.OnSave(func(domain.Server, *domain.Server) { saved = true })
	ctrlS := tcell.NewEventKey(tcell.KeyCtrlS, 0, tcell.ModCtrl)
	if got := form.handleFormShortcut(ctrlS); got != nil || !saved {
		t.Fatalf("Ctrl+S did not save from input field: got=%v saved=%v", got, saved)
	}

	// Esc should pass through root capture to the active form and cancel when no popup is open.
	canceled := false
	form.OnCancel(func() { canceled = true })
	esc := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)
	if got := form.handleFormShortcut(esc); got != esc {
		t.Fatalf("Esc should pass through to active form so popups can consume it first: got=%v", got)
	}
	basicForm.GetFormItem(0).InputHandler()(esc, func(p tview.Primitive) { app.SetFocus(p) })
	if !canceled {
		t.Fatal("Esc did not cancel form when input field had no open popup")
	}
}

func TestServerFormEscapeClosesPopupsBeforeCancelingAndModalEscRestoresForm(t *testing.T) {
	app := tview.NewApplication()
	original := domain.Server{Alias: "prod", Host: "10.0.0.1", Port: 22}
	form := NewServerForm(ServerFormEdit, &original).SetApp(app).SetVersionInfo("test", "test")
	app.SetRoot(form, true)

	canceled := false
	form.OnCancel(func() { canceled = true })
	esc := tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone)

	// 1. Open DropDown: first Esc closes the dropdown, second Esc cancels the form.
	form.switchToTab("Connection")
	connForm := form.forms["Connection"]
	var dd *tview.DropDown
	for i := 0; i < connForm.GetFormItemCount(); i++ {
		if d, ok := connForm.GetFormItem(i).(*tview.DropDown); ok {
			dd = d
			break
		}
	}
	if dd == nil {
		t.Fatal("expected DropDown in Connection form")
	}
	app.SetFocus(dd)
	dd.InputHandler()(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), func(p tview.Primitive) { app.SetFocus(p) })
	if !dd.IsOpen() {
		t.Fatal("DropDown did not open")
	}
	if got := form.Flex.GetInputCapture()(esc); got != esc {
		t.Fatal("root input capture consumed Esc while DropDown was open")
	}
	// Dispatch Esc to the focused dropdown list capture (same order as tview.Application).
	focusedList := app.GetFocus().(*tview.List)
	if got := focusedList.GetInputCapture()(esc); got != nil {
		t.Fatal("open DropDown list did not consume Esc")
	}
	if dd.IsOpen() || canceled {
		t.Fatalf("first Esc on open DropDown should only close it: isOpen=%v canceled=%v", dd.IsOpen(), canceled)
	}
	// Second Esc on closed DropDown cancels the form via Form.SetCancelFunc.
	dd.InputHandler()(esc, func(p tview.Primitive) { app.SetFocus(p) })
	if !canceled {
		t.Fatal("second Esc on closed DropDown did not cancel form")
	}

	// 2. Open InputField autocomplete: first Esc closes autocomplete, second Esc cancels the form.
	canceled = false
	form.switchToTab("Basic")
	basicForm := form.forms["Basic"]
	aliasField := basicForm.GetFormItem(0).(*tview.InputField)
	aliasField.SetAutocompleteFunc(func(string) []string { return []string{"prod-a", "prod-b"} })
	app.SetFocus(aliasField)
	aliasField.Autocomplete()
	if got := form.Flex.GetInputCapture()(esc); got != esc {
		t.Fatal("root input capture consumed Esc while InputField autocomplete was open")
	}
	aliasField.InputHandler()(esc, func(p tview.Primitive) { app.SetFocus(p) })
	if canceled {
		t.Fatal("first Esc with open autocomplete should close autocomplete without canceling form")
	}
	aliasField.InputHandler()(esc, func(p tview.Primitive) { app.SetFocus(p) })
	if !canceled {
		t.Fatal("second Esc after closing autocomplete did not cancel form")
	}

	// 3. Focus outside the active form (e.g., helpPanel): Esc falls back to handleCancel.
	canceled = false
	app.SetFocus(form.helpPanel)
	if got := form.Flex.GetInputCapture()(esc); got != nil || !canceled {
		t.Fatalf("Esc outside active form did not fall back to cancel: got=%v canceled=%v", got, canceled)
	}

	// 4. Unsaved-changes confirmation modal: pressing Esc restores the form view.
	canceled = false
	basicForm.GetFormItem(1).(*tview.InputField).SetText("10.0.0.2")
	form.handleCancel()
	if form.Flex.HasFocus() {
		t.Fatal("expected unsaved-changes modal to take focus")
	}
	// Modal binds Esc on its focused button's exit/cancel handler via tview.Form.
	app.GetFocus().InputHandler()(esc, func(p tview.Primitive) { app.SetFocus(p) })
	if !form.Flex.HasFocus() || canceled {
		t.Fatalf("Esc on unsaved-changes modal should restore form without discarding: formFocused=%v canceled=%v", form.Flex.HasFocus(), canceled)
	}
}

func TestServerFormHelpSeparatorMatchesRenderedPanelWidth(t *testing.T) {
	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()

	original := domain.Server{Alias: "prod", Host: "10.0.0.1", Port: 22}
	form := NewServerForm(ServerFormEdit, &original).SetVersionInfo("test", "test")

	countSeparator := func(text string) int {
		n := 0
		for _, r := range text {
			if r == '─' {
				n++
			}
		}
		return n
	}

	assertHelpWidth := func(screenW int, minInner int) int {
		screen.SetSize(screenW, 40)
		form.SetRect(0, 0, screenW, 40)
		form.Draw(screen)
		ox, oy, ow, oh := form.helpPanel.GetRect()
		ix, iy, iw, ih := form.helpPanel.GetInnerRect()
		if ix != ox+2 || iy != oy+1 || iw != ow-4 || ih != oh-2 || iw <= minInner {
			t.Fatalf("screenW=%d helpPanel inner=(%d,%d,%d,%d) outer=(%d,%d,%d,%d), want inset (+2,+1,-4,-2) and width>%d",
				screenW, ix, iy, iw, ih, ox, oy, ow, oh, minInner)
		}
		if got := countSeparator(form.helpPanel.GetText(true)); got != iw {
			t.Fatalf("screenW=%d help separator width = %d, want %d", screenW, got, iw)
		}
		return iw
	}

	width140 := assertHelpWidth(140, 40)
	_ = assertHelpWidth(180, width140)
}

func TestServerFormPreservesUnavailableBindInterfaceAndZeroPort(t *testing.T) {
	seed := domain.Server{
		Alias:         "vpn-host",
		Host:          "10.8.0.1",
		Port:          0,
		BindInterface: "wg-unavailable99",
	}
	form := NewServerForm(ServerFormAdd, &seed).SetVersionInfo("test", "test")
	data := form.getFormData()
	if data.BindInterface != "wg-unavailable99" {
		t.Fatalf("BindInterface = %q, want wg-unavailable99", data.BindInterface)
	}
	if data.Port == "0" {
		t.Fatal("zero Port prefilled invalid '0'")
	}
	var saved domain.Server
	form.OnSave(func(s domain.Server, _ *domain.Server) { saved = s })
	if !form.handleSave() || saved.BindInterface != "wg-unavailable99" || saved.Port != 22 {
		t.Fatalf("save lost BindInterface or normalized port wrongly: %+v", saved)
	}
}

func TestValidationIPv6ForwardsNumericIPQoSAndTildePaths(t *testing.T) {
	for _, valid := range []string{
		"[::1]:8080:localhost:80",
		"8080:[2001:db8::1]:80",
		"[::1]:8080:[2001:db8::1]:443",
	} {
		if err := validatePortForward(valid); err != nil {
			t.Errorf("validatePortForward(%q) unexpected error: %v", valid, err)
		}
	}
	if err := validatePortForward("8080::80"); err == nil {
		t.Error("validatePortForward accepted empty target host")
	}
	if err := validateDynamicForward("[::1]:1080"); err != nil {
		t.Errorf("validateDynamicForward([::1]:1080) unexpected error: %v", err)
	}

	for _, qos := range []string{"46", "0x10", "0x10 af21"} {
		if err := validateIPQoS(qos); err != nil {
			t.Errorf("validateIPQoS(%q) unexpected error: %v", qos, err)
		}
	}

	if err := validateAlias("-bad"); err == nil {
		t.Error("validateAlias accepted leading hyphen")
	}
	if err := validateHost("bad_host.local"); err == nil {
		t.Error("validateHost accepted underscore inconsistent with service validation")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "id_test"), []byte("k"), 0o600); err != nil {
		t.Fatal(err)
	}
	if exists, _, isDir := validateFilePath("~nonexistent"); exists || isDir {
		t.Fatalf("~nonexistent should not resolve to home dir: exists=%v isDir=%v", exists, isDir)
	}
	if exists, accessible, isDir := validateFilePath(`~\.ssh\id_test`); !exists || !accessible || isDir {
		t.Fatalf(`~\.ssh\id_test failed validation: exists=%v accessible=%v isDir=%v`, exists, accessible, isDir)
	}
}

func TestBuildSSHCommandQuotesSpacesAndPreservesExplicitOptions(t *testing.T) {
	cmd := BuildSSHCommand(domain.Server{
		Alias:             "web",
		Host:              "10.0.0.1",
		IPQoS:             "af21 cs1",
		CanonicalDomains:  "corp.internal example.com",
		ForwardX11:        "no",
		ForwardX11Trusted: "yes",
		Compression:       "no",
		BatchMode:         "no",
		LogLevel:          "VERBOSE",
	})

	if !strings.Contains(cmd, "-o 'IPQoS=af21 cs1'") {
		t.Fatalf("IPQoS with space was not quoted: %s", cmd)
	}
	if !strings.Contains(cmd, "-o 'CanonicalDomains=corp.internal example.com'") {
		t.Fatalf("CanonicalDomains with space was not quoted: %s", cmd)
	}
	if strings.Contains(cmd, " -Y ") || !strings.Contains(cmd, " -x ") {
		t.Fatalf("ForwardX11=no should not emit -Y: %s", cmd)
	}
	if !strings.Contains(cmd, "-o Compression=no") || !strings.Contains(cmd, "-o BatchMode=no") {
		t.Fatalf("explicit 'no' options were dropped: %s", cmd)
	}
	if !strings.Contains(cmd, "-o LogLevel=VERBOSE") || strings.Contains(cmd, " -v ") {
		t.Fatalf("LogLevel=VERBOSE should use -o LogLevel=VERBOSE: %s", cmd)
	}
}

func TestBuildSSHCommandQuotesForwardsBindAddressEscapeCharAndX11Trusted(t *testing.T) {
	cmd := BuildSSHCommand(domain.Server{
		Alias:             "ipv6-box",
		Host:              "10.0.0.1",
		BindAddress:       "*",
		LocalForward:      []string{"[::1]:8080:[2001:db8::1]:443", "*:9000:localhost:9000"},
		RemoteForward:     []string{"*:80:localhost:8080"},
		DynamicForward:    []string{"[::1]:1080"},
		EscapeChar:        "~",
		ForwardX11Trusted: "yes",
	})

	for _, want := range []string{
		"-b '*'",
		"-L '[::1]:8080:[2001:db8::1]:443'",
		"-L '*:9000:localhost:9000'",
		"-R '*:80:localhost:8080'",
		"-D '[::1]:1080'",
		"-e '~'",
		"-o ForwardX11Trusted=yes",
	} {
		if !strings.Contains(cmd, want) {
			t.Fatalf("BuildSSHCommand missing %q in: %s", want, cmd)
		}
	}
	if strings.Contains(cmd, " -Y ") {
		t.Fatalf("ForwardX11Trusted alone should not enable -Y: %s", cmd)
	}

	parsed, err := ParseSSHCommand(cmd)
	if err != nil {
		t.Fatalf("ParseSSHCommand round-trip failed: %v\ncmd: %s", err, cmd)
	}
	if parsed.BindAddress != "*" || parsed.EscapeChar != "~" || len(parsed.LocalForward) != 2 || len(parsed.DynamicForward) != 1 {
		t.Fatalf("round-trip parsed unexpected fields: %+v", parsed)
	}

	caretCmd := BuildSSHCommand(domain.Server{
		Alias:             "x11-box",
		Host:              "10.0.0.1",
		EscapeChar:        "^A",
		ForwardX11:        "yes",
		ForwardX11Trusted: "yes",
	})
	if !strings.Contains(caretCmd, "-e '^A'") || !strings.Contains(caretCmd, " -Y ") {
		t.Fatalf("expected quoted caret escape char and -Y: %s", caretCmd)
	}
}

func TestViewsEscapeTviewBrackets(t *testing.T) {
	details := NewServerDetails()
	details.UpdateServer(domain.Server{
		Alias:         "ipv6-box",
		Aliases:       []string{"ipv6-box"},
		Host:          "2001:db8::1",
		LocalForward:  []string{"[::1]:8080:[2001:db8::1]:80"},
		RemoteCommand: "[ -f /tmp/a ] && echo ok",
		Tags:          []string{"[prod]"},
		Readonly:      true,
	})
	plainDetails := details.GetText(true)
	for _, want := range []string{"[::1]:8080:[2001:db8::1]:80", "[ -f /tmp/a ]", "[prod]", "no (read-only)"} {
		if !strings.Contains(plainDetails, want) {
			t.Fatalf("details stripped or altered %q:\n%s", want, plainDetails)
		}
	}

	sessions := NewActiveSessions()
	sessions.SetSessions([]domain.ActiveSession{
		{PID: 100, Alias: "ipv6-box", Forwards: []string{"[::1]:1080"}},
	})
	if plain := sessions.GetText(true); !strings.Contains(plain, "[::1]:1080") {
		t.Fatalf("active sessions stripped bracketed IPv6 forward:\n%s", plain)
	}
}

func TestInitialFilterSetsListTitleAndDetailsScrollsWithJK(t *testing.T) {
	s := &initialFilterService{}
	view := NewTUI(zap.NewNop().Sugar(), s, "test", "test", Options{InitialFilter: "prod"}).(*tui)
	view.initializeTheme().buildComponents().buildLayout().bindEvents()
	view.loadInitialData()

	if !strings.Contains(view.serverList.GetTitle(), "Sort:") {
		t.Fatalf("initial filter list title missing sort indicator: %q", view.serverList.GetTitle())
	}

	screen := tcell.NewSimulationScreen("")
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()
	screen.SetSize(80, 20)
	view.details.SetRect(0, 0, 40, 10)
	view.details.SetText(strings.Repeat("line\n", 50))
	view.details.Draw(screen)
	view.app.SetFocus(view.details)
	view.handleNavigateDown()
	view.details.Draw(screen)
	row, _ := view.details.GetScrollOffset()
	if row != 1 {
		t.Fatalf("details row after j = %d, want 1", row)
	}
	view.handleNavigateUp()
	view.details.Draw(screen)
	row, _ = view.details.GetScrollOffset()
	if row != 0 {
		t.Fatalf("details row after k = %d, want 0", row)
	}
}

func TestParseSSHCommandDeprecatedAliasesAndWindowsQuotes(t *testing.T) {
	server, err := ParseSSHCommand(`ssh -o ChallengeResponseAuthentication=no -o HostbasedKeyTypes=ssh-ed25519 example`)
	if err != nil {
		t.Fatalf("ParseSSHCommand deprecated aliases failed: %v", err)
	}
	if server.KbdInteractiveAuthentication != "no" || server.HostbasedAcceptedAlgorithms != "ssh-ed25519" {
		t.Fatalf("deprecated aliases not mapped: %+v", server)
	}

	args, err := splitSSHCommandForPlatform(`ssh -o RemoteCommand="echo \"hi\"" host`, true)
	if err != nil {
		t.Fatalf("Windows escaped quote failed: %v", err)
	}
	if len(args) != 4 || args[2] != `RemoteCommand=echo "hi"` {
		t.Fatalf("unexpected Windows args: %q", args)
	}
}

func TestSCPProtocolUsageMultipleFlagGroups(t *testing.T) {
	proto, err := scpProtocolFromUsage("usage: scp [-346ABCpqrTv] [-s] [-F ssh_config] source ... target")
	if err != nil || proto != SCPOverSFTP {
		t.Fatalf("scpProtocolFromUsage = %v, %v; want SCPOverSFTP", proto, err)
	}
}
