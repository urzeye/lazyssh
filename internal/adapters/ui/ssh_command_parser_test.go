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
