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
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/rivo/tview"
)

const windowsOS = "windows"

type SCPProtocol int

const (
	SCPOverSFTP SCPProtocol = iota
	SCPLegacy
)

var (
	scpUsageFlags    = regexp.MustCompile(`\[-([A-Za-z0-9]+)\]`)
	remoteHomePrefix = regexp.MustCompile(`^~[A-Za-z0-9_.-]*(/|$)`)
)

func detectSCPProtocol() (SCPProtocol, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "scp", "-h").CombinedOutput()
	if ctx.Err() != nil {
		return 0, fmt.Errorf("inspect scp: %w", ctx.Err())
	}
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return 0, fmt.Errorf("inspect scp: %w", err)
	}
	return scpProtocolFromUsage(string(output))
}

func scpProtocolFromUsage(usage string) (SCPProtocol, error) {
	flags := scpUsageFlags.FindStringSubmatch(usage)
	if len(flags) != 2 {
		return 0, fmt.Errorf("could not determine supported scp protocol from its usage output")
	}
	if strings.ContainsRune(flags[1], 's') {
		return SCPOverSFTP, nil
	}
	return SCPLegacy, nil
}

func quoteLegacyRemotePath(value string) string {
	// Tilde expansion belongs to the remote shell; all path characters after it
	// remain literal, including spaces, apostrophes and shell metacharacters.
	prefix := remoteHomePrefix.FindString(value)
	value = strings.TrimPrefix(value, prefix)
	if value == "" && prefix != "" {
		return prefix
	}
	return prefix + "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func shellQuote(value string) string {
	if runtime.GOOS == windowsOS {
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func BuildSCPCommand(alias, configPath, localPath, remotePath string, upload, recursive bool, protocol SCPProtocol) (string, error) {
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
	parts := []string{"scp"}
	switch protocol {
	case SCPOverSFTP:
		parts = append(parts, "-s")
	case SCPLegacy:
		remotePath = quoteLegacyRemotePath(remotePath)
	default:
		return "", fmt.Errorf("unsupported SCP protocol")
	}
	parts = append(parts, "-F", shellQuote(configPath))
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
	protocol, err := detectSCPProtocol()
	if err != nil {
		t.showActionError("Inspect SCP client", err)
		return
	}
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(" SCP command ")
	local := tview.NewInputField().SetLabel("Local path:").SetFieldWidth(50)
	remote := tview.NewInputField().SetLabel("Remote path:").SetText(".").SetFieldWidth(50)
	upload, recursive := true, false
	preview := tview.NewTextView().SetWrap(true)
	build := func() (string, error) {
		return BuildSCPCommand(server.Alias, t.serverService.GetConfigFile(), local.GetText(), remote.GetText(), upload, recursive, protocol)
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
