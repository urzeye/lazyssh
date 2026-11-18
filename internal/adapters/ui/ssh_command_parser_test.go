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
	"testing"
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
