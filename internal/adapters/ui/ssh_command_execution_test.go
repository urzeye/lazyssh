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
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/urzeye/lazyssh/internal/adapters/data/ssh_config_file"
	"github.com/urzeye/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

func TestImportedCommandExecRequestSurvivesCopyAndReimport(t *testing.T) {
	if _, err := exec.LookPath("ssh"); err != nil {
		t.Skip("OpenSSH unavailable")
	}
	if output, err := exec.Command("python", "-W", "ignore", "-c", "import paramiko").CombinedOutput(); err != nil {
		t.Skipf("optional loopback SSH fixture requires Python paramiko: %v: %s", err, output)
	}
	for _, tc := range []struct{ input, want string }{
		{`ssh host 'date +%Y'`, "date +%Y"},
		{`ssh host 'echo %h %'`, "echo %h %"},
		{`ssh -o 'RemoteCommand=echo %h %%' host`, "echo 127.0.0.1 %"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			server, err := ParseSSHCommand(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			server.Host, server.Alias = "127.0.0.1", "imported"
			dir := t.TempDir()
			config := filepath.Join(dir, "config")
			if err := os.WriteFile(config, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			repo := ssh_config_file.NewRepository(zap.NewNop().Sugar(), config, filepath.Join(dir, "metadata.json"))
			form := NewServerForm(ServerFormAdd, &server).SetVersionInfo("test", "test")
			var saveErr error
			form.OnSave(func(saved domain.Server, _ *domain.Server) { saveErr = repo.AddServer(saved) })
			if !form.handleSave() || saveErr != nil {
				t.Fatalf("save imported form: %v", saveErr)
			}
			servers, err := repo.ListServers("")
			if err != nil || len(servers) != 1 {
				t.Fatalf("reload imported config: %+v, %v", servers, err)
			}
			copied := BuildSSHCommand(servers[0])
			assertSSHExecRequest(t, config, "", server.Alias, tc.want)
			assertSSHExecRequest(t, config, copied, "", tc.want)
			reimported, err := ParseSSHCommand(copied)
			if err != nil {
				t.Fatal(err)
			}
			reimported.Alias = "reimported"
			if err := repo.AddServer(reimported); err != nil {
				t.Fatal(err)
			}
			assertSSHExecRequest(t, config, "", reimported.Alias, tc.want)
		})
	}
}

func assertSSHExecRequest(t *testing.T, config, copied, alias, want string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	fixture := exec.CommandContext(ctx, "python", "-W", "ignore", "testdata/ssh_command_server.py")
	var fixtureErrors bytes.Buffer
	fixture.Stderr = &fixtureErrors
	stdout, err := fixture.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		_ = fixture.Wait()
	}()
	port, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatalf("start loopback fixture: %v: %s", err, fixtureErrors.String())
	}
	knownHosts := strings.ReplaceAll(filepath.Join(filepath.Dir(config), "known_hosts"), "%", "%%")
	options := []string{"-F", config, "-p", strings.TrimSpace(port), "-T", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=" + knownHosts, "-o", "BatchMode=yes", "-o", "HostKeyAlgorithms=ssh-rsa", "-o", "LogLevel=ERROR", "-o", "ConnectTimeout=5"}
	var command *exec.Cmd
	if copied == "" {
		command = exec.CommandContext(ctx, "ssh", append(options, alias)...)
	} else {
		quoted := make([]string, 0, len(options))
		for _, option := range options {
			quoted = append(quoted, shellQuote(option))
		}
		invocation := strings.Replace(copied, "ssh ", "ssh "+strings.Join(quoted, " ")+" ", 1)
		if runtime.GOOS == windowsOS {
			command = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", invocation)
		} else {
			command = exec.CommandContext(ctx, "sh", "-c", invocation)
		}
	}
	var diagnostics bytes.Buffer
	command.Stderr = &diagnostics
	output, err := command.Output()
	if err != nil || string(output) != want {
		t.Fatalf("SSH exec request = %q, want %q; %v\n%s", output, want, err, diagnostics.String())
	}
}
