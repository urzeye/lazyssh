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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSCPCommandPreservesAliasAndConfig(t *testing.T) {
	for _, upload := range []bool{true, false} {
		command, err := BuildSCPCommand("prod", "/config dir/custom ssh", "/local dir/file's name", "/remote dir/file", upload, true, SCPOverSFTP)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(command, "scp -s -F "+shellQuote("/config dir/custom ssh")) || !strings.Contains(command, " -r -- ") || !strings.Contains(command, shellQuote("prod:/remote dir/file")) {
			t.Fatalf("bad command: %s", command)
		}
		localPos, remotePos := strings.Index(command, shellQuote("/local dir/file's name")), strings.Index(command, shellQuote("prod:/remote dir/file"))
		if (localPos < remotePos) != upload {
			t.Fatalf("wrong direction: %s", command)
		}
	}
}

func TestSCPProtocolCapabilities(t *testing.T) {
	for _, tc := range []struct {
		usage string
		want  SCPProtocol
	}{
		{"usage: scp [-346ABCpqrTv] [-F ssh_config] source ... target", SCPLegacy},
		{"usage: scp [-346ABCOpqRrsTv] [-F ssh_config] source ... target", SCPOverSFTP},
	} {
		got, err := scpProtocolFromUsage(tc.usage)
		if err != nil || got != tc.want {
			t.Fatalf("protocol from %q = %v, %v", tc.usage, got, err)
		}
	}
	if _, err := scpProtocolFromUsage("unrecognized client output"); err == nil {
		t.Fatal("unknown SCP capabilities accepted")
	}
}

func TestLegacyRemotePathsStayLiteral(t *testing.T) {
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("POSIX shell unavailable")
	}
	for _, remote := range []string{"/tmp/a b", "/tmp/O'Brien", `/tmp/a"b`, "/tmp/file;printf unexpected", "/tmp/$(printf unexpected)", "/tmp/`printf unexpected`", "~/folder name", "~"} {
		command := exec.Command(shell, "-c", "printf '%s' "+quoteLegacyRemotePath(remote))
		output, err := command.Output()
		if err != nil {
			t.Fatal(err)
		}
		want := remote
		if strings.HasPrefix(remote, "~/") || remote == "~" {
			homeCmd := exec.Command(shell, "-c", "printf '%s' ~")
			home, homeErr := homeCmd.Output()
			if homeErr != nil {
				t.Fatal(homeErr)
			}
			want = string(home) + strings.TrimPrefix(remote, "~")
		}
		if string(output) != want {
			t.Fatalf("remote path %q interpreted as %q, want %q", remote, output, want)
		}
	}
}

func TestSCPExpandsLocalHomeWithoutExpandingRemoteHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	command, err := BuildSCPCommand("prod", "config", "~/file", "~/remote", true, false, SCPOverSFTP)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(command, shellQuote(filepath.Join(home, "file"))) || !strings.Contains(command, shellQuote("prod:~/remote")) {
		t.Fatalf("wrong home expansion: %s", command)
	}
}

func TestSCPRejectsMissingPaths(t *testing.T) {
	for _, args := range [][4]string{{"prod", "config", "", "."}, {"prod", "config", "file", "\n"}, {"-bad", "config", "file", "."}} {
		if _, err := BuildSCPCommand(args[0], args[1], args[2], args[3], true, false, SCPOverSFTP); err == nil {
			t.Fatalf("accepted %q", args)
		}
	}
}
