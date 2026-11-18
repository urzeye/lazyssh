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
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/urzeye/lazyssh/internal/adapters/data/ssh_config_file"
	"github.com/urzeye/lazyssh/internal/core/domain"
	"go.uber.org/zap"
)

func TestParseSSHCommand(t *testing.T) {
	server, err := ParseSSHCommand(`ssh -p2222 -i "C:\Users\test user\.ssh\id key" -J jump -L 8080:localhost:80 -o IdentitiesOnly=yes user@192.0.2.1`)
	if err != nil {
		t.Fatal(err)
	}
	if server.Port != 2222 || server.User != "user" || server.Host != "192.0.2.1" || server.ProxyJump != "jump" || server.IdentitiesOnly != "yes" || len(server.LocalForward) != 1 || server.IdentityFiles[0] != `C:\Users\test user\.ssh\id key` {
		t.Fatalf("parsed wrong: %+v", server)
	}
	for _, text := range []string{`ssh -o 'LocalForward=8080 localhost:80' -o 'ProxyCommand=nc -X connect %h %p' example`, `ssh -AXCtt -6 -vvv 'ssh://user@[2001:db8::1]:2222'`, `ssh user@example -p 2222`, `ssh -o User=explicit -o User=ignored example`} {
		if _, err := ParseSSHCommand(text); err != nil {
			t.Errorf("%s: %v", text, err)
		}
	}
	server, err = ParseSSHCommand("ssh \\\n-p 2222 example")
	if err != nil || server.Port != 2222 {
		t.Fatalf("continuation: %+v %v", server, err)
	}
}

func TestImportedProxyDirectiveOrder(t *testing.T) {
	for _, tc := range []struct{ command, jump, proxy string }{
		{`ssh -o ProxyJump=bastion -o 'ProxyCommand=nc %h %p' host`, "bastion", ""},
		{`ssh -o 'ProxyCommand=nc %h %p' -o ProxyJump=bastion host`, "", "nc %h %p"},
		{`ssh -o ProxyCommand=none -o ProxyJump=bastion host`, "", "none"},
		{`ssh -o ProxyJump=none -o ProxyCommand=netcat host`, "none", ""},
		{`ssh -o ProxyJump=none -o ProxyJump=bastion host`, "none", ""},
	} {
		server, err := ParseSSHCommand(tc.command)
		if err != nil || server.ProxyJump != tc.jump || server.ProxyCommand != tc.proxy {
			t.Errorf("%s: jump=%q proxy=%q %v", tc.command, server.ProxyJump, server.ProxyCommand, err)
		}
	}
}

