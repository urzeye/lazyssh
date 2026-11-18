package ui

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/rivo/tview"
)

func shellQuote(value string) string {
	if runtime.GOOS == "windows" {
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func BuildSCPCommand(alias, configPath, localPath, remotePath string, upload, recursive bool) (string, error) {
	for _, field := range []struct{ name, value string }{{"alias", alias}, {"SSH config", configPath}, {"local path", localPath}, {"remote path", remotePath}} {
		if strings.TrimSpace(field.value) == "" || strings.ContainsAny(field.value, "\x00\r\n") {
			return "", fmt.Errorf("%s must be nonempty and contain no line breaks", field.name)
		}
	}
	if strings.HasPrefix(alias, "-") || strings.ContainsAny(alias, ":@") {
		return "", fmt.Errorf("SCP requires a configured host alias")
	}
	// A relative local filename containing ':' otherwise looks like a remote target.
	if runtime.GOOS != "windows" && !strings.HasPrefix(localPath, "/") && strings.Contains(localPath, ":") {
		localPath = "./" + localPath
	}
	parts := []string{"scp", "-F", shellQuote(configPath)}
	if recursive {
		parts = append(parts, "-r")
	}
	parts = append(parts, "--")
	local, remote := shellQuote(localPath), shellQuote(alias+":"+remotePath)
	if upload {
		parts = append(parts, local, remote)
	} else {
		parts = append(parts, remote, local)
	}
	return strings.Join(parts, " "), nil
}

func (t *tui) handleSCP() {
	server, ok := t.serverList.GetSelectedServer()
	if !ok {
		return
	}
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(" SCP command ")
	local := tview.NewInputField().SetLabel("Local path:").SetFieldWidth(50)
	remote := tview.NewInputField().SetLabel("Remote path:").SetText(".").SetFieldWidth(50)
	upload, recursive := true, false
	preview := tview.NewTextView().SetWrap(true)
	build := func() (string, error) {
		return BuildSCPCommand(server.Alias, t.serverService.GetConfigFile(), local.GetText(), remote.GetText(), upload, recursive)
	}
	update := func() {
		command, err := build()
		if err != nil {
			preview.SetText(err.Error())
		} else {
			preview.SetText(command)
		}
	}
	local.SetChangedFunc(func(string) { update() })
	remote.SetChangedFunc(func(string) { update() })
	form.AddDropDown("Direction:", []string{"Upload", "Download"}, 0, func(_ string, index int) { upload = index == 0; update() })
	form.AddFormItem(local).AddFormItem(remote)
	form.AddCheckbox("Directory (recursive):", false, func(checked bool) { recursive = checked; update() })
	form.AddButton("Copy command", func() {
		command, err := build()
		if err == nil {
			err = clipboard.WriteAll(command)
		}
		if err != nil {
			modal := tview.NewModal().SetText(tview.Escape(err.Error())).AddButtons([]string{"Close"}).SetDoneFunc(func(_ int, _ string) { t.app.SetRoot(layoutForSCP(form, preview), true) })
			t.app.SetRoot(modal, true)
			return
		}
		t.returnToMain()
		t.showStatusTemp("SCP command copied")
	})
	form.AddButton("Cancel", t.returnToMain).SetCancelFunc(t.returnToMain)
	update()
	t.app.SetRoot(layoutForSCP(form, preview), true)
}

func layoutForSCP(form *tview.Form, preview *tview.TextView) *tview.Flex {
	return tview.NewFlex().SetDirection(tview.FlexRow).AddItem(form, 0, 1, true).AddItem(preview, 4, 0, false)
}
