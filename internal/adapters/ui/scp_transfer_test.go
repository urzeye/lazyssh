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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLegacySCPReceiverChecksRequestedFilename(t *testing.T) {
	scp, err := exec.LookPath("scp")
	if err != nil {
		t.Skip("OpenSSH SCP unavailable")
	}
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("POSIX shell unavailable for SCP transport test")
	}
	helper := filepath.Join(t.TempDir(), "transport")
	if runtime.GOOS == windowsOS {
		helper += ".exe"
	}
	if output, err := exec.Command("go", "build", "-o", helper, "./testdata/scp_transport").CombinedOutput(); err != nil {
		t.Fatalf("build transport: %v\n%s", err, output)
	}
	clients := []string{scp}
	if runtime.GOOS == windowsOS {
		native := filepath.Join(os.Getenv("WINDIR"), "System32", "OpenSSH", "scp.exe")
		if _, err := os.Stat(native); err == nil && !strings.EqualFold(scp, native) {
			clients = append(clients, native)
		}
	}
	for _, scp := range clients {
		usage, _ := exec.Command(scp, "-h").CombinedOutput()
		protocol, err := scpProtocolFromUsage(string(usage))
		if err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS == windowsOS && protocol == SCPOverSFTP {
			// Native Win32 legacy SCP has different fnmatch rules. The product
			// selects SFTP for this client; do not force an unused legacy mode.
			t.Logf("%s selects SFTP; legacy receiver cases do not apply", scp)
			continue
		}
		legacyFlags := []string{}
		if strings.Contains(string(usage), "O") {
			legacyFlags = append(legacyFlags, "-O")
		}
		transfers := t.TempDir()
		for _, name := range []string{"file", "a b", "O'Brien", `a"b`, "a[b]", "a{b,c}", "a*b?", `a\b`, "a;$(echo unexpected)", "a`b", "unrequested-file"} {
			for _, viaPowerShell := range []bool{false, true} {
				if viaPowerShell && runtime.GOOS != windowsOS {
					continue
				}
				t.Run(fmt.Sprintf("%s/%s/powershell=%t", scp, name, viaPowerShell), func(t *testing.T) {
					// A fixed local filename also exercises names Windows cannot create.
					dir, err := os.MkdirTemp(transfers, "transfer-")
					if err != nil {
						t.Fatal(err)
					}
					destination := filepath.Join(dir, "received")
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					requested := name
					if name == "unrequested-file" {
						requested = "expected-file"
					}
					args := append([]string{}, legacyFlags...)
					args = append(args, "-S", filepath.ToSlash(helper), "-F", os.DevNull, "--", "prod:"+quoteLegacyRemotePath("/tmp/"+requested), filepath.ToSlash(destination))
					// A script file avoids Windows/Cygwin argument re-parsing at the
					// launcher boundary. SCP itself still performs the real receive check.
					words := []string{scp}
					words = append(words, args...)
					for i, word := range words {
						words[i] = "'" + strings.ReplaceAll(word, "'", "'\"'\"'") + "'"
					}
					script := filepath.Join(dir, "receive.sh")
					if err := os.WriteFile(script, []byte("exec "+strings.Join(words, " ")+"\n"), 0o600); err != nil {
						t.Fatal(err)
					}
					cmd := exec.CommandContext(ctx, sh, filepath.ToSlash(script))
					if viaPowerShell {
						command := formatNativeCommand(filepath.ToSlash(scp), args)
						cmd = exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", command+"; exit $LASTEXITCODE")
					}
					cmd.Env = append(os.Environ(), "LAZYSSH_SCP_FILENAME="+name)
					output, err := cmd.CombinedOutput()
					if name == "unrequested-file" {
						if err == nil || !strings.Contains(string(output), "filename does not match request") {
							t.Fatalf("unrequested filename accepted: %v\n%s", err, output)
						}
						if _, err := os.Stat(destination); !os.IsNotExist(err) {
							t.Fatalf("unrequested file was written: %v", err)
						}
						return
					}
					if err != nil {
						t.Fatalf("real SCP receiver: %v\n%s", err, output)
					}
					contents, err := os.ReadFile(destination)
					if err != nil || string(contents) != "SCP regression payload\n" {
						t.Fatalf("received contents = %q, %v\n%s", contents, err, output)
					}
				})
			}
		}
	}
}
