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
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/rivo/tview"
)

const windowsOS = "windows"

func shellQuote(value string) string {
	if runtime.GOOS == windowsOS {
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
	if localPath == "~" || strings.HasPrefix(localPath, "~/") || strings.HasPrefix(localPath, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if localPath == "~" {
			localPath = home
		} else {
			localPath = filepath.Join(home, localPath[2:])
		}
	}
	// A relative local filename containing ':' otherwise looks like a remote target.
	if runtime.GOOS != windowsOS && !strings.HasPrefix(localPath, "/") && strings.Contains(localPath, ":") {
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