func TestImportedProxyMatchesOpenSSH(t *testing.T) {
	ssh, err := exec.LookPath("ssh")
	if err != nil {
		t.Skip("OpenSSH unavailable")
	}
	summary := func(output []byte) string {
		values := make(map[string]string)
		for _, line := range strings.Split(string(output), "\n") {
			parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
			if len(parts) == 2 {
				values[parts[0]] = parts[1]
			}
		}
		if jump := values["proxyjump"]; jump != "" && jump != "none" {
			return "jump:" + jump
		}
		if command := values["proxycommand"]; command != "" && command != "none" {
			return "command:" + command
		}
		return "direct"
	}
	for _, command := range []string{`ssh -o ProxyJump=bastion -o 'ProxyCommand=nc %h %p' host`, `ssh -o 'ProxyCommand=nc %h %p' -o ProxyJump=bastion host`, `ssh -o ProxyCommand=none -o ProxyJump=bastion host`, `ssh -o ProxyJump=none -o ProxyCommand=netcat host`, `ssh -o ProxyJump=none -o ProxyJump=bastion host`} {
		dir := t.TempDir()
		original := filepath.Join(dir, "original")
		saved := filepath.Join(dir, "saved")
		for _, path := range []string{original, saved} {
			if err := os.WriteFile(path, nil, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		tokens, err := splitSSHCommand(command)
		if err != nil {
			t.Fatal(err)
		}
		before, err := exec.Command(ssh, append([]string{"-G", "-F", original}, tokens[1:]...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("original config: %v\n%s", err, before)
		}
		server, err := ParseSSHCommand(command)
		if err != nil {
			t.Fatal(err)
		}
		repo := ssh_config_file.NewRepository(zap.NewNop().Sugar(), saved, filepath.Join(dir, "metadata.json"))
		if err := repo.AddServer(server); err != nil {
			t.Fatal(err)
		}
		after, err := exec.Command(ssh, "-G", "-F", saved, server.Alias).CombinedOutput()
		if err != nil {
			t.Fatalf("saved config: %v\n%s", err, after)
		}
		if summary(before) != summary(after) {
			t.Fatalf("%s: proxy changed from %q to %q", command, summary(before), summary(after))
		}
	}
}

func TestImportedCommentsDoNotBecomeRemoteCommands(t *testing.T) {
	server, err := ParseSSHCommand("ssh user@host # production")
	if err != nil || server.User != "user" || server.Host != "host" || server.RemoteCommand != "" {
		t.Fatalf("comment imported as command: %+v %v", server, err)
	}
	for _, windows := range []bool{false, true} {
		for _, tc := range []struct {
			command string
			want    []string
		}{
			{"ssh host # production", []string{"ssh", "host"}},
			{`ssh host '#literal'`, []string{"ssh", "host", "#literal"}},
			{`ssh host "#literal"`, []string{"ssh", "host", "#literal"}},
			{`ssh host echo value#suffix # comment`, []string{"ssh", "host", "echo", "value#suffix"}},
		} {
			args, err := splitSSHCommandForPlatform(tc.command, windows)
			if err != nil || !slices.Equal(args, tc.want) {
				t.Errorf("%q Windows=%v: got %q %v", tc.command, windows, args, err)
			}
		}
		if _, err := splitSSHCommandForPlatform("ssh host # first\nssh other", windows); err == nil {
			t.Fatal("comment hid a second command")
		}
	}
	args, err := splitSSHCommandForPlatform(`ssh host \#literal`, false)
	if err != nil || !slices.Equal(args, []string{"ssh", "host", "#literal"}) {
		t.Fatalf("escaped hash lost: %q %v", args, err)
	}
}

func TestImportedDestinationOptionOrder(t *testing.T) {
	for _, tc := range []struct {
		command, user, host string
		port                int
	}{
		{"ssh alice@host -l bob", "alice", "host", 22},
		{"ssh -l before alice@host -l after", "before", "host", 22},
		{"ssh ssh://alice@host:2222 -p 3333", "alice", "host", 2222},
		{"ssh -p 3333 ssh://alice@host:2222 -p 4444", "alice", "host", 3333},
		{"ssh host -o HostName=127.0.0.1", "", "127.0.0.1", 22},
		{"ssh -o HostName=127.0.0.2 host", "", "127.0.0.2", 22},
	} {
		server, err := ParseSSHCommand(tc.command)
		if err != nil || server.User != tc.user || server.Host != tc.host || server.Port != tc.port {
			t.Errorf("%s: %+v %v", tc.command, server, err)
		}
	}
	for _, command := range []string{"ssh -- host uname", "ssh host -- uname", "ssh host --", "ssh -- host -literal-command"} {
		if _, err := ParseSSHCommand(command); err != nil {
			t.Errorf("%s: %v", command, err)
		}
	}
}

func TestImportedDestinationMatchesOpenSSH(t *testing.T) {
	ssh, err := exec.LookPath("ssh")
	if err != nil {
		t.Skip("OpenSSH unavailable")
	}
	config := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(config, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"ssh alice@host -l bob", "ssh -l before alice@host -l after", "ssh ssh://alice@host:2222 -p 3333", "ssh -p 3333 ssh://alice@host:2222 -p 4444", "ssh host -o HostName=127.0.0.1"} {
		server, err := ParseSSHCommand(command)
		if err != nil {
			t.Fatal(err)
		}
		tokens, err := splitSSHCommand(command)
		if err != nil {
			t.Fatal(err)
		}
		args := append([]string{"-G", "-F", config}, tokens[1:]...)
		output, err := exec.Command(ssh, args...).CombinedOutput()
		if err != nil {
			t.Fatalf("OpenSSH %s: %v\n%s", command, err, output)
		}
		values := make(map[string]string)
		for _, line := range strings.Split(string(output), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 {
				values[fields[0]] = fields[1]
			}
		}
		if server.Host != values["hostname"] || strconv.Itoa(server.Port) != values["port"] || server.User != "" && server.User != values["user"] {
			t.Fatalf("%s: imported user/host/port %q %q %d differs from OpenSSH %q %q %q", command, server.User, server.Host, server.Port, values["user"], values["hostname"], values["port"])
		}
	}
}

func TestSSHCommandLexingPreservesPlatformPaths(t *testing.T) {
	for _, tc := range []struct {
		command string
		windows bool
		want    []string
	}{
		{`ssh -i C:\keys\id_ed25519 user@host`, true, []string{"ssh", "-i", `C:\keys\id_ed25519`, "user@host"}},
		{`ssh -i "C:\test user\id_ed25519" host`, true, []string{"ssh", "-i", `C:\test user\id_ed25519`, "host"}},
		{`ssh -i \\server\share\id_ed25519 host`, true, []string{"ssh", "-i", `\\server\share\id_ed25519`, "host"}},
		{`ssh -i 'C:\keys\O''Brien\id' host`, true, []string{"ssh", "-i", `C:\keys\O'Brien\id`, "host"}},
		{`ssh -i ~/.ssh/test\ key host`, false, []string{"ssh", "-i", "~/.ssh/test key", "host"}},
		{`ssh -o 'ProxyCommand=printf \\value' host`, false, []string{"ssh", "-o", `ProxyCommand=printf \\value`, "host"}},
	} {
		got, err := splitSSHCommandForPlatform(tc.command, tc.windows)
		if err != nil || !slices.Equal(got, tc.want) {
			t.Errorf("%q Windows=%v: got %q, %v, want %q", tc.command, tc.windows, got, err, tc.want)
		}
	}
}

func TestImportedRemoteCommandPercentSemantics(t *testing.T) {
	for _, tc := range []struct{ command, want string }{
		{`ssh host 'date +%Y'`, "date +%%Y"},
		{`ssh -- host 'echo %h 100%'`, "echo %%h 100%%"},
		{`ssh -o 'RemoteCommand=echo %h' host`, "echo %h"},
		{`ssh -o 'RemoteCommand=echo %h' -- host`, "echo %h"},
		{`ssh -o RemoteCommand=none host 'date +%Y'`, "date +%%Y"},
	} {
		server, err := ParseSSHCommand(tc.command)
		if err != nil || server.RemoteCommand != tc.want {
			t.Errorf("%s: got %q, %v, want %q", tc.command, server.RemoteCommand, err, tc.want)
		}
	}
	if _, err := ParseSSHCommand(`ssh -o RemoteCommand=first host second`); err == nil {
		t.Fatal("conflicting remote commands accepted")
	}
}

func TestImportedRemoteCommandSurvivesFormAndConfig(t *testing.T) {
	server, err := ParseSSHCommand(`ssh host 'date +%Y'`)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	config := filepath.Join(dir, "config")
	if err := os.WriteFile(config, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	repo := ssh_config_file.NewRepository(zap.NewNop().Sugar(), config, filepath.Join(dir, "metadata.json"))
	form := NewServerForm(ServerFormAdd, &server).SetVersionInfo("test", "test")
	var saveErr error
	form.OnSave(func(saved domain.Server, original *domain.Server) {
		if original != nil {
			t.Fatal("import treated as update")
		}
		saveErr = repo.AddServer(saved)
	})
	if !form.handleSave() || saveErr != nil {
		t.Fatalf("save imported command: %v", saveErr)
	}
	servers, err := repo.ListServers("")
	if err != nil || len(servers) != 1 || servers[0].RemoteCommand != "date +%%Y" {
		t.Fatalf("percent escape lost: %+v %v", servers, err)
	}
}

func TestParseSSHCommandRejectsAmbiguity(t *testing.T) {
	for _, text := range []string{"", "scp host", "ssh", "ssh -p", "ssh -p 0 host", "ssh -F other.conf host", "ssh -o Unknown=ignored host", "ssh 'unclosed", "ssh host; rm file", "ssh host\nother", "ssh ssh://user:password@host", "ssh -o Alias=other host"} {
		if _, err := ParseSSHCommand(text); err == nil {
			t.Errorf("accepted %q", text)
		}
	}
}

func TestPastedCommandPreservesProxyAndRemoteCommand(t *testing.T) {
	server, err := ParseSSHCommand(`ssh -o 'ProxyCommand=nc %h %p' -i first -i second example uname -a`)
	if err != nil {
		t.Fatal(err)
	}
	if server.RemoteCommand != "uname -a" || !strings.Contains(server.ProxyCommand, "%h") || len(server.IdentityFiles) != 2 {
		t.Fatalf("lost settings: %+v", server)
	}
}

func TestPastedTrustedX11EnablesForwarding(t *testing.T) {
	server, err := ParseSSHCommand("ssh -Y example")
	if err != nil || server.ForwardX11 != "yes" || server.ForwardX11Trusted != "yes" {
		t.Fatalf("-Y lost X11 forwarding: %+v %v", server, err)
	}
}

func TestPastedFlagsPreserveOpenSSHPrecedence(t *testing.T) {
	server, err := ParseSSHCommand("ssh -A -a -X -x -T -t -p 2222 -p 3333 example")
	if err != nil {
		t.Fatal(err)
	}
	if server.ForwardAgent != "no" || server.ForwardX11 != "no" || server.RequestTTY != "yes" || server.Port != 2222 {
		t.Fatalf("flag precedence changed: %+v", server)
	}
}
